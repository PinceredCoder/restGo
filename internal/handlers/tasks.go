package handlers

import (
	"io"
	"net/http"

	tasks "github.com/PinceredCoder/restGo/api/proto/v1"
	"github.com/PinceredCoder/restGo/internal/database"
	"github.com/PinceredCoder/restGo/internal/errors"
	"github.com/PinceredCoder/restGo/internal/helpers"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: add logs for InternalServerError cases

type TaskHandler struct {
	db database.Database
}

func NewTaskHandler(db database.Database) *TaskHandler {
	return &TaskHandler{db: db}
}

func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	taskList, err := h.db.GetTaskRepository().FindAll(r.Context())

	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to retrieve tasks"))
		return
	}

	response := &tasks.ListTasksResponse{
		Tasks: helpers.Map(taskList, func(t *database.Task) *tasks.Task { return t.ToProto() }),
	}

	data, err := protojson.Marshal(response)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to encode response"))
		return
	}

	w.Write(data)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest,
			errors.NewBadRequestError("Failed to read request body"))
		return
	}

	var req tasks.CreateTaskRequest
	if err := protojson.Unmarshal(data, &req); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest,
			errors.NewBadRequestError("Invalid JSON format"))
		return
	}

	if err := req.Validate(); err != nil {
		apiErr := h.convertValidationError(err)
		errors.RespondWithError(w, http.StatusBadRequest, apiErr)
		return
	}

	now := timestamppb.Now().AsTime().Unix()
	taskID := uuid.New()

	taskDb := &database.Task{
		ID:          taskID,
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.db.GetTaskRepository().Create(r.Context(), taskDb); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to create task"))
		return
	}

	response := &tasks.GetTaskResponse{
		Task: taskDb.ToProto(),
	}

	data, err = protojson.Marshal(response)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to encode response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest,
			errors.NewBadRequestError("Invalid task ID format"))
		return
	}

	taskDb, err := h.db.GetTaskRepository().FindByID(r.Context(), id)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to retrieve task"))
		return
	}

	if taskDb == nil {
		errors.RespondWithError(w, http.StatusNotFound,
			errors.NewNotFoundError("Task not found"))
		return
	}

	response := &tasks.GetTaskResponse{
		Task: taskDb.ToProto(),
	}

	data, err := protojson.Marshal(response)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to encode response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest,
			errors.NewBadRequestError("Invalid task ID format"))
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest,
			errors.NewBadRequestError("Failed to read request body"))
		return
	}

	var req tasks.UpdateTaskRequest
	if err := protojson.Unmarshal(data, &req); err != nil {
		errors.RespondWithError(w, http.StatusBadRequest,
			errors.NewBadRequestError("Invalid JSON format"))
		return
	}

	if err := req.Validate(); err != nil {
		apiErr := h.convertValidationError(err)
		errors.RespondWithError(w, http.StatusBadRequest, apiErr)
		return
	}

	task, err := h.db.GetTaskRepository().FindByID(r.Context(), id)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to retrieve task"))
		return
	}
	if task == nil {
		errors.RespondWithError(w, http.StatusNotFound,
			errors.NewNotFoundError("Task not found"))
		return
	}

	task.Title = req.Title
	task.Description = req.Description

	if req.Completed != nil {
		task.Completed = *req.Completed
	}

	task.UpdatedAt = timestamppb.Now().AsTime().Unix()

	if err := h.db.GetTaskRepository().Update(r.Context(), id, task); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to update task"))
		return
	}

	response := &tasks.GetTaskResponse{
		Task: task.ToProto(),
	}

	data, err = protojson.Marshal(response)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to encode response"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest,
			errors.NewBadRequestError("Invalid task ID format"))
		return
	}

	if err := h.db.GetTaskRepository().Delete(r.Context(), id); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError,
			errors.NewInternalError("Failed to delete task"))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
