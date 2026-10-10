package service

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/ArminDashti/lexmora-api/internal/domain"
	"github.com/ArminDashti/lexmora-api/internal/repository"
)

type TransformRequest struct {
	Operation string `json:"operation"`
	Text      string `json:"text"`
	Text1     string `json:"text1"`
	Text2     string `json:"text2"`
	Direction string `json:"direction"`
	Mode      string `json:"mode"`
	Topic     string `json:"topic"`
	MovieName string `json:"movie_name"`
	Language  string `json:"language"`
	Style     string `json:"style"`
}

type TransformService struct {
	historyRepo     *repository.HistoryRepository
	settingsService *SettingsService
	instructionSvc  *InstructionService
	openRouter      *OpenRouterClient
	gemini          *GeminiClient
	openCodeGo      *OpenCodeGoClient
}

func NewTransformService(
	historyRepo *repository.HistoryRepository,
	settingsService *SettingsService,
	instructionSvc *InstructionService,
	openRouter *OpenRouterClient,
	gemini *GeminiClient,
	openCodeGo *OpenCodeGoClient,
) *TransformService {
	return &TransformService{
		historyRepo:     historyRepo,
		settingsService: settingsService,
		instructionSvc:  instructionSvc,
		openRouter:      openRouter,
		gemini:          gemini,
		openCodeGo:      openCodeGo,
	}
}

func (s *TransformService) Transform(ctx context.Context, req TransformRequest) (*domain.TransformResult, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	var inputText string
	if op == "compare" {
		text1 := strings.TrimSpace(req.Text1)
		text2 := strings.TrimSpace(req.Text2)
		if text1 == "" || text2 == "" {
			return nil, fmt.Errorf("text1 and text2 are required")
		}
		inputText = text1 + " vs " + text2
	} else {
		inputText = strings.TrimSpace(req.Text)
		if inputText == "" {
			return nil, fmt.Errorf("text is required")
		}
	}

	historyType, instructionKey, userText, metadata, err := s.resolveTransform(ctx, req, inputText)
	if err != nil {
		return nil, err
	}

	settings, err := s.settingsService.Get(ctx)
	if err != nil {
		return nil, err
	}

	systemPrompt, err := s.instructionSvc.BuildPrompt(ctx, instructionKey)
	if err != nil {
		return nil, err
	}

	result, err := s.complete(ctx, settings, systemPrompt, userText)
	if err != nil {
		return nil, err
	}

	record := domain.HistoryRecord{
		Type:           historyType,
		InputText:      inputText,
		ResultText:     result,
		Model:          settings.ModelName,
		InstructionKey: instructionKey,
		Metadata:       repository.MetadataJSON(metadata),
	}

	saved, err := s.historyRepo.Create(ctx, record)
	if err != nil {
		return nil, err
	}

	return &domain.TransformResult{
		ID:             saved.ID,
		Type:           saved.Type,
		TypeDisplay:    saved.TypeDisplay,
		InputText:      saved.InputText,
		ResultText:     saved.ResultText,
		Model:          saved.Model,
		InstructionKey: saved.InstructionKey,
		CreatedAt:      saved.CreatedAt,
		FormattedDate:  saved.FormattedDate,
	}, nil
}

func (s *TransformService) complete(ctx context.Context, settings *domain.AppSettings, systemPrompt, userText string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(settings.APIProvider))
	if provider == "" {
		provider = "openrouter"
	}
	switch provider {
	case "gemini":
		return s.gemini.Complete(ctx, settings.GeminiAPIKey, settings.ModelName, systemPrompt, userText)
	case "openrouter":
		return s.openRouter.Complete(ctx, settings.OpenRouterAPIKey, settings.ModelName, systemPrompt, userText)
	case "opencode_go":
		return s.openCodeGo.Complete(ctx, settings.OpenCodeGoAPIKey, settings.ModelName, systemPrompt, userText)
	default:
		return "", fmt.Errorf("invalid api_provider: %s", settings.APIProvider)
	}
}

