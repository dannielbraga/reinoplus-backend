package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/httputil"
	"github.com/reinoplus/reinoplus/internal/middleware"
	"github.com/reinoplus/reinoplus/internal/textutil"
	"github.com/reinoplus/reinoplus/internal/usecase/campaign"
	"go.uber.org/zap"
)

type CampaignHandler struct {
	service *campaign.Service
	logger  *zap.Logger
}

func NewCampaignHandler(service *campaign.Service, logger *zap.Logger) *CampaignHandler {
	return &CampaignHandler{service: service, logger: logger}
}

type campaignResponse struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Goal               float64 `json:"goal"`
	Raised             float64 `json:"raised"`
	Promised           float64 `json:"promised"`
	StartDate          string  `json:"start_date"`
	EndDate            string  `json:"end_date"`
	Status             string  `json:"status"`
	IsRecurring        bool    `json:"is_recurring"`
	RecurrenceInterval *string `json:"recurrence_interval,omitempty"`
	DurationMonths     *int    `json:"duration_months,omitempty"`
	TotalInstallments  *int    `json:"total_installments,omitempty"`
}

type campaignRequest struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Goal               float64 `json:"goal"`
	StartDate          string  `json:"start_date"`
	EndDate            string  `json:"end_date,omitempty"`
	IsRecurring        bool    `json:"is_recurring"`
	RecurrenceInterval *string `json:"recurrence_interval,omitempty"`
	DurationMonths     *int    `json:"duration_months,omitempty"`
}

type contributionResponse struct {
	ID                string  `json:"id"`
	CampaignID        string  `json:"campaign_id"`
	MemberID          *string `json:"member_id,omitempty"`
	ContributorName   string  `json:"contributor_name"`
	ContributorPhone  string  `json:"contributor_phone"`
	MemberName        string  `json:"member_name"`
	Amount            float64 `json:"amount"`
	PaymentMethod     string  `json:"payment_method"`
	ContributedAt     string  `json:"contributed_at"`
	IsPaid            bool    `json:"is_paid"`
	InstallmentNumber *int    `json:"installment_number,omitempty"`
	TotalInstallments *int    `json:"total_installments,omitempty"`
	CreatedByName     string  `json:"created_by_name,omitempty"`
}

type contributionRequest struct {
	ContributorName   string  `json:"contributor_name"`
	ContributorPhone  string  `json:"contributor_phone"`
	Amount            float64 `json:"amount"`
	PaymentMethod     string  `json:"payment_method"`
	ContributedAt     string  `json:"contributed_at"`
	IsPaid            bool    `json:"is_paid"`
	InstallmentNumber *int    `json:"installment_number,omitempty"`
}

type campaignStatusRequest struct {
	Active bool `json:"active"`
}

func (h *CampaignHandler) RegisterReadRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Get("/{id}/contributions", h.ListContributions)
}

func (h *CampaignHandler) RegisterWriteRoutes(r chi.Router) {
	r.Post("/", h.Create)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Patch("/{id}/status", h.SetStatus)
	r.Post("/{id}/contributions", h.CreateContribution)
}

func (h *CampaignHandler) RegisterRoutes(r chi.Router) {
	h.RegisterReadRoutes(r)
	h.RegisterWriteRoutes(r)
}

func (h *CampaignHandler) List(w http.ResponseWriter, r *http.Request) {
	campaigns, err := h.service.List(r.Context())
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	response := make([]campaignResponse, 0, len(campaigns))
	for _, item := range campaigns {
		detail, err := h.service.GetByID(r.Context(), item.ID)
		if err != nil {
			httputil.WriteError(w, h.logger, err)
			return
		}
		response = append(response, toCampaignResponse(detail.Campaign, detail.Raised, detail.Promised))
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *CampaignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toCampaignResponse(detail.Campaign, detail.Raised, detail.Promised))
}

