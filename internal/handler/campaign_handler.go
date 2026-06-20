package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/httputil"
	"github.com/reinoplus/reinoplus/internal/middleware"
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
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Goal        float64 `json:"goal"`
	Raised      float64 `json:"raised"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	Status      string  `json:"status"`
}

type campaignRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Goal        float64 `json:"goal"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
}

type contributionResponse struct {
	ID               string  `json:"id"`
	CampaignID       string  `json:"campaign_id"`
	MemberID         *string `json:"member_id,omitempty"`
	ContributorName  string  `json:"contributor_name"`
	ContributorPhone string  `json:"contributor_phone"`
	MemberName       string  `json:"member_name"`
	Amount           float64 `json:"amount"`
	PaymentMethod    string  `json:"payment_method"`
	ContributedAt    string  `json:"contributed_at"`
	CreatedByName    string  `json:"created_by_name,omitempty"`
}

type contributionRequest struct {
	ContributorName  string  `json:"contributor_name"`
	ContributorPhone string  `json:"contributor_phone"`
	Amount           float64 `json:"amount"`
	PaymentMethod    string  `json:"payment_method"`
	ContributedAt    string  `json:"contributed_at"`
}

func (h *CampaignHandler) RegisterReadRoutes(r chi.Router) {
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Get("/{id}/contributions", h.ListContributions)
}

func (h *CampaignHandler) RegisterWriteRoutes(r chi.Router) {
	r.Post("/", h.Create)
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
		raised, err := h.service.GetByID(r.Context(), item.ID)
		if err != nil {
			httputil.WriteError(w, h.logger, err)
			return
		}
		response = append(response, toCampaignResponse(raised.Campaign, raised.Raised))
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *CampaignHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	detail, err := h.service.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toCampaignResponse(detail.Campaign, detail.Raised))
}

func (h *CampaignHandler) ListContributions(w http.ResponseWriter, r *http.Request) {
	contributions, err := h.service.ListContributions(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	response := make([]contributionResponse, 0, len(contributions))
	for _, item := range contributions {
		response = append(response, toContributionResponse(item))
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
	endDate, err := parseDate(req.EndDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	created, err := h.service.Create(r.Context(), campaign.CreateInput{
		Name: req.Name, Description: req.Description, GoalAmount: req.Goal,
		StartDate: startDate, EndDate: endDate,
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toCampaignResponse(created, 0))
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

	created, err := h.service.CreateContribution(r.Context(), campaign.CreateContributionInput{
		CampaignID:       chi.URLParam(r, "id"),
		ContributorName:  req.ContributorName,
		ContributorPhone: req.ContributorPhone,
		Amount:           req.Amount,
		PaymentMethod:    domain.PaymentMethod(req.PaymentMethod),
		ContributedAt:    contributedAt,
		CreatedByUserID:  middleware.GetUserID(r.Context()),
	})
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toContributionResponse(created))
}

func toCampaignResponse(item domain.Campaign, raised float64) campaignResponse {
	return campaignResponse{
		ID: item.ID, Name: item.Name, Description: item.Description,
		Goal: item.GoalAmount, Raised: raised,
		StartDate: item.StartDate.Format("2006-01-02"),
		EndDate: item.EndDate.Format("2006-01-02"),
		Status: mapCampaignStatus(item.Status),
	}
}

func toContributionResponse(item domain.Contribution) contributionResponse {
	return contributionResponse{
		ID: item.ID, CampaignID: item.CampaignID, MemberID: item.MemberID,
		ContributorName: item.ContributorName, ContributorPhone: item.ContributorPhone,
		MemberName: item.MemberName, Amount: item.Amount,
		PaymentMethod: string(item.PaymentMethod),
		ContributedAt: item.ContributedAt.Format("2006-01-02"),
		CreatedByName: item.CreatedByName,
	}
}

func mapCampaignStatus(status domain.CampaignStatus) string {
	if status == domain.CampaignStatusCompleted {
		return "finished"
	}
	return string(status)
}
