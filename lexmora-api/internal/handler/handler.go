package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ArminDashti/lexmora-api/internal/domain"
	"github.com/ArminDashti/lexmora-api/internal/repository"
	"github.com/ArminDashti/lexmora-api/internal/service"
)

type Handler struct {
	authService        *service.AuthService
	transformService   *service.TransformService
	historyService     *service.HistoryService
	statsService       *service.StatsService
	settingsService    *service.SettingsService
	instructionService *service.InstructionService
	quizService        *service.QuizService
	openRouter         *service.OpenRouterClient
	openCodeGo         *service.OpenCodeGoClient
}

func New(
	authService *service.AuthService,
	transformService *service.TransformService,
	historyService *service.HistoryService,
	statsService *service.StatsService,
	settingsService *service.SettingsService,
	instructionService *service.InstructionService,
	quizService *service.QuizService,
	openRouter *service.OpenRouterClient,
	openCodeGo *service.OpenCodeGoClient,
) *Handler {
	return &Handler{
		authService:        authService,
		transformService:   transformService,
		historyService:     historyService,
		statsService:       statsService,
		settingsService:    settingsService,
		instructionService: instructionService,
		quizService:        quizService,
		openRouter:         openRouter,
		openCodeGo:         openCodeGo,
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, domain.APIError{Error: err.Error(), Code: "NOT_FOUND"})
	case errors.Is(err, service.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, domain.APIError{Error: err.Error(), Code: "INVALID_CREDENTIALS"})
	default:
		if err != nil && strings.Contains(err.Error(), "not configured") {
			c.JSON(http.StatusBadRequest, domain.APIError{Error: err.Error(), Code: "VALIDATION_ERROR"})
			return
		}
		if err != nil && strings.Contains(err.Error(), "openrouter") {
			c.JSON(http.StatusBadGateway, domain.APIError{Error: err.Error(), Code: "OPENROUTER_ERROR"})
			return
		}
		if err != nil && strings.Contains(err.Error(), "gemini") {
			c.JSON(http.StatusBadGateway, domain.APIError{Error: err.Error(), Code: "GEMINI_ERROR"})
			return
		}
		if err != nil && strings.Contains(err.Error(), "opencode go") {
			c.JSON(http.StatusBadGateway, domain.APIError{Error: err.Error(), Code: "OPENCODE_GO_ERROR"})
			return
		}
		if err != nil && (strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "not found for key") || strings.Contains(err.Error(), "not enough")) {
			c.JSON(http.StatusBadRequest, domain.APIError{Error: err.Error(), Code: "VALIDATION_ERROR"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.APIError{Error: err.Error(), Code: "INTERNAL_ERROR"})
	}
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{Error: "invalid " + name, Code: "INVALID_ID"})
		return uuid.Nil, false
	}
	return id, true
}
