package product

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request CreateProductRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	product, err := h.service.Create(r.Context(), request)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.JSON(w, http.StatusCreated, product)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	product, err := h.service.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load product")
		return
	}

	httpx.JSON(w, http.StatusOK, product)
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	var categoryID *int64
	if value := r.URL.Query().Get("category_id"); value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			httpx.Error(w, http.StatusBadRequest, "invalid category id")
			return
		}
		categoryID = &id
	}

	products, err := h.service.GetAll(r.Context(), categoryID, r.URL.Query().Get("q"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load products")
		return
	}

	httpx.JSON(w, http.StatusOK, products)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var request UpdateProductRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	product, err := h.service.Update(r.Context(), id, request)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.JSON(w, http.StatusOK, product)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid product id")
		return
	}

	if err := h.service.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not delete product")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