func (s *TransformService) resolveTransform(ctx context.Context, req TransformRequest, text string) (domain.HistoryType, string, string, map[string]string, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	metadata := map[string]string{}

	switch op {
	case "translate":
		dir := normalizeDirection(req.Direction)
		mode := normalizeSlug(req.Mode)
		if mode == "" {
			return "", "", "", nil, fmt.Errorf("invalid translate mode: %s", req.Mode)
		}
		if dir != "english-persian" && dir != "persian-english" {
			return "", "", "", nil, fmt.Errorf("invalid translate direction: %s", req.Direction)
		}
		var historyType domain.HistoryType
		switch dir {
		case "english-persian":
			historyType = domain.HistoryTypeEnFa
		case "persian-english":
			historyType = domain.HistoryTypeFaEn
		}
		key, err := translateInstructionKey(dir, mode, req.Topic)
		if err != nil {
			return "", "", "", nil, err
		}
		if mode == "scientific" {
			topic := normalizeSlug(req.Topic)
			if topic == "" {
				topic = "general"
			}
			metadata["topic"] = topic
		}
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		if mode == "movie" {
			movie := strings.TrimSpace(req.MovieName)
			if movie != "" {
				metadata["movie_name"] = movie
				return historyType, key, fmt.Sprintf("Movie: %s\n\n%s", movie, text), metadata, nil
			}
		}
		return historyType, key, text, metadata, nil

	case "cursor":
		dir := normalizeDirection(req.Direction)
		mode := normalizeSlug(req.Mode)
		if dir != "persian-english" {
			return "", "", "", nil, fmt.Errorf("invalid cursor direction: %s", req.Direction)
		}
		if mode != "skill" && mode != "agent" {
			return "", "", "", nil, fmt.Errorf("invalid cursor mode: %s", req.Mode)
		}
		key := "cursor-persian-english-" + mode
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		return domain.HistoryTypeCursor, key, text, metadata, nil

	case "frontend":
		dir := normalizeDirection(req.Direction)
		mode := normalizeSlug(req.Mode)
		if mode != "for-agent" {
			return "", "", "", nil, fmt.Errorf("invalid frontend mode: %s", req.Mode)
		}
		if dir != "persian-english" && dir != "english-english" {
			return "", "", "", nil, fmt.Errorf("invalid frontend direction: %s", req.Direction)
		}
		key := "frontend-" + dir + "-for-agent"
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		return domain.HistoryTypeFrontend, key, text, metadata, nil

	case "simplify":
		key := "simplify-english-english"
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		return domain.HistoryTypeSimplify, key, text, metadata, nil

	case "term":
		lang := strings.ToLower(strings.TrimSpace(req.Language))
		style := normalizeSlug(req.Style)
		if style == "" {
			return "", "", "", nil, fmt.Errorf("invalid term style: %s", req.Style)
		}
		key := "term-english-english-" + style
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		switch lang {
		case "en":
			return domain.HistoryTypeTermEn, key, "Find an English term for this description:\n\n" + text, metadata, nil
		case "fa":
			return domain.HistoryTypeTermFa, key, "Find a Persian term for this description:\n\n" + text, metadata, nil
		case "":
			return domain.HistoryTypeTermEn, key, text, metadata, nil
		default:
			return "", "", "", nil, fmt.Errorf("invalid term language: %s", req.Language)
		}

	case "refine":
		style := normalizeSlug(req.Style)
		if style == "" {
			return "", "", "", nil, fmt.Errorf("invalid refine style: %s", req.Style)
		}
		key := "refine-english-english-" + style
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		return domain.HistoryTypeRefine, key, text, metadata, nil

	case "symptoms":
		key := "symptoms-english-english"
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		return domain.HistoryTypeSymptoms, key, text, metadata, nil

	case "compare":
		lang := strings.ToLower(strings.TrimSpace(req.Language))
		if lang == "" {
			lang = "en"
		}
		text1 := strings.TrimSpace(req.Text1)
		text2 := strings.TrimSpace(req.Text2)
		metadata["text1"] = text1
		metadata["text2"] = text2
		metadata["language"] = lang
		var key string
		switch lang {
		case "en":
			key = "compare-english-english"
		case "fa":
			key = "compare-persian-persian"
		default:
			return "", "", "", nil, fmt.Errorf("invalid compare language: %s", req.Language)
		}
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		userMsg := fmt.Sprintf("Compare these two words or phrases:\n\n1: %s\n2: %s", text1, text2)
		switch lang {
		case "en":
			return domain.HistoryTypeCompareEn, key, userMsg, metadata, nil
		case "fa":
			return domain.HistoryTypeCompareFa, key, userMsg, metadata, nil
		default:
			return "", "", "", nil, fmt.Errorf("invalid compare language: %s", req.Language)
		}

	case "grammar":
		lang := strings.ToLower(strings.TrimSpace(req.Language))
		if lang == "" {
			lang = "en"
		}
		metadata["language"] = lang
		var key string
		switch lang {
		case "en":
			key = "grammar-english-english"
		case "fa":
			key = "grammar-persian-persian"
		default:
			return "", "", "", nil, fmt.Errorf("invalid grammar language: %s", req.Language)
		}
		if err := s.requireInstruction(ctx, key); err != nil {
			return "", "", "", nil, err
		}
		switch lang {
		case "en":
			return domain.HistoryTypeGrammarEn, key, text, metadata, nil
		case "fa":
			return domain.HistoryTypeGrammarFa, key, text, metadata, nil
		default:
			return "", "", "", nil, fmt.Errorf("invalid grammar language: %s", req.Language)
		}

	default:
		return "", "", "", nil, fmt.Errorf("invalid operation: %s", req.Operation)
	}
}

