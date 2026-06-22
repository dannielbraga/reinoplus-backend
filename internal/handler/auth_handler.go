package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	jwtmanager "github.com/reinoplus/reinoplus/internal/auth/jwt"
	"github.com/reinoplus/reinoplus/internal/domain"
	"github.com/reinoplus/reinoplus/internal/httputil"
	"github.com/reinoplus/reinoplus/internal/middleware"
	"github.com/reinoplus/reinoplus/internal/textutil"
	authuc "github.com/reinoplus/reinoplus/internal/usecase/auth"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	service *authuc.Service
	jwt     *jwtmanager.Manager
	logger  *zap.Logger
}

func NewAuthHandler(service *authuc.Service, jwtManager *jwtmanager.Manager, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{service: service, jwt: jwtManager, logger: logger}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
	Address   string `json:"address"`
}

type updateProfileRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Password  string `json:"password,omitempty"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
	Address   string `json:"address"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type authUserResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Role     string  `json:"role"`
	MemberID *string `json:"member_id,omitempty"`
}

type profileResponse struct {
	User   authUserResponse `json:"user"`
	Member memberResponse   `json:"member"`
}

type loginResponse struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	User         authUserResponse `json:"user"`
}

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.Login)
	r.Post("/register", h.Register)
	r.Post("/refresh", h.Refresh)
}

func (h *AuthHandler) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/me", h.Me)
	r.Get("/profile", h.Profile)
	r.Put("/profile", h.UpdateProfile)
	r.Post("/logout", h.Logout)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	output, err := h.service.Login(r.Context(), authuc.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}, verifyPassword)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, loginResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		User:         toAuthUser(output.User),
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	birthDate, err := parseDate(req.BirthDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	output, err := h.service.Register(r.Context(), authuc.RegisterInput{
		Name:      req.Name,
		Email:     req.Email,
		Password:  req.Password,
		Phone:     req.Phone,
		BirthDate: birthDate,
		Address:   req.Address,
	}, hashPassword)
	if err != nil {
		if err == domain.ErrConflict {
			httputil.WriteJSON(w, http.StatusConflict, httputil.ErrorResponse{
				Message: "E-mail já cadastrado",
				Code:    "conflict",
			})
			return
		}
		httputil.WriteError(w, h.logger, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, loginResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		User:         toAuthUser(output.User),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	output, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, refreshResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Me(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toAuthUser(user))
}

func (h *AuthHandler) Profile(w http.ResponseWriter, r *http.Request) {
	output, err := h.service.GetProfile(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toProfileResponse(output))
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req updateProfileRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	birthDate, err := parseDate(req.BirthDate)
	if err != nil {
		httputil.WriteError(w, h.logger, err)
		return
	}

	output, err := h.service.UpdateProfile(r.Context(), authuc.UpdateProfileInput{
		UserID:    middleware.GetUserID(r.Context()),
		Name:      req.Name,
		Email:     req.Email,
		Password:  req.Password,
		Phone:     req.Phone,
		BirthDate: birthDate,
		Address:   req.Address,
	}, hashPassword)
	if err != nil {
		if err == domain.ErrConflict {
			httputil.WriteJSON(w, http.StatusConflict, httputil.ErrorResponse{
				Message: "E-mail já cadastrado",
				Code:    "conflict",
			})
			return
		}
		httputil.WriteError(w, h.logger, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, toProfileResponse(output))
}

func (h *AuthHandler) Logout(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func toAuthUser(user domain.User) authUserResponse {
	return authUserResponse{
		ID:       user.ID,
		Name:     textutil.TitleCaseName(user.Name),
		Email:    user.Email,
		Role:     string(user.Role),
		MemberID: user.MemberID,
	}
}

func toProfileResponse(output authuc.ProfileOutput) profileResponse {
	return profileResponse{
		User:   toAuthUser(output.User),
		Member: toMemberResponse(output.Member),
	}
}

func verifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
