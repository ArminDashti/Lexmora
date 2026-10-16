package service

import (
	"fmt"
	"strings"
)

// opencodeGoEndpointKind matches https://opencode.ai/v2/docs/console/go#endpoints
type opencodeGoEndpointKind string

const (
	opencodeGoChat      opencodeGoEndpointKind = "chat"
	opencodeGoMessages  opencodeGoEndpointKind = "messages"
	opencodeGoResponses opencodeGoEndpointKind = "responses"
)

var opencodeGoModelEndpoints = map[string]opencodeGoEndpointKind{
	"grok-4.7":                    opencodeGoResponses,
	"grok-4.6":                    opencodeGoResponses,
	"gpt-6-luna":                  opencodeGoResponses,
	"gpt-5.6-luna":                opencodeGoResponses,
	"muse-spark-1.3-contributor":  opencodeGoResponses,
	"muse-spark-1.2-contributor":  opencodeGoResponses,
	"claude-haiku-5-5":            opencodeGoMessages,
	"minimax-m3":                  opencodeGoMessages,
	"minimax-m2.7":                opencodeGoMessages,
	"minimax-m2.5":                opencodeGoMessages,
	"qwen3.8-max":                 opencodeGoMessages,
	"qwen3.8-flash":               opencodeGoMessages,
	"qwen3.7-plus":                opencodeGoMessages,
	"qwen3.7-max":                 opencodeGoMessages,
	"qwen3.6-plus":                opencodeGoMessages,
	"glm-5.3-flash":               opencodeGoChat,
	"glm-5.3":                     opencodeGoChat,
	"glm-5.2":                     opencodeGoChat,
	"glm-5.1":                     opencodeGoChat,
	"kimi-k3":                     opencodeGoChat,
	"kimi-k2.7-code":              opencodeGoChat,
	"kimi-k2.6":                   opencodeGoChat,
	"longcat-2.0":                 opencodeGoChat,
	"longcat-2.5-preview-free":    opencodeGoChat,
	"deepseek-v4.1-flash":         opencodeGoChat,
	"deepseek-v4-pro":             opencodeGoChat,
	"deepseek-v4-flash":           opencodeGoChat,
	"deepseek-v4-flash-vision-exp": opencodeGoChat,
	"mimo-v2.6-flash":             opencodeGoChat,
	"mimo-v2.6-pro":               opencodeGoChat,
	"mimo-v2.5":                   opencodeGoChat,
	"mimo-v2.5-pro":               opencodeGoChat,
	"hy4-preview":                 opencodeGoChat,
	"hy3":                         opencodeGoChat,
	"space-bunny":                 opencodeGoChat,
	"space-bunny-free":            opencodeGoChat,
	"step-5-preview":              opencodeGoChat,
	"beam-free":                   opencodeGoChat,
}

func normalizeOpenCodeGoModelID(model string) string {
	model = strings.TrimSpace(model)
	model = strings.TrimPrefix(model, "opencode-go/")
	return model
}

func opencodeGoEndpointForModel(model string) (opencodeGoEndpointKind, error) {
	id := normalizeOpenCodeGoModelID(model)
	if id == "" {
		return "", fmt.Errorf("model is not configured")
	}
	if kind, ok := opencodeGoModelEndpoints[id]; ok {
		return kind, nil
	}
	// Models not listed in the docs table (new or unlisted ones returned by /models)
	// fall back to the OpenAI-compatible chat/completions endpoint, which Go serves for most models.
	return opencodeGoChat, nil
}
