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

	"job_search/internal/models"
)

// ClientConfig holds Gemini client configuration.
type ClientConfig struct {
	APIKey         string
	Model          string
	EmbeddingModel string
	Temperature    float32
}

// Client wraps the Gemini API client.
type Client struct {
	client         *genai.Client
	model          *genai.GenerativeModel
	embeddingModel *genai.EmbeddingModel
	config         ClientConfig
}

// NewClient creates a new Gemini client.
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	if cfg.Model == "" {
		cfg.Model = "gemini-2.0-flash"
	}
	if cfg.EmbeddingModel == "" {
		cfg.EmbeddingModel = "text-embedding-004"
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.APIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	model := client.GenerativeModel(cfg.Model)
	model.SetTemperature(cfg.Temperature)

	embeddingModel := client.EmbeddingModel(cfg.EmbeddingModel)

	return &Client{
		client:         client,
		model:          model,
		embeddingModel: embeddingModel,
		config:         cfg,
	}, nil
}

// Close closes the Gemini client.
func (c *Client) Close() error {
	return c.client.Close()
}

// ParseNaturalLanguageQuery uses AI to understand a natural language search query.
func (c *Client) ParseNaturalLanguageQuery(ctx context.Context, query string) (*models.AIQueryResult, error) {
	start := time.Now()

	prompt := fmt.Sprintf(`Analyze this job search query and extract structured information.

Query: "%s"

Return JSON:
{
  "keywords": ["keyword1", "keyword2"],
  "skills": ["skill1", "skill2"],
  "job_titles": ["title1", "title2"],
  "locations": ["location1"],
  "intent": "search",
  "expanded_query": "expanded version of the query with synonyms"
}

Intent can be: "search" (looking for jobs), "filter" (refining results), "recommendation" (want suggestions)
If no specific information is found for a field, return an empty array.
Return ONLY valid JSON, no markdown.`, query)

	c.model.ResponseMIMEType = "application/json"
	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := extractText(resp.Candidates[0].Content.Parts)
	text = cleanJSON(text)

	var result models.AIQueryResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		// Return basic result on parse error
		return &models.AIQueryResult{
			Keywords:      []string{query},
			ExpandedQuery: query,
			Intent:        "search",
		}, nil
	}

	slog.Debug("Query parsed",
		"duration_ms", time.Since(start).Milliseconds(),
		"query", query,
		"keywords", result.Keywords,
	)

	return &result, nil
}

// GenerateEmbedding generates a vector embedding for text.
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	start := time.Now()

	resp, err := c.embeddingModel.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("embedding error: %w", err)
	}

	if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	slog.Debug("Embedding generated",
		"duration_ms", time.Since(start).Milliseconds(),
		"dimensions", len(resp.Embedding.Values),
	)

	return resp.Embedding.Values, nil
}

// GenerateEmbeddingBatch generates embeddings for multiple texts.
func (c *Client) GenerateEmbeddingBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))

	for i, text := range texts {
		emb, err := c.GenerateEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}

	return embeddings, nil
}

func extractText(parts []genai.Part) string {
	var texts []string
	for _, part := range parts {
		if text, ok := part.(genai.Text); ok {
			texts = append(texts, string(text))
		}
	}
	return strings.Join(texts, "")
}

func cleanJSON(text string) string {
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
	}
	if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
	}
	if strings.HasSuffix(text, "```") {
		text = strings.TrimSuffix(text, "```")
	}
	return strings.TrimSpace(text)
}
