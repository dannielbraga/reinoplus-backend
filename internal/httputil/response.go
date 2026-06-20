package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/reinoplus/reinoplus/internal/domain"
	"go.uber.org/zap"
)

const internalErrorMessage = "erro interno do servidor"

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
		response.Message = internalErrorMessage
	}
	WriteJSON(w, status, response)
}

func MapDomainError(err error) (int, ErrorResponse) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, ErrorResponse{Message: domain.ErrNotFound.Error(), Code: "not_found"}
	case errors.Is(err, domain.ErrNoActiveRaffle):
		return http.StatusNotFound, ErrorResponse{Message: domain.ErrNoActiveRaffle.Error(), Code: "not_found"}
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, ErrorResponse{Message: domain.ErrInvalidCredentials.Error(), Code: "invalid_credentials"}
	case errors.Is(err, domain.ErrInvalidToken):
		return http.StatusUnauthorized, ErrorResponse{Message: domain.ErrInvalidToken.Error(), Code: "invalid_token"}
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, ErrorResponse{Message: domain.ErrUnauthorized.Error(), Code: "unauthorized"}
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, ErrorResponse{Message: domain.ErrForbidden.Error(), Code: "forbidden"}
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, ErrorResponse{Message: domain.ErrValidation.Error(), Code: "validation_error"}
	case errors.Is(err, domain.ErrCampaignHasContributions):
		return http.StatusConflict, ErrorResponse{Message: domain.ErrCampaignHasContributions.Error(), Code: "conflict"}
	case errors.Is(err, domain.ErrRaffleHasSales):
		return http.StatusConflict, ErrorResponse{Message: domain.ErrRaffleHasSales.Error(), Code: "conflict"}
	case errors.Is(err, domain.ErrRaffleNumberAlreadySold):
		return http.StatusConflict, ErrorResponse{Message: domain.ErrRaffleNumberAlreadySold.Error(), Code: "conflict"}
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, ErrorResponse{Message: domain.ErrConflict.Error(), Code: "conflict"}
	case errors.Is(err, domain.ErrRaffleNotActive):
		return http.StatusBadRequest, ErrorResponse{Message: domain.ErrRaffleNotActive.Error(), Code: "invalid_state"}
	case errors.Is(err, domain.ErrCampaignNotActive):
		return http.StatusBadRequest, ErrorResponse{Message: domain.ErrCampaignNotActive.Error(), Code: "invalid_state"}
	case errors.Is(err, domain.ErrInvalidRaffleNumber):
		return http.StatusBadRequest, ErrorResponse{Message: domain.ErrInvalidRaffleNumber.Error(), Code: "invalid_raffle_number"}
	case errors.Is(err, domain.ErrRaffleNumberNotSold):
		return http.StatusBadRequest, ErrorResponse{Message: domain.ErrRaffleNumberNotSold.Error(), Code: "invalid_raffle_number"}
	case errors.Is(err, domain.ErrWinningNumberNotSold):
		return http.StatusBadRequest, ErrorResponse{Message: domain.ErrWinningNumberNotSold.Error(), Code: "invalid_raffle_number"}
	default:
		return http.StatusInternalServerError, ErrorResponse{Message: internalErrorMessage, Code: "internal_error"}
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
