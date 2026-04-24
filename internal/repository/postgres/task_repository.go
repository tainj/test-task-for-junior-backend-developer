package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func New(pool *pgxpool.Pool, logger *slog.Logger) *Repository {
	return &Repository{pool: pool, logger: logger}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, recurrence_type, recurrence_config, next_run_date, parent_task_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, title, description, status, recurrence_type, recurrence_config, next_run_date, parent_task_id, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status,
		task.RecurrenceType, task.RecurrenceConfig, task.NextRunDate, task.ParentTaskID,
		task.CreatedAt, task.UpdatedAt,
	)
	return scanTask(row)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, recurrence_type, recurrence_config, next_run_date, parent_task_id, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			status = $3,
			updated_at = $4
		WHERE id = $5
		RETURNING id, title, description, status, recurrence_type, recurrence_config, next_run_date, parent_task_id, created_at, updated_at
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context, filter string) ([]taskdomain.Task, error) {
	query := `
		SELECT id, title, description, status, recurrence_type, recurrence_config, next_run_date, parent_task_id, created_at, updated_at
		FROM tasks
	`

	switch filter {
	case "templates":
		query += " WHERE parent_task_id IS NULL"
	case "instances":
		query += " WHERE parent_task_id IS NOT NULL"
	case "", "all":
		// no-op
	}

	query += " ORDER BY id DESC"

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task           taskdomain.Task
		status         string
		recurrenceType *string
		recConfigBytes []byte
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&recurrenceType,
		&recConfigBytes,
		&task.NextRunDate,
		&task.ParentTaskID,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)
	task.RecurrenceType = recurrenceType
	if recConfigBytes != nil {
		task.RecurrenceConfig = json.RawMessage(recConfigBytes)
	}
	return &task, nil
}

func (r *Repository) GetDueTemplates(ctx context.Context, asOf time.Time, limit int) ([]taskdomain.Task, error) {
	if limit <= 0 {
		limit = 100
	}

	const query = `
		SELECT id, title, description, status, recurrence_type, recurrence_config, next_run_date, parent_task_id, created_at, updated_at
		FROM tasks
		WHERE parent_task_id IS NULL 
		  AND next_run_date IS NOT NULL 
		  AND next_run_date <= $1
		ORDER BY next_run_date ASC, id ASC
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`
	rows, err := r.pool.Query(ctx, query, asOf, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []taskdomain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, rows.Err()
}

func (r *Repository) CreateInstanceAndAdvanceTemplate(ctx context.Context, templateID int64, asOf time.Time, nextRunDate *time.Time, now time.Time) (*taskdomain.Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const lockTemplateQuery = `
		SELECT id
		FROM tasks
		WHERE id = $1
		  AND parent_task_id IS NULL
		  AND next_run_date IS NOT NULL
		  AND next_run_date <= $2
		FOR UPDATE
	`

	var lockedID int64
	if err := tx.QueryRow(ctx, lockTemplateQuery, templateID, asOf).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	const createInstanceQuery = `
		INSERT INTO tasks (title, description, status, parent_task_id, created_at, updated_at)
		SELECT title, description, $2, id, $3, $3
		FROM tasks
		WHERE id = $1
		RETURNING id, title, description, status, recurrence_type, recurrence_config, next_run_date, parent_task_id, created_at, updated_at
	`

	createdRow := tx.QueryRow(ctx, createInstanceQuery, templateID, taskdomain.StatusNew, now)
	created, err := scanTask(createdRow)
	if err != nil {
		return nil, err
	}

	const advanceTemplateQuery = `
		UPDATE tasks
		SET next_run_date = $1,
			updated_at = $2
		WHERE id = $3
	`
	if _, err := tx.Exec(ctx, advanceTemplateQuery, nextRunDate, now, templateID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	tx = nil

	return created, nil
}