func (h *CampaignHandler) ListContributions(w http.ResponseWriter, r *http.Request) {
	campaignID := chi.URLParam(r, "id")
	campaignEntity, err := h.service.GetByID(r.Context(), campaignID)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	contributions, err := h.service.ListContributions(r.Context(), campaignID, r.URL.Query().Get("search"))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	totalInstallments := campaignTotalInstallments(campaignEntity.Campaign)
	response := make([]contributionResponse, 0, len(contributions))
	for _, item := range contributions {
		response = append(response, toContributionResponse(item, totalInstallments))
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *CampaignHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req campaignRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	endDate := time.Time{}
	if !req.IsRecurring {
		if req.EndDate == "" {
			httputil.WriteError(w, h.logger, domain.ErrValidation)
			return
		}
		endDate, err = parseDate(req.EndDate)
		if err != nil {
			httputil.WriteError(w, h.logger, err)
			return
		}
	}

	created, err := h.service.Create(r.Context(), campaign.CreateInput{
		Name:               req.Name,
		Description:        req.Description,
		GoalAmount:         req.Goal,
		StartDate:          startDate,
		EndDate:            endDate,
		IsRecurring:        req.IsRecurring,
		RecurrenceInterval: parseCampaignRecurrenceInterval(req.IsRecurring, req.RecurrenceInterval),
		DurationMonths:     req.DurationMonths,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toCampaignResponse(created, 0, 0))
}

func (h *CampaignHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req campaignRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	endDate := time.Time{}
	if !req.IsRecurring {
		if req.EndDate == "" {
			httputil.WriteError(w, h.logger, domain.ErrValidation)
			return
		}
		endDate, err = parseDate(req.EndDate)
		if err != nil {
			httputil.WriteError(w, h.logger, err)
			return
		}
	}

	updated, err := h.service.Update(r.Context(), chi.URLParam(r, "id"), campaign.UpdateInput{
		Name:               req.Name,
		Description:        req.Description,
		GoalAmount:         req.Goal,
		StartDate:          startDate,
		EndDate:            endDate,
		IsRecurring:        req.IsRecurring,
		RecurrenceInterval: parseCampaignRecurrenceInterval(req.IsRecurring, req.RecurrenceInterval),
		DurationMonths:     req.DurationMonths,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	detail, err := h.service.GetByID(r.Context(), updated.ID)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toCampaignResponse(detail.Campaign, detail.Raised, detail.Promised))
}

func (h *CampaignHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CampaignHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	var req campaignStatusRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	updated, err := h.service.SetStatus(r.Context(), chi.URLParam(r, "id"), req.Active)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	detail, err := h.service.GetByID(r.Context(), updated.ID)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toCampaignResponse(detail.Campaign, detail.Raised, detail.Promised))
}

func (h *CampaignHandler) CreateContribution(w http.ResponseWriter, r *http.Request) {
	var req contributionRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	contributedAt, err := parseDate(req.ContributedAt)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	campaignID := chi.URLParam(r, "id")
	created, err := h.service.CreateContribution(r.Context(), campaign.CreateContributionInput{
		CampaignID:        campaignID,
		ContributorName:   req.ContributorName,
		ContributorPhone:  req.ContributorPhone,
		Amount:            req.Amount,
		PaymentMethod:     domain.PaymentMethod(req.PaymentMethod),
		ContributedAt:     contributedAt,
		IsPaid:            req.IsPaid,
		InstallmentNumber: req.InstallmentNumber,
		CreatedByUserID:   middleware.GetUserID(r.Context()),
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	campaignEntity, err := h.service.GetByID(r.Context(), campaignID)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	httputil.WriteJSON(
		w,
		http.StatusCreated,
		toContributionResponse(created, campaignTotalInstallments(campaignEntity.Campaign)),
	)
}

func toCampaignResponse(item domain.Campaign, raised float64, promised float64) campaignResponse {
	return campaignResponse{
		ID: item.ID, Name: item.Name, Description: item.Description,
		Goal: item.GoalAmount, Raised: raised, Promised: promised,
		StartDate: item.StartDate.Format("2006-01-02"),
		EndDate:   item.EndDate.Format("2006-01-02"),
		Status:    mapCampaignStatus(item.Status),
		IsRecurring: item.IsRecurring,
		RecurrenceInterval: formatRecurrenceInterval(item.RecurrenceInterval),
		DurationMonths: item.DurationMonths,
		TotalInstallments: campaignTotalInstallments(item),
	}
}

func toContributionResponse(item domain.Contribution, totalInstallments *int) contributionResponse {
	return contributionResponse{
		ID: item.ID, CampaignID: item.CampaignID, MemberID: item.MemberID,
		ContributorName: textutil.TitleCaseName(item.ContributorName), ContributorPhone: item.ContributorPhone,
		MemberName: textutil.TitleCaseName(item.MemberName), Amount: item.Amount,
		PaymentMethod: string(item.PaymentMethod),
		ContributedAt: item.ContributedAt.Format("2006-01-02"),
		IsPaid: item.IsPaid,
		InstallmentNumber: item.InstallmentNumber,
		TotalInstallments: totalInstallments,
		CreatedByName: textutil.TitleCaseName(item.CreatedByName),
	}
}

func campaignTotalInstallments(item domain.Campaign) *int {
	if !item.IsRecurring || item.RecurrenceInterval == nil || item.DurationMonths == nil {
		return nil
	}
	total := domain.TotalInstallments(*item.DurationMonths, *item.RecurrenceInterval)
	return &total
}

func parseCampaignRecurrenceInterval(isRecurring bool, value *string) *domain.RecurrenceInterval {
	if !isRecurring || value == nil {
		return nil
	}
	interval := domain.RecurrenceInterval(*value)
	if !domain.IsValidRecurrenceInterval(interval) {
		return nil
	}
	return &interval
}

func formatRecurrenceInterval(value *domain.RecurrenceInterval) *string {
	if value == nil {
		return nil
	}
	formatted := string(*value)
	return &formatted
}

func mapCampaignStatus(status domain.CampaignStatus) string {
	switch status {
	case domain.CampaignStatusCompleted:
		return "finished"
	case domain.CampaignStatusCancelled:
		return "inactive"
	default:
		return string(status)
	}
}
