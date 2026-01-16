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

	"auth_service/internal/models"
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

// ParseCV extracts structured data from CV content.
func (c *Client) ParseCV(ctx context.Context, content, fileName string) (*models.ParsedCV, error) {
	start := time.Now()

	prompt := buildCVParsePrompt(content, fileName)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	text := extractText(resp.Candidates[0].Content.Parts)

	var result models.ParsedCV
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		text = extractJSONFromMarkdown(text)
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			return nil, fmt.Errorf("failed to parse Gemini response: %w", err)
		}
	}

	slog.Debug("CV parsing completed",
		"duration_ms", time.Since(start).Milliseconds(),
		"file_name", fileName,
	)

	return &result, nil
}

func buildCVParsePrompt(content, fileName string) string {
	return fmt.Sprintf(`Parse this Resume/CV and extract all relevant information into a structured format.

File Name: %s

CV Content:
%s

Extract the following information:

1. PERSONAL INFO: First name, last name, email, phone, location, LinkedIn URL, GitHub URL, website, professional headline and summary.

2. WORK EXPERIENCE: For each position, extract:
   - Job title
   - Company name
   - Location
   - Employment type (full_time, part_time, contract, internship, freelance, temporary)
   - Start date (YYYY-MM format if possible, or YYYY)
   - End date (YYYY-MM format, or "present"/"current" if still working there)
   - Whether it's current position
   - Job description
   - Key achievements (as array)

3. EDUCATION: For each entry, extract:
   - Institution name
   - Degree (e.g., Bachelor, Master, PhD)
   - Field of study
   - Start date
   - End date
   - Grade/GPA if mentioned
   - Activities (as array)

4. SKILLS: For each skill, extract:
   - Skill name
   - Category (programming, language, soft_skill, tool, framework, database, cloud, etc.)
   - Proficiency level if mentioned (beginner, intermediate, advanced, expert)

Rules:
- If information is not available, use empty string or empty array
- Dates should be in YYYY-MM or YYYY format
- Be thorough - extract ALL experiences, education, and skills mentioned
- For is_current, only set true if explicitly stated as current/present role

Return as JSON:
{
  "first_name": "string",
  "last_name": "string",
  "email": "string",
  "phone": "string",
  "location": "string",
  "headline": "string",
  "summary": "string",
  "linkedin_url": "string",
  "github_url": "string",
  "website": "string",
  "experiences": [
    {
      "title": "string",
      "company_name": "string",
      "location": "string",
      "employment_type": "string",
      "start_date": "YYYY-MM",
      "end_date": "YYYY-MM or empty if current",
      "is_current": boolean,
      "description": "string",
      "achievements": ["string"]
    }
  ],
  "education": [
    {
      "institution_name": "string",
      "degree": "string",
      "field_of_study": "string",
      "start_date": "YYYY-MM",
      "end_date": "YYYY-MM",
      "grade": "string",
      "activities": ["string"]
    }
  ],
  "skills": [
    {
      "name": "string",
      "category": "string",
      "proficiency_level": "string"
    }
  ]
}`, fileName, content)
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
