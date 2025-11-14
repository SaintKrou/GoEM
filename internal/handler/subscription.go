package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"subscription-service/internal/model"
	"subscription-service/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Handler struct {
	SubscriptionRepo *repository.SubscriptionRepository
}

func NewHandler(repo *repository.SubscriptionRepository) *Handler {
	return &Handler{SubscriptionRepo: repo}
}

func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Ошибка при отправке JSON: %v", err)
		http.Error(w, "Ошибка сериализации", http.StatusInternalServerError)
	}
}

func LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("→ %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// CreateSubscription creates a new subscription.
//
//	@Summary		Create a subscription
//	@Description	Create a new user subscription record
//	@Tags			subscriptions
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateSubscriptionRequest	true	"Subscription data"
//	@Success		201	{object}	model.Subscription
//	@Failure		400	{string}	string	"Bad request"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/subscriptions [post]
func (h *Handler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ServiceName string  `json:"service_name"`
		Price       int     `json:"price"`
		UserID      string  `json:"user_id"`
		StartDate   string  `json:"start_date"`
		EndDate     *string `json:"end_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	if input.ServiceName == "" {
		http.Error(w, "service_name не может быть пустым", http.StatusBadRequest)
		return
	}

	if input.Price <= 0 {
		http.Error(w, "price должен быть положительным целым числом", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		http.Error(w, "user_id должен быть валидным UUID", http.StatusBadRequest)
		return
	}

	if !isValidMonthYear(input.StartDate) {
		http.Error(w, "start_date должен быть в формате MM-YYYY (например, 07-2025)", http.StatusBadRequest)
		return
	}

	sub := &model.Subscription{
		ID:          uuid.New(),
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      userID,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
	}

	if err := h.SubscriptionRepo.Create(sub); err != nil {
		http.Error(w, "Не удалось сохранить подписку", http.StatusInternalServerError)
		return
	}

	saved, err := h.SubscriptionRepo.GetByID(sub.ID)
	if err != nil {
		http.Error(w, "Не удалось получить сохранённую подписку", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusCreated, saved)
}

// GetSubscriptionByID retrieves a subscription by ID.
//
//	@Summary		Get subscription by ID
//	@Description	Get a single subscription by its UUID
//	@Tags			subscriptions
//	@Produce		json
//	@Param			id	path		string	true	"Subscription ID"
//	@Success		200	{object}	model.Subscription
//	@Failure		400	{string}	string	"Invalid ID format"
//	@Failure		404	{string}	string	"Subscription not found"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/subscriptions/{id} [get]
func (h *Handler) GetSubscriptionByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Некорректный ID подписки", http.StatusBadRequest)
		return
	}

	sub, err := h.SubscriptionRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Подписка не найдена", http.StatusNotFound)
		return
	}

	SendJSON(w, http.StatusOK, sub)
}

// ListSubscriptions lists all subscriptions with optional filters.
//
//	@Summary		List subscriptions
//	@Description	Get a list of subscriptions, optionally filtered by user_id or service_name
//	@Tags			subscriptions
//	@Produce		json
//	@Param			user_id			query		string	false	"Filter by user ID"
//	@Param			service_name	query		string	false	"Filter by service name"
//	@Success		200	{array}		model.Subscription
//	@Failure		400	{string}	string	"Bad request"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/subscriptions [get]
func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")

	var userID *uuid.UUID
	if userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, "user_id должен быть валидным UUID", http.StatusBadRequest)
			return
		}
		userID = &id
	}

	var serviceNamePtr *string
	if serviceName != "" {
		serviceNamePtr = &serviceName
	}

	subs, err := h.SubscriptionRepo.List(userID, serviceNamePtr)
	if err != nil {
		http.Error(w, "Ошибка при получении списка подписок", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, subs)
}

// GetTotalCost calculates total cost for a period.
//
//	@Summary		Get total cost
//	@Description	Calculate total subscription cost for a given period with optional filters
//	@Tags			subscriptions
//	@Produce		json
//	@Param			start_month		query		string	true	"Start month in MM-YYYY format"
//	@Param			end_month		query		string	true	"End month in MM-YYYY format"
//	@Param			user_id			query		string	false	"Filter by user ID"
//	@Param			service_name	query		string	false	"Filter by service name"
//	@Success		200	{object}	object{total=int}
//	@Failure		400	{string}	string	"Bad request"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/subscriptions/total [get]
func (h *Handler) GetTotalCost(w http.ResponseWriter, r *http.Request) {
	startMonth := r.URL.Query().Get("start_month")
	endMonth := r.URL.Query().Get("end_month")

	if startMonth == "" || endMonth == "" {
		http.Error(w, "Параметры start_month и end_month обязательны", http.StatusBadRequest)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")

	var userID *uuid.UUID
	if userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			http.Error(w, "user_id должен быть валидным UUID", http.StatusBadRequest)
			return
		}
		userID = &id
	}

	var serviceNamePtr *string
	if serviceName != "" {
		serviceNamePtr = &serviceName
	}

	total, err := h.SubscriptionRepo.GetTotalCost(startMonth, endMonth, userID, serviceNamePtr)
	if err != nil {
		http.Error(w, "Ошибка при расчёте суммы", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, map[string]int{"total": total})
}

// UpdateSubscription updates an existing subscription.
//
//	@Summary		Update subscription
//	@Description	Update an existing subscription by ID
//	@Tags			subscriptions
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string	true	"Subscription ID"
//	@Param			request			body		CreateSubscriptionRequest	true	"Updated subscription data"
//	@Success		200	{object}	model.Subscription
//	@Failure		400	{string}	string	"Bad request"
//	@Failure		404	{string}	string	"Subscription not found"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/subscriptions/{id} [put]
func (h *Handler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Некорректный ID подписки", http.StatusBadRequest)
		return
	}

	var input struct {
		ServiceName string  `json:"service_name"`
		Price       int     `json:"price"`
		UserID      string  `json:"user_id"`
		StartDate   string  `json:"start_date"`
		EndDate     *string `json:"end_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Некорректное тело запроса", http.StatusBadRequest)
		return
	}

	if input.ServiceName == "" {
		http.Error(w, "service_name не может быть пустым", http.StatusBadRequest)
		return
	}

	if input.Price <= 0 {
		http.Error(w, "price должен быть положительным целым числом", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		http.Error(w, "user_id должен быть валидным UUID", http.StatusBadRequest)
		return
	}

	if !isValidMonthYear(input.StartDate) {
		http.Error(w, "start_date должен быть в формате MM-YYYY (например, 07-2025)", http.StatusBadRequest)
		return
	}

	sub := &model.Subscription{
		ID:          id,
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      userID,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
	}

	if err := h.SubscriptionRepo.Update(sub); err != nil {
		if err.Error() == fmt.Sprintf("подписка с ID %s не найдена", id) {
			http.Error(w, "Подписка не найдена", http.StatusNotFound)
			return
		}
		http.Error(w, "Не удалось обновить подписку", http.StatusInternalServerError)
		return
	}

	updated, err := h.SubscriptionRepo.GetByID(id)
	if err != nil {
		http.Error(w, "Не удалось получить обновлённую подписку", http.StatusInternalServerError)
		return
	}

	SendJSON(w, http.StatusOK, updated)
}

