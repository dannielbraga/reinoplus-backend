package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/reinoplus/reinoplus/internal/domain"
	"go.uber.org/zap"
)

type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, logger *zap.Logger, err error) {
	status, response := MapDomainError(err)
	if status >= http.StatusInternalServerError {
		logger.Error("internal error", zap.Error(err))
		response.Message = "internal server error"
	}
	WriteJSON(w, status, response)
}

func MapDomainError(err error) (int, ErrorResponse) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{Message: "resource not found", Code: "not_found"}
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidToken):
		return http.StatusUnauthorized, ErrorResponse{Message: "unauthorized", Code: "unauthorized"}
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, ErrorResponse{Message: err.Error(), Code: "validation_error"}
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, ErrorResponse{Message: "forbidden", Code: "forbidden"}
	case errors.Is(err, domain.ErrRaffleHasSales):
		return http.StatusConflict, ErrorResponse{Message: "não é possível excluir rifa com números vendidos", Code: "raffle_has_sales"}
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrRaffleNumberAlreadySold):
		return http.StatusConflict, ErrorResponse{Message: "conflict", Code: "conflict"}
	case errors.Is(err, domain.ErrRaffleNotActive), errors.Is(err, domain.ErrCampaignNotActive):
		return http.StatusBadRequest, ErrorResponse{Message: err.Error(), Code: "invalid_state"}
	case errors.Is(err, domain.ErrInvalidRaffleNumber), errors.Is(err, domain.ErrRaffleNumberNotSold), errors.Is(err, domain.ErrWinningNumberNotSold):
		return http.StatusBadRequest, ErrorResponse{Message: err.Error(), Code: "invalid_raffle_number"}
	default:
		return http.StatusInternalServerError, ErrorResponse{Message: "internal server error", Code: "internal_error"}
	}
}

func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.Join(domain.ErrValidation, err)
	}
	return nil
}
