package task

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID               int64           `json:"id"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Status           Status          `json:"status"`
	
	// Новые поля периодичности
	RecurrenceType   *string         `json:"recurrence_type,omitempty"`   // daily, monthly, dates, even_odd
	RecurrenceConfig json.RawMessage `json:"recurrence_config,omitempty"` // хранит сырой JSONB без base64-кодирования
	NextRunDate      *time.Time      `json:"next_run_date,omitempty"`     // дата следующего запуска шаблона
	ParentTaskID     *int64          `json:"-"`                           // ссылка на шаблон (NULL = шаблон, не NULL = экземпляр)
	
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}
