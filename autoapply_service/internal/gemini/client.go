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

	"autoapply_service/internal/models"
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

// GenerateCoverLetter generates a personalized cover letter.
func (c *Client) GenerateCoverLetter(ctx context.Context, resume *models.ResumeData, jobTitle, companyName, jobDescription, customMessage, language string) (*models.CoverLetterResponse, error) {
	start := time.Now()

	if language == "" {
		language = "English"
	}

	prompt := buildCoverLetterPrompt(resume, jobTitle, companyName, jobDescription, customMessage, language)

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

	var result models.CoverLetterResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	slog.Debug("Cover letter generated",
		"duration_ms", time.Since(start).Milliseconds(),
		"job_title", jobTitle,
	)

	return &result, nil
}

// GenerateFormResponses generates responses for form fields.
func (c *Client) GenerateFormResponses(ctx context.Context, resume *models.ResumeData, jobTitle, companyName, jobDescription string, fields []models.FormField) ([]models.FormResponse, error) {
	start := time.Now()

	prompt := buildFormResponsePrompt(resume, jobTitle, companyName, jobDescription, fields)

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

	var result []models.FormResponse
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		// Try wrapping in array
		var single models.FormResponse
		if err2 := json.Unmarshal([]byte(text), &single); err2 == nil {
			result = []models.FormResponse{single}
		} else {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	slog.Debug("Form responses generated",
		"duration_ms", time.Since(start).Milliseconds(),
		"field_count", len(fields),
	)

	return result, nil
}

func buildCoverLetterPrompt(resume *models.ResumeData, jobTitle, companyName, jobDescription, customMessage, language string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`Generate a professional cover letter for a job application in %s.

JOB DETAILS:
- Title: %s
- Company: %s
- Description:
%s

`, language, jobTitle, companyName, jobDescription))

	// Add resume data
	sb.WriteString("CANDIDATE INFORMATION:\n")
	if resume.Profile != nil {
		if resume.Profile.FirstName != nil || resume.Profile.LastName != nil {
			name := ""
			if resume.Profile.FirstName != nil {
				name = *resume.Profile.FirstName
			}
			if resume.Profile.LastName != nil {
				name += " " + *resume.Profile.LastName
			}
			sb.WriteString(fmt.Sprintf("- Name: %s\n", strings.TrimSpace(name)))
		}
		if resume.Profile.Headline != nil {
			sb.WriteString(fmt.Sprintf("- Professional Title: %s\n", *resume.Profile.Headline))
		}
		if resume.Profile.Summary != nil {
			sb.WriteString(fmt.Sprintf("- Summary: %s\n", *resume.Profile.Summary))
		}
	}

	// Top experiences
	if len(resume.Experiences) > 0 {
		sb.WriteString("\nRECENT EXPERIENCE:\n")
		max := 3
		if max > len(resume.Experiences) {
			max = len(resume.Experiences)
		}
		for i := 0; i < max; i++ {
			exp := resume.Experiences[i]
			sb.WriteString(fmt.Sprintf("- %s at %s\n", exp.Title, exp.CompanyName))
		}
	}

	// Top skills
	if len(resume.Skills) > 0 {
		sb.WriteString("\nKEY SKILLS:\n")
		skills := []string{}
		for i := 0; i < len(resume.Skills) && i < 10; i++ {
			skills = append(skills, resume.Skills[i].Name)
		}
		sb.WriteString("- " + strings.Join(skills, ", ") + "\n")
	}

	if customMessage != "" {
		sb.WriteString(fmt.Sprintf("\nADDITIONAL INSTRUCTIONS: %s\n", customMessage))
	}

	sb.WriteString(`
REQUIREMENTS:
1. Professional, concise, and tailored to this specific job
2. Highlight relevant experience and skills
3. Show enthusiasm for the role and company
4. Keep it to 3-4 paragraphs maximum
5. Do NOT include placeholder text like [Your Name], use the actual name

Return JSON:
{
  "subject": "Email subject line for the application",
  "cover_letter": "The full cover letter text"
}`)

	return sb.String()
}

func buildFormResponsePrompt(resume *models.ResumeData, jobTitle, companyName, jobDescription string, fields []models.FormField) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`Generate appropriate responses for a job application form.

JOB: %s at %s
DESCRIPTION: %s

`, jobTitle, companyName, jobDescription))

	// Add resume data
	sb.WriteString("CANDIDATE INFO:\n")
	if resume.Profile != nil {
		if resume.Profile.FirstName != nil {
			sb.WriteString(fmt.Sprintf("- First Name: %s\n", *resume.Profile.FirstName))
		}
		if resume.Profile.LastName != nil {
			sb.WriteString(fmt.Sprintf("- Last Name: %s\n", *resume.Profile.LastName))
		}
		if resume.Profile.Phone != nil {
			sb.WriteString(fmt.Sprintf("- Phone: %s\n", *resume.Profile.Phone))
		}
		if resume.Profile.City != nil {
			sb.WriteString(fmt.Sprintf("- City: %s\n", *resume.Profile.City))
		}
		if resume.Profile.LinkedInURL != nil {
			sb.WriteString(fmt.Sprintf("- LinkedIn: %s\n", *resume.Profile.LinkedInURL))
		}
	}
	sb.WriteString(fmt.Sprintf("- Email: %s\n", resume.User.Email))

	sb.WriteString("\nFORM FIELDS TO FILL:\n")
	for i, field := range fields {
		sb.WriteString(fmt.Sprintf("%d. Field: %s (type: %s, label: %s", i+1, field.Name, field.Type, field.Label))
		if field.Required {
			sb.WriteString(", REQUIRED")
		}
		if len(field.Options) > 0 {
			sb.WriteString(fmt.Sprintf(", options: %v", field.Options))
		}
		sb.WriteString(")\n")
	}

	sb.WriteString(`
Generate appropriate values for each field. Use the candidate's actual information.
For questions like "Why do you want this job?", create a brief, compelling response.
For salary expectations, use a reasonable range if not specified.
For select fields, choose the most appropriate option from the provided options.

Return JSON array:
[
  {"field_name": "fieldName1", "value": "response1"},
  {"field_name": "fieldName2", "value": "response2"}
]`)

	return sb.String()
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
	// Remove markdown code blocks
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
