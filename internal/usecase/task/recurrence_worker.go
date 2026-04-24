package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

const defaultRecurringBatchSize = 100

func (s *Service) StartRecurringWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Info("recurring worker started", "interval", interval.String())

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("recurring worker stopped")
			return
		case <-ticker.C:
			if err := s.ProcessDueRecurringTemplates(ctx, defaultRecurringBatchSize); err != nil {
				s.logger.Error("recurring worker iteration failed", "error", err)
			}
		}
	}
}

func (s *Service) ProcessDueRecurringTemplates(ctx context.Context, limit int) error {
	asOf := s.now()
	templates, err := s.repo.GetDueTemplates(ctx, asOf, limit)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		s.logger.Debug("recurring worker: no due templates")
		return nil
	}

	s.logger.Debug("recurring worker: due templates found", "count", len(templates))

	for i := range templates {
		template := templates[i]
		if template.RecurrenceType == nil || template.NextRunDate == nil {
			s.logger.Debug("recurring worker: template skipped due to missing recurrence fields", "template_id", template.ID)
			continue
		}

		nextRunDate, err := calculateNextRunDate(*template.RecurrenceType, template.RecurrenceConfig, *template.NextRunDate)
		if err != nil {
			s.logger.Error("recurring worker: failed to calculate next run date", "template_id", template.ID, "error", err)
			continue
		}

		created, err := s.repo.CreateInstanceAndAdvanceTemplate(ctx, template.ID, asOf, nextRunDate, s.now())
		if err != nil {
			if errors.Is(err, taskdomain.ErrNotFound) {
				s.logger.Debug("recurring worker: template already processed by another worker", "template_id", template.ID)
				continue
			}

			s.logger.Error("recurring worker: failed to create task instance", "template_id", template.ID, "error", err)
			continue
		}

		s.logger.Info("recurring task instance created", "template_id", template.ID, "task_id", created.ID, "next_run_date", nextRunDate)
	}

	return nil
}

func calculateNextRunDate(recurrenceType string, recurrenceConfig json.RawMessage, currentDue time.Time) (*time.Time, error) {
	recurrenceType = strings.ToLower(strings.TrimSpace(recurrenceType))
	base := currentDue.UTC()

	switch recurrenceType {
	case "daily":
		var cfg struct {
			IntervalDays int `json:"interval_days"`
		}
		if len(recurrenceConfig) > 0 {
			if err := json.Unmarshal(recurrenceConfig, &cfg); err != nil {
				return nil, fmt.Errorf("invalid daily recurrence config: %w", err)
			}
		}
		if cfg.IntervalDays <= 0 {
			cfg.IntervalDays = 1
		}
		next := base.AddDate(0, 0, cfg.IntervalDays)
		return &next, nil
	case "monthly":
		var cfg struct {
			IntervalMonths int `json:"interval_months"`
			Day            *int `json:"day"`
		}
		if len(recurrenceConfig) > 0 {
			if err := json.Unmarshal(recurrenceConfig, &cfg); err != nil {
				return nil, fmt.Errorf("invalid monthly recurrence config: %w", err)
			}
		}
		if cfg.IntervalMonths <= 0 {
			cfg.IntervalMonths = 1
		}

		next := base.AddDate(0, cfg.IntervalMonths, 0)
		if cfg.Day != nil {
			if *cfg.Day < 1 || *cfg.Day > 31 {
				return nil, fmt.Errorf("invalid monthly recurrence day")
			}
			targetDay := *cfg.Day
			maxDay := daysInMonth(next.Year(), next.Month())
			if targetDay > maxDay {
				targetDay = maxDay
			}
			next = time.Date(next.Year(), next.Month(), targetDay, base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), time.UTC)
		}

		return &next, nil
	case "dates":
		var cfg struct {
			Dates []string `json:"dates"`
		}
		if err := json.Unmarshal(recurrenceConfig, &cfg); err != nil {
			return nil, fmt.Errorf("invalid dates recurrence config: %w", err)
		}
		if len(cfg.Dates) == 0 {
			return nil, fmt.Errorf("dates recurrence config must include dates")
		}

		nextDates := make([]time.Time, 0, len(cfg.Dates))
		for _, raw := range cfg.Dates {
			date, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
			if err != nil {
				return nil, fmt.Errorf("invalid date in dates recurrence config: %w", err)
			}
			nextDates = append(nextDates, date.UTC())
		}
		sort.Slice(nextDates, func(i, j int) bool {
			return nextDates[i].Before(nextDates[j])
		})

		for _, candidate := range nextDates {
			if candidate.After(base) {
				next := time.Date(candidate.Year(), candidate.Month(), candidate.Day(), base.Hour(), base.Minute(), base.Second(), base.Nanosecond(), time.UTC)
				return &next, nil
			}
		}

		return nil, nil
	case "even_odd":
		var cfg struct {
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(recurrenceConfig, &cfg); err != nil {
			return nil, fmt.Errorf("invalid even_odd recurrence config: %w", err)
		}
		mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
		if mode != "even" && mode != "odd" {
			return nil, fmt.Errorf("even_odd recurrence mode must be 'even' or 'odd'")
		}

		candidate := base.AddDate(0, 0, 1)
		for i := 0; i < 3; i++ {
			isEven := candidate.Day()%2 == 0
			if (mode == "even" && isEven) || (mode == "odd" && !isEven) {
				next := candidate
				return &next, nil
			}
			candidate = candidate.AddDate(0, 0, 1)
		}

		return nil, fmt.Errorf("failed to resolve next even_odd run date")
	default:
		return nil, fmt.Errorf("unsupported recurrence_type")
	}
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
