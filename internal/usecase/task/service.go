package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo   Repository
	now    func() time.Time
	logger *slog.Logger
}

const (
	ListFilterAll       = "all"
	ListFilterTemplates = "templates"
	ListFilterInstances = "instances"
)

func NewService(repo Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		now:    func() time.Time { return time.Now().UTC() },
		logger: logger,
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:            normalized.Title,
		Description:      normalized.Description,
		Status:           normalized.Status,
		RecurrenceType:   normalized.RecurrenceType,
		RecurrenceConfig: normalized.RecurrenceConfig,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now
	if normalized.RecurrenceType != nil {
		nextRunDate := now
		model.NextRunDate = &nextRunDate
	}

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, filter string) ([]taskdomain.Task, error) {
	if filter == "" {
		filter = ListFilterAll
	}

	switch filter {
	case ListFilterAll, ListFilterTemplates, ListFilterInstances:
		return s.repo.List(ctx, filter)
	default:
		return nil, fmt.Errorf("%w: invalid list filter", ErrInvalidInput)
	}
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	recurrenceType, err := normalizeRecurrenceType(input.RecurrenceType)
	if err != nil {
		return CreateInput{}, err
	}
	input.RecurrenceType = recurrenceType

	recurrenceConfig := bytes.TrimSpace(input.RecurrenceConfig)
	if input.RecurrenceType == nil {
		if len(recurrenceConfig) > 0 {
			return CreateInput{}, fmt.Errorf("%w: recurrence_config requires recurrence_type", ErrInvalidInput)
		}

		input.RecurrenceConfig = nil
		return input, nil
	}

	if len(recurrenceConfig) == 0 {
		return CreateInput{}, fmt.Errorf("%w: recurrence_config is required for recurring tasks", ErrInvalidInput)
	}

	if !json.Valid(recurrenceConfig) {
		return CreateInput{}, fmt.Errorf("%w: recurrence_config must be valid json", ErrInvalidInput)
	}

	input.RecurrenceConfig = json.RawMessage(recurrenceConfig)

	return input, nil
}

func normalizeRecurrenceType(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(*raw))
	if normalized == "" {
		return nil, nil
	}

	switch normalized {
	case "daily", "monthly", "dates", "even_odd":
		return &normalized, nil
	default:
		return nil, fmt.Errorf("%w: unsupported recurrence_type", ErrInvalidInput)
	}
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}
