package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"scrapper/internal/models"
)

// Common browser User-Agent strings for rotation
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
}

// ClientConfig holds configuration for the scraper client.
type ClientConfig struct {
	Polite       bool
	DelayMinMs   int
	DelayMaxMs   int
	Timeout      time.Duration
	MaxRetries   int
	RetryDelayMs int
}

// DefaultClientConfig returns sensible defaults for the scraper client.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Polite:       true,
		DelayMinMs:   2000,
		DelayMaxMs:   5000,
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		RetryDelayMs: 1000,
	}
}

// Client handles HTTP communication with the job-room.ch API.
type Client struct {
	http    *http.Client
	config  ClientConfig
	BaseURL string
}

// NewClient creates a new scraper client.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
		config:  cfg,
		BaseURL: "https://www.job-room.ch/jobadservice/api/jobAdvertisements",
	}
}

// sleep pauses execution for a random duration if Polite mode is enabled.
func (c *Client) sleep() {
	if !c.config.Polite {
		return
	}

	minMs := c.config.DelayMinMs
	maxMs := c.config.DelayMaxMs
	if minMs <= 0 {
		minMs = 2000
	}
	if maxMs <= minMs {
		maxMs = minMs + 3000
	}

	delayMs := minMs + rand.Intn(maxMs-minMs+1)
	delay := time.Duration(delayMs) * time.Millisecond

	slog.Debug("polite mode: sleeping before request", "delay_ms", delayMs)
	time.Sleep(delay)
}

// randomUserAgent returns a random User-Agent string from the pool.
func randomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// setHeaders applies common headers to mimic a browser request.
func setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,de;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Origin", "https://www.job-room.ch")
	req.Header.Set("Referer", "https://www.job-room.ch/home/job-seeker")
}

// FetchJobsWithRequest retrieves a page of job listings using a ScrapeRequest for filtering.
// Uses POST request with JSON body as required by the job-room.ch API.
func (c *Client) FetchJobsWithRequest(req *models.ScrapeRequest, page int) ([]models.JobDetail, error) {
	c.sleep()

	// Build URL with query parameters for pagination only
	u, err := url.Parse(c.BaseURL + "/_search")
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("size", "20")
	q.Set("sort", "date_desc")
	q.Set("_ng", "ZW4=") // English
	u.RawQuery = q.Encode()

	// Build JSON request body
	body := req.BuildSearchBody()
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	slog.Debug("fetching job list with filters",
		"url", u.String(),
		"page", page,
		"keywords", req.Keywords,
		"cantons", req.Cantons,
	)

	return c.fetchJobsWithPOST(u.String(), jsonBody)
}

// fetchJobsWithPOST performs a POST request and parses the response.
func (c *Client) fetchJobsWithPOST(urlStr string, jsonBody []byte) ([]models.JobDetail, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(c.config.RetryDelayMs*attempt) * time.Millisecond
			slog.Debug("retrying request", "attempt", attempt, "delay_ms", delay.Milliseconds())
			time.Sleep(delay)
		}

		req, err := http.NewRequest(http.MethodPost, urlStr, bytes.NewReader(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		setHeaders(req)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))

			// Don't retry on 4xx errors (except 429)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
				return nil, lastErr
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// The API returns: [ { "jobAdvertisement": { ... } }, ... ]
		type listResponse []struct {
			JobAdvertisement models.JobDetail `json:"jobAdvertisement"`
		}

		var wrapper listResponse
		if err := json.Unmarshal(body, &wrapper); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Extract JobDetail objects from wrappers
		jobs := make([]models.JobDetail, len(wrapper))
		for i, item := range wrapper {
			jobs[i] = item.JobAdvertisement
			// Store the raw JSON for each job
			rawBytes, _ := json.Marshal(item.JobAdvertisement)
			jobs[i].RawData = string(rawBytes)
		}

		slog.Debug("fetched jobs", "count", len(jobs))
		return jobs, nil
	}

	return nil, lastErr
}

// FetchJobDetail retrieves a single job's full details by ID.
func (c *Client) FetchJobDetail(id string) (*models.JobDetail, error) {
	c.sleep()

	// Build URL
	u := fmt.Sprintf("%s/%s?_ng=ZW4=", c.BaseURL, url.PathEscape(id))

	slog.Debug("fetching job detail", "id", id, "url", u)

	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(c.config.RetryDelayMs*attempt) * time.Millisecond
			slog.Debug("retrying request", "attempt", attempt, "delay_ms", delay.Milliseconds())
			time.Sleep(delay)
		}

		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		setHeaders(req)

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("job not found: %s", id)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))

			// Don't retry on 4xx errors (except 429)
			if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
				return nil, lastErr
			}
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		var job models.JobDetail
		if err := json.Unmarshal(body, &job); err != nil {
			return nil, fmt.Errorf("failed to unmarshal job detail: %w", err)
		}

		// Store the raw JSON
		job.RawData = strings.TrimSpace(string(body))

		slog.Debug("fetched job detail", "id", id, "status", job.Status)
		return &job, nil
	}

	return nil, lastErr
}
