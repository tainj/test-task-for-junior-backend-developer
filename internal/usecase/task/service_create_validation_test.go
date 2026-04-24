package task

import (
	"encoding/json"
	"errors"
	"testing"

	taskdomain "example.com/taskservice/internal/domain/task"
)

func TestValidateCreateInput_RecurrenceConfigWithoutType(t *testing.T) {
	input := CreateInput{
		Title:            "Task",
		Status:           taskdomain.StatusNew,
		RecurrenceConfig: json.RawMessage(`{"interval_days":1}`),
	}

	_, err := validateCreateInput(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateCreateInput_RecurringRequiresConfig(t *testing.T) {
	typeValue := "daily"
	input := CreateInput{
		Title:          "Task",
		Status:         taskdomain.StatusNew,
		RecurrenceType: &typeValue,
	}

	_, err := validateCreateInput(input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestValidateCreateInput_NormalizesRecurrenceType(t *testing.T) {
	typeValue := "  DAILY "
	input := CreateInput{
		Title:            "Task",
		Status:           taskdomain.StatusNew,
		RecurrenceType:   &typeValue,
		RecurrenceConfig: json.RawMessage(`{"interval_days":1}`),
	}

	normalized, err := validateCreateInput(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized.RecurrenceType == nil {
		t.Fatal("expected normalized recurrence type, got nil")
	}
	if *normalized.RecurrenceType != "daily" {
		t.Fatalf("expected daily, got %s", *normalized.RecurrenceType)
	}
}

func TestServiceList_InvalidFilter(t *testing.T) {
	s := &Service{}

	_, err := s.List(nil, "invalid")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
