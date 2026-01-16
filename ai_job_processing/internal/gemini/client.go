package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"ai_job_processing/internal/models"
)

// ClientConfig holds Gemini client configuration.
type ClientConfig struct {
	APIKey      string
	Model       string
	Temperature float32
}

// Client wraps the Gemini API client.
type Client struct {
	client *genai.Client
	model  *genai.GenerativeModel
	config ClientConfig
}

// NewClient creates a new Gemini client.
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	if cfg.Model == "" {
		cfg.Model = "gemini-2.0-flash"
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	model := client.GenerativeModel(cfg.Model)
	model.SetTemperature(cfg.Temperature)

	// Configure for structured output
	model.ResponseMIMEType = "application/json"

	return &Client{
		client: client,
		model:  model,
		config: cfg,
	}, nil
}

// Close closes the Gemini client.
func (c *Client) Close() error {
	return c.client.Close()
}

// NormalizeJobDescription extracts structured sections from a job description.
func (c *Client) NormalizeJobDescription(ctx context.Context, title, description, sourceLanguage string) (*models.NormalizedContent, error) {
	start := time.Now()

	prompt := buildNormalizationPrompt(title, description, sourceLanguage)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := extractText(resp.Candidates[0].Content.Parts)

	var result struct {
		Tasks        string `json:"tasks"`
		Requirements string `json:"requirements"`
		Offer        string `json:"offer"`
	}

	if err := json.Unmarshal([]byte(text), &result); err != nil {
		text = extractJSONFromMarkdown(text)
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
		}
	}

	slog.Debug("normalization completed",
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.NormalizedContent{
		Tasks:        strings.TrimSpace(result.Tasks),
		Requirements: strings.TrimSpace(result.Requirements),
		Offer:        strings.TrimSpace(result.Offer),
	}, nil
}

// TranslateNormalizedContent translates normalized content to a target language.
func (c *Client) TranslateNormalizedContent(ctx context.Context, title string, normalized *models.NormalizedContent, sourceLanguage, targetLanguage string) (*models.TranslatedContent, error) {
	start := time.Now()

	prompt := buildTranslationPrompt(title, normalized, sourceLanguage, targetLanguage)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := extractText(resp.Candidates[0].Content.Parts)

	var result struct {
		Title        string `json:"title"`
		Tasks        string `json:"tasks"`
		Requirements string `json:"requirements"`
		Offer        string `json:"offer"`
	}

	if err := json.Unmarshal([]byte(text), &result); err != nil {
		text = extractJSONFromMarkdown(text)
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			return nil, fmt.Errorf("failed to parse translation response: %w", err)
		}
	}

	translated := &models.TranslatedContent{
		Language:     targetLanguage,
		Title:        strings.TrimSpace(result.Title),
		Tasks:        strings.TrimSpace(result.Tasks),
		Requirements: strings.TrimSpace(result.Requirements),
		Offer:        strings.TrimSpace(result.Offer),
	}

	// Build combined description using translated sections
	translatedNormalized := &models.NormalizedContent{
		Tasks:        translated.Tasks,
		Requirements: translated.Requirements,
		Offer:        translated.Offer,
	}
	translated.Description = translatedNormalized.BuildDescription(targetLanguage)

	slog.Debug("normalized translation completed",
		"target_language", targetLanguage,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return translated, nil
}

// TranslateRawDescription translates a raw job description (without normalization).
func (c *Client) TranslateRawDescription(ctx context.Context, title, description, sourceLanguage, targetLanguage string) (*models.TranslatedContent, error) {
	start := time.Now()

	prompt := buildRawTranslationPrompt(title, description, sourceLanguage, targetLanguage)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := extractText(resp.Candidates[0].Content.Parts)

	var result struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal([]byte(text), &result); err != nil {
		text = extractJSONFromMarkdown(text)
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			return nil, fmt.Errorf("failed to parse translation response: %w", err)
		}
	}

	slog.Debug("raw translation completed",
		"target_language", targetLanguage,
		"duration_ms", time.Since(start).Milliseconds(),
	)

	return &models.TranslatedContent{
		Language:    targetLanguage,
		Title:       strings.TrimSpace(result.Title),
		Description: strings.TrimSpace(result.Description),
	}, nil
}

// TranslateMultipleNormalized translates normalized content to multiple languages.
func (c *Client) TranslateMultipleNormalized(ctx context.Context, title string, normalized *models.NormalizedContent, sourceLanguage string, targetLanguages []string) ([]models.TranslatedContent, error) {
	results := make([]models.TranslatedContent, 0, len(targetLanguages))

	for _, lang := range targetLanguages {
		// Skip if target is same as source
		if lang == sourceLanguage {
			results = append(results, models.TranslatedContent{
				Language:     lang,
				Title:        title,
				Tasks:        normalized.Tasks,
				Requirements: normalized.Requirements,
				Offer:        normalized.Offer,
				Description:  normalized.BuildDescription(lang),
			})
			continue
		}

		translated, err := c.TranslateNormalizedContent(ctx, title, normalized, sourceLanguage, lang)
		if err != nil {
			slog.Error("translation failed",
				"target_language", lang,
				"error", err,
			)
			continue
		}

		results = append(results, *translated)
	}

	return results, nil
}

