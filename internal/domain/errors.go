package domain

import "errors"

var (
	ErrNotFound                  = errors.New("resource not found")
	ErrUnauthorized              = errors.New("unauthorized")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrInvalidToken              = errors.New("invalid token")
	ErrConflict                  = errors.New("conflict")
	ErrValidation                = errors.New("validation error")
	ErrRaffleNumberAlreadySold   = errors.New("raffle number already sold")
	ErrRaffleNumberNotSold       = errors.New("raffle number is not sold")
	ErrRaffleNotActive           = errors.New("raffle is not active")
	ErrCampaignNotActive         = errors.New("campaign is not active")
	ErrInvalidRaffleNumber       = errors.New("invalid raffle number")
	ErrNoActiveRaffle            = errors.New("no active raffle")
	ErrWinningNumberNotSold      = errors.New("winning number is not sold")
	ErrRaffleHasSales            = errors.New("raffle has sold numbers")
	ErrForbidden                 = errors.New("forbidden")
)
