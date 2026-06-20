package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/reinoplus/reinoplus/internal/httputil"

	"github.com/go-chi/chi/v5"
	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/middleware"
	"github.com/reinoplus/reinoplus/internal/usecase/raffle"
	"go.uber.org/zap"
)

type RaffleHandler struct {
	service *raffle.Service
	logger  *zap.Logger
}

func NewRaffleHandler(service *raffle.Service, logger *zap.Logger) *RaffleHandler {
	return &RaffleHandler{service: service, logger: logger}
}

type raffleResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Goal         float64  `json:"goal"`
	TicketPrice  float64  `json:"ticket_price"`
	TotalNumbers int      `json:"total_numbers"`
	SoldNumbers  int      `json:"sold_numbers"`
	DrawDate     string   `json:"draw_date"`
	Prizes       []string `json:"prizes"`
	Status       string   `json:"status"`
	WinningNumber *int    `json:"winning_number,omitempty"`
	WinnerName   *string  `json:"winner_name,omitempty"`
}

type raffleRequest struct {
	Name         string   `json:"name"`
	Goal         float64  `json:"goal"`
	TicketPrice  float64  `json:"ticket_price"`
	TotalNumbers int      `json:"total_numbers"`
	DrawDate     string   `json:"draw_date"`
	Prizes       []string `json:"prizes"`
}

type raffleTicketResponse struct {
	Number        int     `json:"number"`
	Status        string  `json:"status"`
	BuyerName     *string `json:"buyer_name,omitempty"`
	BuyerPhone    *string `json:"buyer_phone,omitempty"`
	MemberID      *string `json:"member_id,omitempty"`
	PaymentMethod *string `json:"payment_method,omitempty"`
	SoldAt        *string `json:"sold_at,omitempty"`
	SoldByName    *string `json:"sold_by_name,omitempty"`
}

type sellNumberRequest struct {
	BuyerName     string  `json:"buyer_name"`
	BuyerPhone    string  `json:"buyer_phone"`
	PaymentMethod string  `json:"payment_method"`
	MemberID      *string `json:"member_id,omitempty"`
}

type sellBatchRequest struct {
	Numbers       []int   `json:"numbers"`
	BuyerName     string  `json:"buyer_name"`
	BuyerPhone    string  `json:"buyer_phone"`
	PaymentMethod string  `json:"payment_method"`
	MemberID      *string `json:"member_id,omitempty"`
}

type drawRequest struct {
	WinningNumber int `json:"winning_number"`
}

func (h *RaffleHandler) RegisterReadRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Get("/active", h.GetActive)
	r.Get("/{id}", h.GetByID)
	r.Get("/{id}/numbers", h.ListNumbers)
	r.Get("/{id}/tickets", h.ListNumbers)
}

func (h *RaffleHandler) RegisterSellRoutes(r chi.Router) {
	r.Post("/{id}/numbers/{number}/sell", h.SellNumber)
	r.Post("/{id}/sales", h.SellBatch)
}

func (h *RaffleHandler) RegisterWriteRoutes(r chi.Router) {
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Post("/{id}/draw", h.Draw)
}

func (h *RaffleHandler) RegisterRoutes(r chi.Router) {
	h.RegisterReadRoutes(r)
	h.RegisterSellRoutes(r)
	h.RegisterWriteRoutes(r)
}

func (h *RaffleHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context())
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	response := make([]raffleResponse, 0, len(items))
	for _, item := range items {
		response = append(response, toRaffleResponse(item))
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *RaffleHandler) GetActive(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetActive(r.Context())
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	if item == nil {
		httputil.WriteJSON(w, http.StatusOK, nil)
		return
	}
	response := toRaffleResponse(*item)
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *RaffleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toRaffleResponse(item))
}

func (h *RaffleHandler) ListNumbers(w http.ResponseWriter, r *http.Request) {
	numbers, err := h.service.ListNumbers(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	response := make([]raffleTicketResponse, 0, len(numbers))
	for _, number := range numbers {
		response = append(response, toRaffleTicketResponse(number))
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *RaffleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req raffleRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	drawDate, err := parseDate(req.DrawDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	created, err := h.service.Create(r.Context(), raffle.CreateInput{
		Name: req.Name, GoalAmount: req.Goal, PointValue: req.TicketPrice,
		TotalNumbers: req.TotalNumbers, DrawDate: drawDate, Prizes: req.Prizes,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toRaffleResponse(created))
}

func (h *RaffleHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req raffleRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	drawDate, err := parseDate(req.DrawDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	updated, err := h.service.Update(r.Context(), chi.URLParam(r, "id"), raffle.UpdateInput{
		Name: req.Name, GoalAmount: req.Goal, PointValue: req.TicketPrice,
		DrawDate: drawDate, Prizes: req.Prizes,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toRaffleResponse(updated))
}

func (h *RaffleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RaffleHandler) SellNumber(w http.ResponseWriter, r *http.Request) {
	var req sellNumberRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	number, err := strconv.Atoi(chi.URLParam(r, "number"))
	if err != nil {
		httputil.WriteError(w, h.logger, domain.ErrValidation)
		return
	}

	err = h.service.SellNumber(r.Context(), chi.URLParam(r, "id"), number, raffle.SellNumberInput{
		BuyerName: req.BuyerName, BuyerPhone: req.BuyerPhone, MemberID: req.MemberID,
		PaymentMethod: domain.PaymentMethod(req.PaymentMethod),
		SoldByUserID: middleware.GetUserID(r.Context()),
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RaffleHandler) SellBatch(w http.ResponseWriter, r *http.Request) {
	var req sellBatchRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	err := h.service.SellNumbers(r.Context(), chi.URLParam(r, "id"), req.Numbers, raffle.SellNumberInput{
		BuyerName: req.BuyerName, BuyerPhone: req.BuyerPhone, MemberID: req.MemberID,
		PaymentMethod: domain.PaymentMethod(req.PaymentMethod),
		SoldByUserID: middleware.GetUserID(r.Context()),
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RaffleHandler) Draw(w http.ResponseWriter, r *http.Request) {
	var req drawRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	result, err := h.service.Draw(r.Context(), raffle.DrawInput{
		RaffleID: chi.URLParam(r, "id"), WinningNumber: req.WinningNumber,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toRaffleResponse(result))
}

func toRaffleResponse(item domain.RaffleSummary) raffleResponse {
	prizes := make([]string, 0, len(item.Prizes))
	for _, prize := range item.Prizes {
		prizes = append(prizes, prize.Description)
	}
	return raffleResponse{
		ID: item.ID, Name: item.Name, Goal: item.GoalAmount, TicketPrice: item.PointValue,
		TotalNumbers: item.TotalNumbers, SoldNumbers: item.SoldNumbers,
		DrawDate: item.DrawDate.Format("2006-01-02"), Prizes: prizes,
		Status: string(item.Status), WinningNumber: item.WinningNumber, WinnerName: item.WinnerName,
	}
}

func toRaffleTicketResponse(item domain.RaffleNumber) raffleTicketResponse {
	resp := raffleTicketResponse{
		Number: item.Number, Status: string(item.Status),
		BuyerName: item.BuyerName, BuyerPhone: item.BuyerPhone, MemberID: item.MemberID,
	}
	if item.PaymentMethod != nil {
		method := string(*item.PaymentMethod)
		resp.PaymentMethod = &method
	}
	if item.SoldAt != nil {
		soldAt := item.SoldAt.Format(time.RFC3339)
		resp.SoldAt = &soldAt
	}
	if item.SoldByName != nil {
		resp.SoldByName = item.SoldByName
	}
	return resp
}
