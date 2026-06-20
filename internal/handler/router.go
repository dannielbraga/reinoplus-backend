package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	jwtmanager "github.com/reinoplus/reinoplus/internal/auth/jwt"
	"github.com/reinoplus/reinoplus/internal/httputil"
	"github.com/reinoplus/reinoplus/internal/middleware"
	"github.com/reinoplus/reinoplus/internal/usecase/auth"
	"github.com/reinoplus/reinoplus/internal/usecase/campaign"
	"github.com/reinoplus/reinoplus/internal/usecase/member"
	"github.com/reinoplus/reinoplus/internal/usecase/raffle"
	"go.uber.org/zap"
)

type Dependencies struct {
	Logger          *zap.Logger
	JWTManager      *jwtmanager.Manager
	AuthService     *auth.Service
	MemberService   *member.Service
	CampaignService *campaign.Service
	RaffleService   *raffle.Service
}

func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recovery(deps.Logger))
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.CORS([]string{"*"}))

	authHandler := NewAuthHandler(deps.AuthService, deps.JWTManager, deps.Logger)
	memberHandler := NewMemberHandler(deps.MemberService, deps.Logger)
	campaignHandler := NewCampaignHandler(deps.CampaignService, deps.Logger)
	raffleHandler := NewRaffleHandler(deps.RaffleService, deps.Logger)
	noticeHandler := NewNoticeHandler(deps.Logger)

	r.Route("/api", func(api chi.Router) {
		api.Route("/auth", func(authRoutes chi.Router) {
			authHandler.RegisterRoutes(authRoutes)
			authRoutes.Group(func(protected chi.Router) {
				protected.Use(middleware.Auth(deps.JWTManager, deps.Logger))
				authHandler.RegisterProtectedRoutes(protected)
			})
		})

		api.Group(func(protected chi.Router) {
			protected.Use(middleware.Auth(deps.JWTManager, deps.Logger))

			protected.Route("/members", func(members chi.Router) {
				memberHandler.RegisterReadRoutes(members)
				members.Group(func(admin chi.Router) {
					admin.Use(middleware.RequireAdmin(deps.Logger))
					memberHandler.RegisterWriteRoutes(admin)
				})
			})

			protected.Route("/campaigns", func(campaigns chi.Router) {
				campaignHandler.RegisterReadRoutes(campaigns)
				campaigns.Group(func(admin chi.Router) {
					admin.Use(middleware.RequireAdmin(deps.Logger))
					campaignHandler.RegisterWriteRoutes(admin)
				})
			})

			protected.Route("/raffles", func(raffles chi.Router) {
				raffleHandler.RegisterReadRoutes(raffles)
				raffles.Group(func(all chi.Router) {
					raffleHandler.RegisterSellRoutes(all)
				})
				raffles.Group(func(admin chi.Router) {
					admin.Use(middleware.RequireAdmin(deps.Logger))
					raffleHandler.RegisterWriteRoutes(admin)
				})
			})

			protected.Route("/notices", noticeHandler.RegisterRoutes)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return r
}
