package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/ArminDashti/lexmora-api/internal/config"
	"github.com/ArminDashti/lexmora-api/internal/db"
	"github.com/ArminDashti/lexmora-api/internal/handler"
	"github.com/ArminDashti/lexmora-api/internal/middleware"
	"github.com/ArminDashti/lexmora-api/internal/repository"
	"github.com/ArminDashti/lexmora-api/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := db.RunMigrations(cfg.DatabasePath); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	ctx := context.Background()
	sqlDB, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer sqlDB.Close()

	userRepo := repository.NewUserRepository(sqlDB)
	settingsRepo := repository.NewSettingsRepository(sqlDB)
	historyRepo := repository.NewHistoryRepository(sqlDB)
	instructionRepo := repository.NewInstructionRepository(sqlDB)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	settingsService := service.NewSettingsService(settingsRepo)
	instructionService := service.NewInstructionService(instructionRepo)
	openRouter, err := service.NewOpenRouterClient("", cfg.OpenRouterHTTPProxy)
	if err != nil {
		log.Fatalf("openrouter client: %v", err)
	}
	gemini, err := service.NewGeminiClient("", cfg.OpenRouterHTTPProxy)
	if err != nil {
		log.Fatalf("gemini client: %v", err)
	}
	openCodeGo, err := service.NewOpenCodeGoClient("", cfg.OpenRouterHTTPProxy)
	if err != nil {
		log.Fatalf("opencode go client: %v", err)
	}
	transformService := service.NewTransformService(historyRepo, settingsService, instructionService, openRouter, gemini, openCodeGo)
	historyService := service.NewHistoryService(historyRepo)
	statsService := service.NewStatsService(historyRepo)
	quizService := service.NewQuizService(historyRepo)

	if err := authService.EnsureDefaultUser(ctx, cfg.DefaultUsername, cfg.DefaultPassword); err != nil {
		log.Fatalf("default user: %v", err)
	}
	if err := instructionService.EnsureDefaults(ctx); err != nil {
		log.Fatalf("instructions: %v", err)
	}

	h := handler.New(authService, transformService, historyService, statsService, settingsService, instructionService, quizService, openRouter, openCodeGo)

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CorsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	router.GET("/api/v1/health", h.Health)
	router.POST("/api/v1/auth/login", h.Login)

	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuth(authService))
	{
		api.POST("/transform", h.Transform)
		api.GET("/transform/options", h.GetTransformOptions)
		api.GET("/history", h.ListHistory)
		api.GET("/history/:id", h.GetHistory)
		api.DELETE("/history/:id", h.DeleteHistory)
		api.POST("/quiz", h.CreateQuiz)
		api.GET("/stats", h.GetStats)

		api.GET("/instructions", h.ListInstructions)
		api.POST("/instructions", h.CreateInstruction)
		api.GET("/instructions/:key", h.GetInstruction)
		api.PUT("/instructions/:key", h.PutInstruction)

		api.GET("/settings", h.GetSettings)
		api.PATCH("/settings", h.PatchSettings)
		api.DELETE("/settings/data", h.ClearData)
		api.GET("/settings/models", h.SearchModels)
		api.GET("/settings/credits", h.GetCredits)
	}

	if cfg.StaticDir != "" {
		if _, err := os.Stat(cfg.StaticDir); err == nil {
			router.Static("/assets", cfg.StaticDir+"/assets")
			router.NoRoute(func(c *gin.Context) {
				if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				c.File(cfg.StaticDir + "/index.html")
			})
		}
	}

	srv := &http.Server{
		Addr:    "0.0.0.0:" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
