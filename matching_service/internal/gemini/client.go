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

	"matching_service/internal/models"
)

type ClientConfig struct {
	APIKey         string
	Model          string
	EmbeddingModel string
	Temperature    float32
}

type Client struct {
	client         *genai.Client
	model          *genai.GenerativeModel
	embeddingModel *genai.EmbeddingModel
	config         ClientConfig
}

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

func (c *Client) Close() error {
	return c.client.Close()
}

// AnalyzeProfile analyzes a user profile and extracts insights.
func (c *Client) AnalyzeProfile(ctx context.Context, profile *models.UserProfile) (*models.ProfileAnalysis, error) {
	start := time.Now()

	skillNames := make([]string, len(profile.Skills))
	for i, s := range profile.Skills {
		skillNames[i] = s.Name
	}

	expDescriptions := make([]string, len(profile.Experiences))
	for i, e := range profile.Experiences {
		expDescriptions[i] = fmt.Sprintf("%s at %s: %s", e.Title, e.CompanyName, e.Description)
	}

	prompt := fmt.Sprintf(`Analyze this professional profile and extract insights.

Profile:
- Name: %s %s
- Headline: %s
- Summary: %s
- Skills: %s
- Experience: %s

Return JSON:
{
  "skills_extracted": ["skill1", "skill2"],
  "experience_years": 5,
  "seniority_level": "mid",
  "industries": ["tech", "finance"],
  "suggested_skills": ["skill to learn"],
  "profile_strength": 75,
  "improvement_tips": ["tip1", "tip2"]
}

seniority_level: junior (0-2 years), mid (3-6 years), senior (7+ years)
profile_strength: 0-100 based on completeness and quality
Return ONLY valid JSON.`,
		profile.FirstName, profile.LastName,
		profile.Headline, profile.Summary,
		strings.Join(skillNames, ", "),
		strings.Join(expDescriptions, "; "),
	)

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

	var analysis models.ProfileAnalysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		return &models.ProfileAnalysis{
			SkillsExtracted: skillNames,
			ProfileStrength: 50,
		}, nil
	}

	slog.Debug("Profile analyzed", "duration_ms", time.Since(start).Milliseconds())
	return &analysis, nil
}

// ScoreJobMatch uses AI to score how well a job matches a profile.
func (c *Client) ScoreJobMatch(ctx context.Context, profile *models.UserProfile, jobTitle, jobDescription string) (*models.JobScoreResponse, error) {
	start := time.Now()

	skillNames := make([]string, len(profile.Skills))
	for i, s := range profile.Skills {
		skillNames[i] = s.Name
	}

	prompt := fmt.Sprintf(`Score how well this candidate matches this job.

Candidate Profile:
- Headline: %s
- Skills: %s
- Experience Summary: %s

Job:
- Title: %s
- Description: %s

Return JSON:
{
  "match_score": 75,
  "skills_match": 80,
  "experience_match": 70,
  "matched_skills": ["Python", "React"],
  "missing_skills": ["Kubernetes"],
  "strengths": ["Strong Python experience"],
  "gaps": ["No cloud experience"],
  "recommendations": ["Consider getting AWS certification"]
}

All scores 0-100. Return ONLY valid JSON.`,
		profile.Headline,
		strings.Join(skillNames, ", "),
		profile.Summary,
		jobTitle, jobDescription,
	)

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

	var score models.JobScoreResponse
	if err := json.Unmarshal([]byte(text), &score); err != nil {
		return nil, fmt.Errorf("failed to parse score: %w", err)
	}

	slog.Debug("Job scored", "duration_ms", time.Since(start).Milliseconds(), "score", score.MatchScore)
	return &score, nil
}

// GenerateEmbedding generates a vector embedding for text.
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	resp, err := c.embeddingModel.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("embedding error: %w", err)
	}

	if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return resp.Embedding.Values, nil
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
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
