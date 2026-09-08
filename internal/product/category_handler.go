package product

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
)

type CategoryHandler struct {
	repository *CategoryRepository
}

func NewCategoryHandler(repository *CategoryRepository) *CategoryHandler {
	return &CategoryHandler{repository: repository}
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request CategoryRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "category name is required")
		return
	}

	category, err := h.repository.Create(r.Context(), name)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, category)
}

func (h *CategoryHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repository.FindAll(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load categories")
		return
	}
	httpx.JSON(w, http.StatusOK, categories)
}

func (h *CategoryHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	category, err := h.repository.FindByID(r.Context(), id)
	if errors.Is(err, ErrCategoryNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load category")
		return
	}

	httpx.JSON(w, http.StatusOK, category)
}

func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	if err := h.repository.Delete(r.Context(), id); errors.Is(err, ErrCategoryNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not delete category")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid category id")
		return
	}

	var request CategoryRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		httpx.Error(w, http.StatusBadRequest, "category name is required")
		return
	}

	category, err := h.repository.Update(r.Context(), id, name)
	if errors.Is(err, ErrCategoryNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, category)
}
