package handlers

import (
	"encoding/json"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Status      taskdomain.Status    `json:"status"`
	RecurrenceType   *string         `json:"recurrence_type,omitempty"`
	RecurrenceConfig json.RawMessage `json:"recurrence_config,omitempty"`
}

type taskDTO struct {
	ID          int64                `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Status      taskdomain.Status    `json:"status"`

	RecurrenceType   *string         `json:"recurrence_type,omitempty"`
	RecurrenceConfig json.RawMessage `json:"recurrence_config,omitempty"`
	NextRunDate      *time.Time      `json:"next_run_date,omitempty"`

	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		RecurrenceType: task.RecurrenceType,
		RecurrenceConfig: task.RecurrenceConfig,
		NextRunDate: task.NextRunDate,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}
