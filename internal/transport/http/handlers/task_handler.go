package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TaskHandler struct {
	usecase taskusecase.Usecase
	logger  *slog.Logger
}

func NewTaskHandler(usecase taskusecase.Usecase, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{usecase: usecase, logger: logger}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("create task: request received")

	var req taskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		h.logger.Error("create task: invalid request body", "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), taskusecase.CreateInput{
		Title:            req.Title,
		Description:      req.Description,
		Status:           req.Status,
		RecurrenceType:   req.RecurrenceType,
		RecurrenceConfig: req.RecurrenceConfig,
	})
	if err != nil {
		h.logger.Error("create task: usecase failed", "error", err)
		writeUsecaseError(w, err)
		return
	}

	h.logger.Info("task created", "task_id", created.ID, "title", created.Title)

	writeJSON(w, http.StatusCreated, newTaskDTO(created))
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("get task by id: request received")

	id, err := getIDFromRequest(r)
	if err != nil {
		h.logger.Error("get task by id: invalid id", "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	task, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		h.logger.Error("get task by id: usecase failed", "task_id", id, "error", err)
		writeUsecaseError(w, err)
		return
	}

	h.logger.Info("task fetched", "task_id", task.ID)

	writeJSON(w, http.StatusOK, newTaskDTO(task))
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("update task: request received")

	id, err := getIDFromRequest(r)
	if err != nil {
		h.logger.Error("update task: invalid id", "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req taskMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		h.logger.Error("update task: invalid request body", "task_id", id, "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, taskusecase.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
	})
	if err != nil {
		h.logger.Error("update task: usecase failed", "task_id", id, "error", err)
		writeUsecaseError(w, err)
		return
	}

	h.logger.Info("task updated", "task_id", updated.ID, "title", updated.Title)

	writeJSON(w, http.StatusOK, newTaskDTO(updated))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("delete task: request received")

	id, err := getIDFromRequest(r)
	if err != nil {
		h.logger.Error("delete task: invalid id", "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		h.logger.Error("delete task: usecase failed", "task_id", id, "error", err)
		writeUsecaseError(w, err)
		return
	}

	h.logger.Info("task deleted", "task_id", id)

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter")
	h.logger.Debug("list tasks: request received", "filter", filter)

	tasks, err := h.usecase.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list tasks: usecase failed", "filter", filter, "error", err)
		writeUsecaseError(w, err)
		return
	}

	response := make([]taskDTO, 0, len(tasks))
	for i := range tasks {
		response = append(response, newTaskDTO(&tasks[i]))
	}

	h.logger.Info("tasks listed", "filter", filter, "count", len(response))

	writeJSON(w, http.StatusOK, response)
}

func getIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing task id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid task id")
	}

	if id <= 0 {
		return 0, errors.New("invalid task id")
	}

	return id, nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

func writeUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