func translateInstructionKey(dir, mode, topic string) (string, error) {
	if mode == "scientific" {
		t := normalizeSlug(topic)
		if t == "" {
			t = "general"
		}
		return "translate-" + dir + "-scientific-" + t, nil
	}
	return "translate-" + dir + "-" + mode, nil
}

func normalizeDirection(dir string) string {
	switch strings.ToLower(strings.TrimSpace(dir)) {
	case "en-fa", "english-persian":
		return "english-persian"
	case "fa-en", "persian-english":
		return "persian-english"
	case "en-en", "english-english":
		return "english-english"
	case "fa-fa", "persian-persian":
		return "persian-persian"
	default:
		return ""
	}
}

func directionLabel(dir string) string {
	switch dir {
	case "english-persian":
		return "English → Persian"
	case "persian-english":
		return "Persian → English"
	case "english-english":
		return "English → English"
	case "persian-persian":
		return "Persian → Persian"
	default:
		return titleFromSlug(dir)
	}
}

func (s *TransformService) requireInstruction(ctx context.Context, key string) error {
	if _, err := s.instructionSvc.Get(ctx, key); err != nil {
		return fmt.Errorf("instruction not found for key %s", key)
	}
	return nil
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func normalizeSlug(value string) string {
	s := strings.ToLower(strings.TrimSpace(value))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.Join(strings.Fields(s), "-")
	if !slugPattern.MatchString(s) {
		return ""
	}
	return s
}

func titleFromSlug(slug string) string {
	parts := strings.Split(slug, "-")
	for i, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		r[0] = unicode.ToUpper(r[0])
		parts[i] = string(r)
	}
	return strings.Join(parts, " ")
}

// --- Transform options catalog (derived from instruction keys) ---

type OptionItem struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ModeOption struct {
	Value  string       `json:"value"`
	Label  string       `json:"label"`
	Topics []OptionItem `json:"topics,omitempty"`
}

type DirectionOption struct {
	Value string       `json:"value"`
	Label string       `json:"label"`
	Modes []ModeOption `json:"modes"`
}

type OperationOption struct {
	Value      string            `json:"value"`
	Label      string            `json:"label"`
	Directions []DirectionOption `json:"directions,omitempty"`
	Styles     []OptionItem      `json:"styles,omitempty"`
	Languages  []OptionItem      `json:"languages,omitempty"`
}

type TransformOptions struct {
	Operations []OperationOption `json:"operations"`
}

var knownDirections = []string{
	"english-persian",
	"persian-english",
	"english-english",
	"persian-persian",
}

