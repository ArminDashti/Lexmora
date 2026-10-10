package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type AppSettings struct {
	OpenRouterAPIKey  string    `json:"openrouter_api_key"`
	GeminiAPIKey      string    `json:"gemini_api_key"`
	OpenCodeGoAPIKey  string    `json:"opencode_go_api_key"`
	APIProvider       string    `json:"api_provider"`
	ModelName         string    `json:"model_name"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Instruction struct {
	Key       string    `json:"key"`
	Content   string    `json:"content"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HistoryType string

const (
	HistoryTypeSimplify  HistoryType = "simplify"
	HistoryTypeEnFa      HistoryType = "en_fa"
	HistoryTypeFaEn      HistoryType = "fa_en"
	HistoryTypeTermEn    HistoryType = "term_en"
	HistoryTypeTermFa    HistoryType = "term_fa"
	HistoryTypeRefine    HistoryType = "refine"
	HistoryTypeSymptoms  HistoryType = "symptoms"
	HistoryTypeCompareEn HistoryType = "compare_en"
	HistoryTypeCompareFa HistoryType = "compare_fa"
	HistoryTypeGrammarEn HistoryType = "grammar_en"
	HistoryTypeGrammarFa HistoryType = "grammar_fa"
	HistoryTypeCursor    HistoryType = "cursor"
	HistoryTypeFrontend  HistoryType = "frontend"
)

func (t HistoryType) DisplayName() string {
	switch t {
	case HistoryTypeSimplify:
		return "Simplify"
	case HistoryTypeEnFa:
		return "English-Persian"
	case HistoryTypeFaEn:
		return "Persian-English"
	case HistoryTypeTermEn:
		return "Term English"
	case HistoryTypeTermFa:
		return "Term Persian"
	case HistoryTypeRefine:
		return "Refine"
	case HistoryTypeSymptoms:
		return "Symptoms"
	case HistoryTypeCompareEn:
		return "Compare English"
	case HistoryTypeCompareFa:
		return "Compare Persian"
	case HistoryTypeGrammarEn:
		return "Grammar English"
	case HistoryTypeGrammarFa:
		return "Grammar Persian"
	case HistoryTypeCursor:
		return "Cursor"
	case HistoryTypeFrontend:
		return "Frontend"
	default:
		return string(t)
	}
}

type HistoryRecord struct {
	ID             uuid.UUID       `json:"id"`
	Type           HistoryType     `json:"type"`
	TypeDisplay    string          `json:"type_display"`
	InputText      string          `json:"input_text"`
	ResultText     string          `json:"result_text"`
	Model          string          `json:"model"`
	InstructionKey string          `json:"instruction_key"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	FormattedDate  string          `json:"formatted_date"`
	QuizShownCount int             `json:"-"`
}

type StatsBucket struct {
	Simplify  int `json:"simplify"`
	EnFa      int `json:"en_fa"`
	FaEn      int `json:"fa_en"`
	Term      int `json:"term"`
	Refine    int `json:"refine"`
	Symptoms  int `json:"symptoms"`
	Compare   int `json:"compare"`
	Grammar   int `json:"grammar"`
	Cursor    int `json:"cursor"`
	Frontend  int `json:"frontend"`
	Total     int `json:"total"`
}

type HistoryListPage struct {
	Items  []HistoryRecord `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type StatsResponse struct {
	Today     StatsBucket `json:"today"`
	Yesterday StatsBucket `json:"yesterday"`
	Week      StatsBucket `json:"week"`
	Month     StatsBucket `json:"month"`
	AllTime   StatsBucket `json:"all_time"`
}

type TransformResult struct {
	ID             uuid.UUID   `json:"id"`
	Type           HistoryType `json:"type"`
	TypeDisplay    string      `json:"type_display"`
	InputText      string      `json:"input_text"`
	ResultText     string      `json:"result_text"`
	Model          string      `json:"model"`
	InstructionKey string      `json:"instruction_key"`
	CreatedAt      time.Time   `json:"created_at"`
	FormattedDate  string      `json:"formatted_date"`
}

type APIError struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// ScientificTopics are required extras when mode is scientific.
var ScientificTopics = []string{
	"general",
	"aerospace",
	"biology",
	"chemistry",
	"physics",
	"medicine",
	"computer-science",
	"mathematics",
	"earth-science",
}

var InstructionKeys = buildInstructionKeys()

func buildInstructionKeys() []string {
	keys := []string{
		"translate-english-persian-general",
		"translate-english-persian-movie",
		"translate-english-persian-formal",
		"translate-english-persian-music",
	}
	for _, t := range ScientificTopics {
		keys = append(keys, "translate-english-persian-scientific-"+t)
	}
	keys = append(keys,
		"translate-persian-english-general",
		"translate-persian-english-formal",
	)
	for _, t := range ScientificTopics {
		keys = append(keys, "translate-persian-english-scientific-"+t)
	}
	keys = append(keys,
		"simplify-english-english",
		"refine-english-english-everyday",
		"refine-english-english-formal",
		"refine-english-english-slang",
		"symptoms-english-english",
		"term-english-english-everyday",
		"term-english-english-formal",
		"term-english-english-slang",
		"compare-english-english",
		"compare-persian-persian",
		"grammar-english-english",
		"grammar-persian-persian",
		"cursor-persian-english-skill",
		"cursor-persian-english-agent",
		"frontend-persian-english-for-agent",
		"frontend-english-english-for-agent",
	)
	return keys
}

// OldInstructionKeyMigrations maps legacy keys to the new naming scheme.
var OldInstructionKeyMigrations = map[string]string{
	"en-to-fa-general":     "translate-english-persian-general",
	"en-to-fa-movie":       "translate-english-persian-movie",
	"en-to-fa-formal":      "translate-english-persian-formal",
	"en-to-fa-scientific":  "translate-english-persian-scientific-general",
	"en-to-fa-music":       "translate-english-persian-music",
	"fa-to-en-general":     "translate-persian-english-general",
	"fa-to-en-formal":      "translate-persian-english-formal",
	"fa-to-en-scientific":  "translate-persian-english-scientific-general",
	"simplify-en":          "simplify-english-english",
	"refine-to-everyday":   "refine-english-english-everyday",
	"refine-to-formal":     "refine-english-english-formal",
	"refine-to-slang":      "refine-english-english-slang",
	"symptoms":             "symptoms-english-english",
	"term-for-everyday":    "term-english-english-everyday",
	"term-for-formal":      "term-english-english-formal",
	"term-for-slang":       "term-english-english-slang",
	"compare-en":           "compare-english-english",
	"compare-fa":           "compare-persian-persian",
	"grammar-en":           "grammar-english-english",
	"grammar-fa":           "grammar-persian-persian",
}

func FormatDateTime(t time.Time) string {
	return t.Format("2006:01:02 15:04")
}
