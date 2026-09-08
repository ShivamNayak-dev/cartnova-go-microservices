package order

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ShivamNayak-dev/cartnova/internal/auth"
	"github.com/ShivamNayak-dev/cartnova/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var request CreateOrderRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.service.Create(r.Context(), claims.UserID, request)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	httpx.JSON(w, http.StatusCreated, order)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load order")
		return
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || (claims.Role != "ADMIN" && claims.UserID != order.UserID) {
		httpx.Error(w, http.StatusForbidden, "you cannot access this order")
		return
	}

	httpx.JSON(w, http.StatusOK, order)
}

func (h *Handler) GetMine(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	orders, err := h.service.GetByUserID(r.Context(), claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load orders")
		return
	}

	httpx.JSON(w, http.StatusOK, orders)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok || claims.Role != "ADMIN" {
		httpx.Error(w, http.StatusForbidden, "admin access required")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid order id")
		return
	}

	var request UpdateStatusRequest
	if err := httpx.DecodeJSON(r, &request); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.service.UpdateStatus(r.Context(), id, request.Status)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, err.Error())
		return
	}
	if errors.Is(err, ErrInvalidStatus) {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not update order status")
		return
	}

	httpx.JSON(w, http.StatusOK, order)
}