func (s *TransformService) GetOptions(ctx context.Context) (*TransformOptions, error) {
	items, err := s.instructionSvc.List(ctx)
	if err != nil {
		return nil, err
	}

	// dir -> mode -> topics (empty set means plain mode)
	translateDirs := map[string]map[string]map[string]struct{}{}
	cursorDirs := map[string]map[string]map[string]struct{}{}
	frontendDirs := map[string]map[string]map[string]struct{}{}
	refineStyles := map[string]struct{}{}
	termStyles := map[string]struct{}{}
	compareLangs := map[string]struct{}{}
	grammarLangs := map[string]struct{}{}
	hasSimplify := false
	hasSymptoms := false

	ensureMode := func(store map[string]map[string]map[string]struct{}, dir, mode, topic string) {
		if store[dir] == nil {
			store[dir] = map[string]map[string]struct{}{}
		}
		if store[dir][mode] == nil {
			store[dir][mode] = map[string]struct{}{}
		}
		if topic != "" {
			store[dir][mode][topic] = struct{}{}
		}
	}

	for _, item := range items {
		key := item.Key
		switch {
		case strings.HasPrefix(key, "translate-"):
			dir, mode, topic, ok := splitOpDirMode(strings.TrimPrefix(key, "translate-"))
			if ok {
				ensureMode(translateDirs, dir, mode, topic)
			}
		case strings.HasPrefix(key, "cursor-"):
			dir, mode, topic, ok := splitOpDirMode(strings.TrimPrefix(key, "cursor-"))
			if ok {
				ensureMode(cursorDirs, dir, mode, topic)
			}
		case strings.HasPrefix(key, "frontend-"):
			dir, mode, topic, ok := splitOpDirMode(strings.TrimPrefix(key, "frontend-"))
			if ok {
				ensureMode(frontendDirs, dir, mode, topic)
			}
		case strings.HasPrefix(key, "refine-english-english-"):
			style := strings.TrimPrefix(key, "refine-english-english-")
			if style != "" {
				refineStyles[style] = struct{}{}
			}
		case strings.HasPrefix(key, "term-english-english-"):
			style := strings.TrimPrefix(key, "term-english-english-")
			if style != "" {
				termStyles[style] = struct{}{}
			}
		case key == "compare-english-english":
			compareLangs["en"] = struct{}{}
		case key == "compare-persian-persian":
			compareLangs["fa"] = struct{}{}
		case key == "grammar-english-english":
			grammarLangs["en"] = struct{}{}
		case key == "grammar-persian-persian":
			grammarLangs["fa"] = struct{}{}
		case key == "simplify-english-english":
			hasSimplify = true
		case key == "symptoms-english-english":
			hasSymptoms = true
		}
	}

	ops := make([]OperationOption, 0, 8)

	if len(translateDirs) > 0 {
		ops = append(ops, OperationOption{
			Value:      "translate",
			Label:      "Translate",
			Directions: buildDirectionOptions(translateDirs),
		})
	}
	if len(cursorDirs) > 0 {
		ops = append(ops, OperationOption{
			Value:      "cursor",
			Label:      "Cursor",
			Directions: buildDirectionOptions(cursorDirs),
		})
	}
	if len(frontendDirs) > 0 {
		ops = append(ops, OperationOption{
			Value:      "frontend",
			Label:      "Frontend",
			Directions: buildDirectionOptions(frontendDirs),
		})
	}
	if hasSimplify {
		ops = append(ops, OperationOption{Value: "simplify", Label: "Simplify"})
	}
	if len(termStyles) > 0 {
		ops = append(ops, OperationOption{
			Value:  "term",
			Label:  "Term",
			Styles: sortedOptions(termStyles),
			Languages: []OptionItem{
				{Value: "en", Label: "English"},
				{Value: "fa", Label: "Persian"},
			},
		})
	}
	if len(refineStyles) > 0 {
		ops = append(ops, OperationOption{
			Value:  "refine",
			Label:  "Refine",
			Styles: sortedOptions(refineStyles),
		})
	}
	if hasSymptoms {
		ops = append(ops, OperationOption{Value: "symptoms", Label: "Symptoms"})
	}
	if len(compareLangs) > 0 {
		ops = append(ops, OperationOption{
			Value:     "compare",
			Label:     "Compare",
			Languages: sortedLanguageOptions(compareLangs),
		})
	}
	if len(grammarLangs) > 0 {
		ops = append(ops, OperationOption{
			Value:     "grammar",
			Label:     "Grammar",
			Languages: sortedLanguageOptions(grammarLangs),
		})
	}

	sortOperationsByLabel(ops)
	for i := range ops {
		sortDirectionsByLabel(ops[i].Directions)
	}

	return &TransformOptions{Operations: ops}, nil
}

