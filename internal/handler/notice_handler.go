package handler

import (
	"github.com/reinoplus/reinoplus/internal/httputil"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type NoticeHandler struct {
	logger *zap.Logger
}

func NewNoticeHandler(logger *zap.Logger) *NoticeHandler {
	return &NoticeHandler{logger: logger}
}

type noticeResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (h *NoticeHandler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.List)
}

func (h *NoticeHandler) List(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, []noticeResponse{})
}
