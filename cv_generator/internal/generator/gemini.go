package generator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"cv_generator/internal/models"
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

// GenerateCV generates an HTML CV from resume data and options.
func (c *Client) GenerateCV(ctx context.Context, data *models.ResumeData, opts *models.GenerateCVRequest) (string, error) {
	start := time.Now()

	// Use defaults if not specified
	if opts == nil {
		defaultOpts := models.DefaultCVRequest()
		opts = &defaultOpts
	}

	prompt := BuildCVPrompt(data, opts)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	html := extractText(resp.Candidates[0].Content.Parts)

	// Clean up the response - remove any markdown code blocks if present
	html = cleanHTMLResponse(html)

	slog.Debug("CV generation completed",
		"duration_ms", time.Since(start).Milliseconds(),
		"style", opts.Style,
		"html_length", len(html),
	)

	return html, nil
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

// cleanHTMLResponse removes markdown code blocks and extracts pure HTML.
func cleanHTMLResponse(text string) string {
	// Remove markdown code block markers
	if strings.HasPrefix(text, "```html") {
		text = strings.TrimPrefix(text, "```html")
	}
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
	}
	if strings.HasSuffix(text, "```") {
		text = strings.TrimSuffix(text, "```")
	}

	text = strings.TrimSpace(text)

	// Ensure it starts with DOCTYPE
	if !strings.HasPrefix(text, "<!DOCTYPE") && !strings.HasPrefix(text, "<html") {
		// Try to find the HTML start
		if idx := strings.Index(text, "<!DOCTYPE"); idx > 0 {
			text = text[idx:]
		} else if idx := strings.Index(text, "<html"); idx > 0 {
			text = text[idx:]
		}
	}

	return text
}