func splitOpDirMode(rest string) (dir, mode, topic string, ok bool) {
	for _, d := range knownDirections {
		if rest == d {
			return d, "", "", true
		}
		prefix := d + "-"
		if !strings.HasPrefix(rest, prefix) {
			continue
		}
		modePart := strings.TrimPrefix(rest, prefix)
		if modePart == "" {
			return d, "", "", true
		}
		if strings.HasPrefix(modePart, "scientific-") {
			topic = strings.TrimPrefix(modePart, "scientific-")
			if topic == "" {
				return "", "", "", false
			}
			return d, "scientific", topic, true
		}
		return d, modePart, "", true
	}
	return "", "", "", false
}

func buildDirectionOptions(store map[string]map[string]map[string]struct{}) []DirectionOption {
	dirs := make([]DirectionOption, 0, len(store))
	for dir, modes := range store {
		modeKeys := make([]string, 0, len(modes))
		for m := range modes {
			modeKeys = append(modeKeys, m)
		}
		sort.Strings(modeKeys)
		modeOpts := make([]ModeOption, 0, len(modeKeys))
		for _, m := range modeKeys {
			opt := ModeOption{Value: m, Label: titleFromSlug(m)}
			topics := modes[m]
			if len(topics) > 0 || m == "scientific" {
				topicKeys := make([]string, 0, len(topics))
				for t := range topics {
					topicKeys = append(topicKeys, t)
				}
				sort.Strings(topicKeys)
				opt.Topics = make([]OptionItem, 0, len(topicKeys))
				for _, t := range topicKeys {
					opt.Topics = append(opt.Topics, OptionItem{Value: t, Label: titleFromSlug(t)})
				}
			}
			modeOpts = append(modeOpts, opt)
		}
		dirs = append(dirs, DirectionOption{
			Value: dir,
			Label: directionLabel(dir),
			Modes: modeOpts,
		})
	}
	return dirs
}

func sortOperationsByLabel(ops []OperationOption) {
	sort.Slice(ops, func(i, j int) bool {
		return strings.ToLower(ops[i].Label) < strings.ToLower(ops[j].Label)
	})
}

func sortDirectionsByLabel(dirs []DirectionOption) {
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Label) < strings.ToLower(dirs[j].Label)
	})
}

func sortedOptions(set map[string]struct{}) []OptionItem {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]OptionItem, 0, len(keys))
	for _, k := range keys {
		out = append(out, OptionItem{Value: k, Label: titleFromSlug(k)})
	}
	return out
}

func sortedLanguageOptions(set map[string]struct{}) []OptionItem {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]OptionItem, 0, len(keys))
	for _, k := range keys {
		label := titleFromSlug(k)
		switch k {
		case "en":
			label = "English"
		case "fa":
			label = "Persian"
		}
		out = append(out, OptionItem{Value: k, Label: label})
	}
	return out
}

// --- Create instruction from structured operation fields ---

type CreateInstructionRequest struct {
	Operation string `json:"operation"`
	Direction string `json:"direction"`
	Mode      string `json:"mode"`
	Topic     string `json:"topic"`
	Style     string `json:"style"`
	Language  string `json:"language"`
	Content   string `json:"content"`
}

func (s *InstructionService) CreateFromOperation(ctx context.Context, req CreateInstructionRequest) (*domain.Instruction, error) {
	key, err := buildInstructionKey(req)
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = defaultInstructionContent(key)
	}
	return s.repo.Upsert(ctx, key, content)
}