// TranslateMultipleRaw translates raw description to multiple languages.
func (c *Client) TranslateMultipleRaw(ctx context.Context, title, description, sourceLanguage string, targetLanguages []string) ([]models.TranslatedContent, error) {
	results := make([]models.TranslatedContent, 0, len(targetLanguages))

	for _, lang := range targetLanguages {
		// Skip if target is same as source
		if lang == sourceLanguage {
			results = append(results, models.TranslatedContent{
				Language:    lang,
				Title:       title,
				Description: description,
			})
			continue
		}

		translated, err := c.TranslateRawDescription(ctx, title, description, sourceLanguage, lang)
		if err != nil {
			slog.Error("translation failed",
				"target_language", lang,
				"error", err,
			)
			continue
		}

		results = append(results, *translated)
	}

	return results, nil
}

// buildNormalizationPrompt creates the prompt for job description normalization.
func buildNormalizationPrompt(title, description, sourceLanguage string) string {
	langHint := ""
	if sourceLanguage != "" {
		langHint = fmt.Sprintf("The job description is in %s. ", sourceLanguage)
	}

	return fmt.Sprintf(`%sAnalyze this job posting and extract the content into three distinct sections.

Job Title: %s

Job Description:
%s

Extract and categorize the content into:

1. TASKS: What the employee will actually do day-to-day. Include responsibilities, activities, and work tasks.

2. REQUIREMENTS: Skills, experience, qualifications, education, and competencies needed. Include both hard and soft skills.

3. OFFER: What the company offers - salary, benefits, perks, work environment, career opportunities, work-life balance.

Rules:
- Remove marketing fluff and generic company descriptions
- Use bullet points (- item) for each point
- Be concise but complete
- Keep the output in the same language as the original
- If a section has no content, return an empty string

Return as JSON:
{
  "tasks": "- task 1\n- task 2...",
  "requirements": "- requirement 1\n- requirement 2...",
  "offer": "- benefit 1\n- benefit 2..."
}`, langHint, title, description)
}

// buildTranslationPrompt creates the prompt for translating normalized content.
func buildTranslationPrompt(title string, normalized *models.NormalizedContent, sourceLanguage, targetLanguage string) string {
	sourceName := getLanguageName(sourceLanguage)
	targetName := getLanguageName(targetLanguage)

	return fmt.Sprintf(`Translate this job content from %s to %s.

Title: %s

Tasks:
%s

Requirements:
%s

Offer:
%s

Rules:
- Maintain the bullet point format
- Preserve professional tone
- Keep technical terms where appropriate
- Translate section content accurately

Return as JSON:
{
  "title": "translated title",
  "tasks": "translated tasks with bullet points",
  "requirements": "translated requirements with bullet points",
  "offer": "translated offer with bullet points"
}`, sourceName, targetName, title, normalized.Tasks, normalized.Requirements, normalized.Offer)
}

// buildRawTranslationPrompt creates the prompt for translating raw descriptions.
func buildRawTranslationPrompt(title, description, sourceLanguage, targetLanguage string) string {
	sourceName := getLanguageName(sourceLanguage)
	targetName := getLanguageName(targetLanguage)

	return fmt.Sprintf(`Translate this job posting from %s to %s.

Title: %s

Description:
%s

Rules:
- Maintain the original formatting (bullet points, paragraphs, etc.)
- Preserve professional tone
- Keep technical terms where appropriate
- Translate accurately while keeping natural flow

Return as JSON:
{
  "title": "translated title",
  "description": "translated description"
}`, sourceName, targetName, title, description)
}

// extractText extracts text from Gemini response parts.
func extractText(parts []genai.Part) string {
	var texts []string
	for _, part := range parts {
		if text, ok := part.(genai.Text); ok {
			texts = append(texts, string(text))
		}
	}
	return strings.Join(texts, "")
}

// extractJSONFromMarkdown extracts JSON from markdown code blocks.
func extractJSONFromMarkdown(text string) string {
	if start := strings.Index(text, "```json"); start != -1 {
		text = text[start+7:]
		if end := strings.Index(text, "```"); end != -1 {
			return strings.TrimSpace(text[:end])
		}
	}
	if start := strings.Index(text, "```"); start != -1 {
		text = text[start+3:]
		if end := strings.Index(text, "```"); end != -1 {
			return strings.TrimSpace(text[:end])
		}
	}
	if start := strings.Index(text, "{"); start != -1 {
		if end := strings.LastIndex(text, "}"); end != -1 {
			return text[start : end+1]
		}
	}
	return text
}

// getLanguageName returns the full name of a language code.
func getLanguageName(code string) string {
	names := map[string]string{
		"de": "German",
		"fr": "French",
		"it": "Italian",
		"en": "English",
		"rm": "Romansh",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}