// DeleteSubscription deletes a subscription by ID.
//
//	@Summary		Delete subscription
//	@Description	Delete a subscription by its UUID
//	@Tags			subscriptions
//	@Param			id	path		string	true	"Subscription ID"
//	@Success		204	{string}	string
//	@Failure		400	{string}	string	"Invalid ID format"
//	@Failure		404	{string}	string	"Subscription not found"
//	@Failure		500	{string}	string	"Internal server error"
//	@Router			/subscriptions/{id} [delete]
func (h *Handler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Некорректный ID подписки", http.StatusBadRequest)
		return
	}

	if err := h.SubscriptionRepo.Delete(id); err != nil {
		if err.Error() == fmt.Sprintf("подписка с ID %s не найдена", id) {
			http.Error(w, "Подписка не найдена", http.StatusNotFound)
			return
		}
		http.Error(w, "Ошибка при удалении подписки", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func isValidMonthYear(s string) bool {
	_, err := time.Parse("01-2006", s)
	return err == nil
}

// CreateSubscriptionRequest represents the request body for creating/updating a subscription.
//
//	@Description	Subscription creation request
type CreateSubscriptionRequest struct {
	ServiceName string  `json:"service_name" example:"Yandex Plus"`
	Price       int     `json:"price" example:"400"`
	UserID      string  `json:"user_id" example:"60601fee-2bf1-4721-ae6f-7636e79a0cba"`
	StartDate   string  `json:"start_date" example:"07-2025"`
	EndDate     *string `json:"end_date,omitempty" example:"12-2025"`
}
