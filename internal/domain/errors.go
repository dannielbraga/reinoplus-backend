package domain

import "errors"

var (
	ErrNotFound                 = errors.New("recurso não encontrado")
	ErrUnauthorized             = errors.New("não autorizado")
	ErrInvalidCredentials       = errors.New("e-mail ou senha inválidos")
	ErrInvalidToken             = errors.New("sessão inválida ou expirada")
	ErrConflict                 = errors.New("conflito de dados")
	ErrValidation               = errors.New("dados inválidos")
	ErrRaffleNumberAlreadySold  = errors.New("este número da rifa já foi vendido")
	ErrRaffleNumberNotSold      = errors.New("este número da rifa não está vendido")
	ErrRaffleNotActive          = errors.New("esta rifa não está ativa")
	ErrCampaignHasContributions = errors.New("não é possível excluir campanha com contribuições")
	ErrCampaignNotActive        = errors.New("esta campanha não está ativa")
	ErrInvalidRaffleNumber      = errors.New("número da rifa inválido")
	ErrNoActiveRaffle           = errors.New("nenhuma rifa ativa no momento")
	ErrWinningNumberNotSold     = errors.New("o número sorteado não está vendido")
	ErrRaffleHasSales           = errors.New("não é possível excluir rifa com números vendidos")
	ErrForbidden                = errors.New("você não tem permissão para esta ação")
)