func buildInstructionKey(req CreateInstructionRequest) (string, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	switch op {
	case "translate":
		dir := normalizeDirection(req.Direction)
		mode := normalizeSlug(req.Mode)
		if mode == "" {
			return "", fmt.Errorf("invalid mode: %s", req.Mode)
		}
		if dir != "english-persian" && dir != "persian-english" {
			return "", fmt.Errorf("invalid direction: %s", req.Direction)
		}
		return translateInstructionKey(dir, mode, req.Topic)
	case "cursor":
		dir := normalizeDirection(req.Direction)
		mode := normalizeSlug(req.Mode)
		if dir != "persian-english" {
			return "", fmt.Errorf("invalid direction: %s", req.Direction)
		}
		if mode != "skill" && mode != "agent" {
			return "", fmt.Errorf("invalid mode: %s", req.Mode)
		}
		return "cursor-persian-english-" + mode, nil
	case "frontend":
		dir := normalizeDirection(req.Direction)
		mode := normalizeSlug(req.Mode)
		if mode != "for-agent" {
			return "", fmt.Errorf("invalid mode: %s", req.Mode)
		}
		if dir != "persian-english" && dir != "english-english" {
			return "", fmt.Errorf("invalid direction: %s", req.Direction)
		}
		return "frontend-" + dir + "-for-agent", nil
	case "refine":
		style := normalizeSlug(req.Style)
		if style == "" {
			style = normalizeSlug(req.Mode)
		}
		if style == "" {
			return "", fmt.Errorf("invalid style: %s", req.Style)
		}
		return "refine-english-english-" + style, nil
	case "term":
		style := normalizeSlug(req.Style)
		if style == "" {
			style = normalizeSlug(req.Mode)
		}
		if style == "" {
			return "", fmt.Errorf("invalid style: %s", req.Style)
		}
		return "term-english-english-" + style, nil
	case "simplify":
		return "simplify-english-english", nil
	case "symptoms":
		return "symptoms-english-english", nil
	case "compare":
		lang := strings.ToLower(strings.TrimSpace(req.Language))
		if lang == "" {
			lang = normalizeSlug(req.Mode)
		}
		switch lang {
		case "en":
			return "compare-english-english", nil
		case "fa":
			return "compare-persian-persian", nil
		default:
			return "", fmt.Errorf("invalid language: %s", req.Language)
		}
	case "grammar":
		lang := strings.ToLower(strings.TrimSpace(req.Language))
		if lang == "" {
			lang = "en"
		}
		switch lang {
		case "en":
			return "grammar-english-english", nil
		case "fa":
			return "grammar-persian-persian", nil
		default:
			return "", fmt.Errorf("invalid language: %s", req.Language)
		}
	default:
		return "", fmt.Errorf("invalid operation: %s", req.Operation)
	}
}

type HistoryService struct {
	repo *repository.HistoryRepository
}

func NewHistoryService(repo *repository.HistoryRepository) *HistoryService {
	return &HistoryService{repo: repo}
}

func (s *HistoryService) List(ctx context.Context, filter repository.HistoryListFilter) (*domain.HistoryListPage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	filter.Limit = limit

	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &domain.HistoryListPage{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: filter.Offset,
	}, nil
}

func (s *HistoryService) Get(ctx context.Context, id string) (*domain.HistoryRecord, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid id")
	}
	return s.repo.GetByID(ctx, uid)
}

func (s *HistoryService) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid id")
	}
	return s.repo.Delete(ctx, uid)
}

type StatsService struct {
	repo *repository.HistoryRepository
}

func NewStatsService(repo *repository.HistoryRepository) *StatsService {
	return &StatsService{repo: repo}
}

func (s *StatsService) Get(ctx context.Context) (*domain.StatsResponse, error) {
	now := time.Now()
	startToday := startOfDay(now)
	startYesterday := startToday.AddDate(0, 0, -1)
	startWeek := startToday.AddDate(0, 0, -6)
	startMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	today, err := s.repo.CountByPeriod(ctx, &startToday, nil)
	if err != nil {
		return nil, err
	}
	yesterday, err := s.repo.CountByPeriod(ctx, &startYesterday, &startToday)
	if err != nil {
		return nil, err
	}
	week, err := s.repo.CountByPeriod(ctx, &startWeek, nil)
	if err != nil {
		return nil, err
	}
	month, err := s.repo.CountByPeriod(ctx, &startMonth, nil)
	if err != nil {
		return nil, err
	}
	allTime, err := s.repo.CountByPeriod(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	return &domain.StatsResponse{
		Today:     today,
		Yesterday: yesterday,
		Week:      week,
		Month:     month,
		AllTime:   allTime,
	}, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
