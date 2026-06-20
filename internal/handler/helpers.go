package handler

import (
	"time"

	"github.com/reinoplus/reinoplus/internal/domain"
)

func parseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, domain.ErrValidation
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, domain.ErrValidation
	}
	return parsed, nil
}
