package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"scrapper/internal/config"
)

// Client is an HTTP client for the AI job processing service.
type Client struct {
	baseURL    string
	httpClient *http.Client
	mode       config.AIProcessingMode
}

// ProcessResponse represents the response from the AI service.
type ProcessResponse struct {
	JobID      string `json:"job_id,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
	SavedToDB  bool   `json:"saved_to_db,omitempty"`
}

// NewClient creates a new AI service client.
func NewClient(baseURL string, mode config.AIProcessingMode) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // AI processing can take time
		},
		mode: mode,
	}
}

// ProcessJob sends a job to the AI service for processing.
// The processing mode (process, normalize, translate) is determined by the client configuration.
func (c *Client) ProcessJob(ctx context.Context, jobID string) (*ProcessResponse, error) {
	var endpoint string
	switch c.mode {
	case config.AIProcessingModeProcess:
		endpoint = fmt.Sprintf("%s/api/v1/process/%s", c.baseURL, jobID)
	case config.AIProcessingModeNormalize:
		endpoint = fmt.Sprintf("%s/api/v1/normalize/%s", c.baseURL, jobID)
	case config.AIProcessingModeTranslate:
		endpoint = fmt.Sprintf("%s/api/v1/translate/%s", c.baseURL, jobID)
	default:
		return nil, fmt.Errorf("unknown processing mode: %s", c.mode)
	}

	slog.Debug("calling AI service",
		"endpoint", endpoint,
		"job_id", jobID,
		"mode", c.mode,
	)

	// Create request with empty body (the service reads job from DB)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte("{}")))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call AI service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		slog.Warn("AI service returned non-OK status",
			"status", resp.StatusCode,
			"body", string(body),
			"job_id", jobID,
		)
		return nil, fmt.Errorf("AI service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ProcessResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	slog.Info("AI processing completed",
		"job_id", jobID,
		"mode", c.mode,
		"skipped", result.Skipped,
		"saved_to_db", result.SavedToDB,
	)

	return &result, nil
}

// HealthCheck checks if the AI service is available.
func (c *Client) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("AI service not reachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AI service health check failed: status %d", resp.StatusCode)
	}

	return nil
}
