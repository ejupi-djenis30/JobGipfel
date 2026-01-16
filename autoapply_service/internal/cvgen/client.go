package cvgen

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the cv_generator service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new cv_generator client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for PDF generation
		},
	}
}

// GenerateCVRequest is the request to generate a CV.
type GenerateCVRequest struct {
	Style       string `json:"style"`
	ColorScheme string `json:"color_scheme"`
	Sections    struct {
		Summary        bool `json:"summary"`
		Experiences    bool `json:"experiences"`
		Education      bool `json:"education"`
		Skills         bool `json:"skills"`
		Certifications bool `json:"certifications"`
	} `json:"sections"`
}

// GenerateCVResult contains the generated CV data.
type GenerateCVResult struct {
	PDFBytes    []byte
	PDFBase64   string
	Style       string
	ColorScheme string
}

// GenerateCV generates a CV and returns the PDF bytes.
func (c *Client) GenerateCV(ctx context.Context, accessToken string, style, colorScheme string) (*GenerateCVResult, error) {
	url := fmt.Sprintf("%s/api/v1/cv/generate", c.baseURL)

	// Use defaults if not provided
	if style == "" {
		style = "modern"
	}
	if colorScheme == "" {
		colorScheme = "blue"
	}

	reqBody := GenerateCVRequest{
		Style:       style,
		ColorScheme: colorScheme,
	}
	reqBody.Sections.Summary = true
	reqBody.Sections.Experiences = true
	reqBody.Sections.Education = true
	reqBody.Sections.Skills = true
	reqBody.Sections.Certifications = true

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("cv_generator error: %s - %s", resp.Status, string(body))
	}

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read CV PDF: %w", err)
	}

	return &GenerateCVResult{
		PDFBytes:    pdfBytes,
		PDFBase64:   base64.StdEncoding.EncodeToString(pdfBytes),
		Style:       style,
		ColorScheme: colorScheme,
	}, nil
}

// HealthCheck checks if cv_generator is available.
func (c *Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cv_generator unhealthy: %s", resp.Status)
	}

	return nil
}
