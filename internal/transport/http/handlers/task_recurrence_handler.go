package handlers

import (
	"errors"
	"net/http"

	taskrecurrencedomain "example.com/taskservice/internal/domain/taskrecurrence"
	taskrecurrenceusecase "example.com/taskservice/internal/usecase/taskrecurrence"
)

type TaskRecurrenceHandler struct {
	usecase taskrecurrenceusecase.Usecase
}

func NewTaskRecurrenceHandler(usecase taskrecurrenceusecase.Usecase) *TaskRecurrenceHandler {
	return &TaskRecurrenceHandler{usecase: usecase}
}

func (h *TaskRecurrenceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req taskRecurrenceMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	input, err := req.toCreateInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.usecase.Create(r.Context(), input)
	if err != nil {
		writeTaskRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTaskRecurrenceDTO(created))
}

func (h *TaskRecurrenceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	recurrence, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeTaskRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskRecurrenceDTO(recurrence))
}

func (h *TaskRecurrenceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req taskRecurrenceMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	input, err := req.toUpdateInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.usecase.Update(r.Context(), id, input)
	if err != nil {
		writeTaskRecurrenceUsecaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskRecurrenceDTO(updated))
}

func (h *TaskRecurrenceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeTaskRecurrenceUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskRecurrenceHandler) List(w http.ResponseWriter, r *http.Request) {
	recurrences, err := h.usecase.List(r.Context())
	if err != nil {
		writeTaskRecurrenceUsecaseError(w, err)
		return
	}

	response := make([]taskRecurrenceDTO, 0, len(recurrences))
	for i := range recurrences {
		response = append(response, newTaskRecurrenceDTO(&recurrences[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func writeTaskRecurrenceUsecaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskrecurrencedomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskrecurrenceusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}
