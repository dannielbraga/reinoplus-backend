package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/httputil"
	"github.com/reinoplus/reinoplus/internal/textutil"
	"github.com/reinoplus/reinoplus/internal/usecase/member"
	"go.uber.org/zap"
)

type MemberHandler struct {
	service *member.Service
	logger  *zap.Logger
}

func NewMemberHandler(service *member.Service, logger *zap.Logger) *MemberHandler {
	return &MemberHandler{service: service, logger: logger}
}

type memberResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
	Address   string `json:"address"`
	CreatedAt string `json:"created_at,omitempty"`
}

type memberRequest struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
	Address   string `json:"address"`
}

func (h *MemberHandler) RegisterReadRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Get("/search", h.SearchByPhone)
	r.Get("/{id}", h.GetByID)
}

func (h *MemberHandler) RegisterWriteRoutes(r chi.Router) {
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
}

func (h *MemberHandler) RegisterRoutes(r chi.Router) {
	h.RegisterReadRoutes(r)
	h.RegisterWriteRoutes(r)
}

func (h *MemberHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	members, err := h.service.List(r.Context(), search, limit, offset)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	response := make([]memberResponse, 0, len(members))
	for _, item := range members {
		response = append(response, toMemberResponse(item))
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *MemberHandler) SearchByPhone(w http.ResponseWriter, r *http.Request) {
	phone := r.URL.Query().Get("phone")
	found, err := h.service.SearchByPhone(r.Context(), phone)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	if found == nil {
		httputil.WriteJSON(w, http.StatusOK, nil)
		return
	}
	response := toMemberResponse(*found)
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *MemberHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toMemberResponse(item))
}

func (h *MemberHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req memberRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	birthDate, err := parseDate(req.BirthDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	created, err := h.service.Create(r.Context(), member.CreateInput{
		Name: req.Name, Phone: req.Phone, BirthDate: birthDate, Address: req.Address,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toMemberResponse(created))
}

func (h *MemberHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req memberRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	birthDate, err := parseDate(req.BirthDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	updated, err := h.service.Update(r.Context(), member.UpdateInput{
		ID: chi.URLParam(r, "id"), Name: req.Name, Phone: req.Phone, BirthDate: birthDate, Address: req.Address,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toMemberResponse(updated))
}

func toMemberResponse(item domain.Member) memberResponse {
	return memberResponse{
		ID:        item.ID,
		Name:      textutil.TitleCaseName(item.Name),
		Phone:     item.Phone,
		BirthDate: item.BirthDate.Format("2006-01-02"),
		Address:   item.Address,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}
}
