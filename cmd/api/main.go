package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/reinoplus/reinoplus/internal/auth/jwt"
	"github.com/reinoplus/reinoplus/internal/config"
	"github.com/reinoplus/reinoplus/internal/handler"
	"github.com/reinoplus/reinoplus/internal/repository/postgres"
	"github.com/reinoplus/reinoplus/internal/usecase/auth"
	"github.com/reinoplus/reinoplus/internal/usecase/campaign"
	"github.com/reinoplus/reinoplus/internal/usecase/member"
	"github.com/reinoplus/reinoplus/internal/usecase/raffle"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("load config", zap.Error(err))
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		logger.Fatal("connect database", zap.Error(err))
	}
	defer pool.Close()

	jwtManager, err := jwt.NewManager(
		cfg.JWT.PrivateKeyPath,
		cfg.JWT.PublicKeyPath,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)
	if err != nil {
		logger.Fatal("init jwt manager", zap.Error(err))
	}

	userRepo := postgres.NewUserRepository(pool)
	registerRepo := postgres.NewRegisterRepository(pool)
	memberRepo := postgres.NewMemberRepository(pool)
	campaignRepo := postgres.NewCampaignRepository(pool)
	raffleRepo := postgres.NewRaffleRepository(pool)

	authService := auth.NewService(userRepo, registerRepo, jwtManager)
	memberService := member.NewService(memberRepo)
	campaignService := campaign.NewService(campaignRepo, campaignRepo, memberRepo)
	raffleService := raffle.NewService(raffleRepo, raffleRepo, memberRepo)

	router := handler.NewRouter(handler.Dependencies{
		Logger:          logger,
		JWTManager:      jwtManager,
		AuthService:     authService,
		MemberService:   memberService,
		CampaignService: campaignService,
		RaffleService:   raffleService,
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("server starting", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", zap.Error(err))
	}
}
