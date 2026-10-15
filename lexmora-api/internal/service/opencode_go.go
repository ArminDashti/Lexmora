package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultOpenCodeGoBaseURL = "https://opencode.ai/zen/go/v1"

const opencodeGoUserAgent = "lexmora-api/1.0"

type OpenCodeGoClient struct {
	baseURL string
	client  *http.Client
}

func NewOpenCodeGoClient(baseURL, httpProxy string) (*OpenCodeGoClient, error) {
	if baseURL == "" {
		baseURL = defaultOpenCodeGoBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.TrimSpace(httpProxy) != "" {
		proxyURL, err := url.Parse(httpProxy)
		if err != nil {
			return nil, fmt.Errorf("opencode go http proxy: %w", err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return &OpenCodeGoClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
	}, nil
}

func (c *OpenCodeGoClient) Complete(ctx context.Context, apiKey, model, systemPrompt, userText string) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", errors.New("opencode go API key is not configured")
	}
	model = normalizeOpenCodeGoModelID(model)
	if model == "" {
		return "", errors.New("model is not configured")
	}

	kind, err := opencodeGoEndpointForModel(model)
	if err != nil {
		return "", err
	}

	sessionID := uuid.NewString()

	switch kind {
	case opencodeGoChat:
		return c.completeChat(ctx, apiKey, model, systemPrompt, userText, sessionID)
	case opencodeGoMessages:
		return c.completeMessages(ctx, apiKey, model, systemPrompt, userText, sessionID)
	case opencodeGoResponses:
		return c.completeResponses(ctx, apiKey, model, systemPrompt, userText, sessionID)
	default:
		return "", fmt.Errorf("opencode go: unsupported endpoint for model %q", model)
	}
}

func (c *OpenCodeGoClient) completeChat(ctx context.Context, apiKey, model, systemPrompt, userText, sessionID string) (string, error) {
	messages := []chatMessage{}
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: userText})

	body, err := json.Marshal(chatRequest{Model: model, Messages: messages})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	respBody, status, err := c.do(ctx, http.MethodPost, "/chat/completions", apiKey, sessionID, body)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("opencode go status %d: %s", status, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if chatResp.Error != nil {
		return "", fmt.Errorf("opencode go error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("opencode go returned no choices")
	}
	return trimModelFences(chatResp.Choices[0].Message.Content), nil
}

type anthropicMessagesRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []chatMessage `json:"messages"`
}

type anthropicMessagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *OpenCodeGoClient) completeMessages(ctx context.Context, apiKey, model, systemPrompt, userText, sessionID string) (string, error) {
	reqBody := anthropicMessagesRequest{
		Model:     model,
		MaxTokens: 8192,
		System:    strings.TrimSpace(systemPrompt),
		Messages:  []chatMessage{{Role: "user", Content: userText}},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	respBody, status, err := c.do(ctx, http.MethodPost, "/messages", apiKey, sessionID, body)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("opencode go status %d: %s", status, string(respBody))
	}

	var parsed anthropicMessagesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("opencode go error: %s", parsed.Error.Message)
	}
	var parts []string
	for _, block := range parsed.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("opencode go returned no message content")
	}
	return trimModelFences(strings.Join(parts, "\n")), nil
}

type responsesRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type responsesResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *OpenCodeGoClient) completeResponses(ctx context.Context, apiKey, model, systemPrompt, userText, sessionID string) (string, error) {
	input := userText
	if strings.TrimSpace(systemPrompt) != "" {
		input = systemPrompt + "\n\n" + userText
	}

	body, err := json.Marshal(responsesRequest{Model: model, Input: input})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	respBody, status, err := c.do(ctx, http.MethodPost, "/responses", apiKey, sessionID, body)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("opencode go status %d: %s", status, string(respBody))
	}

	var parsed responsesResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("opencode go error: %s", parsed.Error.Message)
	}
	if text := strings.TrimSpace(parsed.OutputText); text != "" {
		return trimModelFences(text), nil
	}
	var parts []string
	for _, item := range parsed.Output {
		for _, block := range item.Content {
			if strings.TrimSpace(block.Text) != "" {
				parts = append(parts, block.Text)
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("opencode go returned no response output")
	}
	return trimModelFences(strings.Join(parts, "\n")), nil
}

func trimModelFences(content string) string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	return strings.TrimSpace(content)
}

func (c *OpenCodeGoClient) ListModels(ctx context.Context, apiKey, q string, limit int) ([]OpenRouterModel, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("opencode go API key is not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	respBody, status, err := c.do(ctx, http.MethodGet, "/models", apiKey, uuid.NewString(), nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("opencode go status %d: %s", status, string(respBody))
	}

	var parsed modelsListResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal models: %w", err)
	}

	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]OpenRouterModel, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		id := normalizeOpenCodeGoModelID(m.ID)
		if id == "" {
			continue
		}
		name := m.Name
		if name == "" {
			name = id
		}
		if q != "" {
			hay := strings.ToLower(id + " " + name)
			if !strings.Contains(hay, q) {
				continue
			}
		}
		out = append(out, OpenRouterModel{
			ID:            id,
			Name:          name,
			ContextLength: m.ContextLength,
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (c *OpenCodeGoClient) do(ctx context.Context, method, path, apiKey, sessionID string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", opencodeGoUserAgent)
	if sessionID != "" {
		req.Header.Set("x-opencode-session", sessionID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("opencode go request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}
