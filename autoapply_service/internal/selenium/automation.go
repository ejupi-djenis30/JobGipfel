package selenium

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"

	"autoapply_service/internal/models"
)

// Automation handles web form automation.
type Automation struct {
	chromePath string
	delay      time.Duration
}

// NewAutomation creates a new web automation instance.
func NewAutomation(chromePath string, delay time.Duration) *Automation {
	return &Automation{
		chromePath: chromePath,
		delay:      delay,
	}
}

// ApplyResult contains the result of a web application.
type ApplyResult struct {
	Success    bool
	Screenshot []byte
	Message    string
}

// ApplyToJob automates filling out and submitting a job application form.
func (a *Automation) ApplyToJob(ctx context.Context, jobURL string, formResponses []models.FormResponse, resumeBytes []byte) (*ApplyResult, error) {
	slog.Info("Starting web application", "url", jobURL)

	// Create browser context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1920, 1080),
	)

	if a.chromePath != "" {
		opts = append(opts, chromedp.ExecPath(a.chromePath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, 5*time.Minute)
	defer cancel()

	var screenshot []byte

	// Navigate to job URL
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(jobURL),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Fill in form fields
	for _, response := range formResponses {
		if err := a.fillField(browserCtx, response); err != nil {
			slog.Warn("Failed to fill field", "field", response.FieldName, "error", err)
		}
		// Add delay between actions to avoid detection
		chromedp.Run(browserCtx, chromedp.Sleep(a.delay))
	}

	// Take screenshot before submit
	if err := chromedp.Run(browserCtx,
		chromedp.CaptureScreenshot(&screenshot),
	); err != nil {
		slog.Warn("Failed to capture screenshot", "error", err)
	}

	// Look for submit button and click
	submitted := false
	submitSelectors := []string{
		`button[type="submit"]`,
		`input[type="submit"]`,
		`button:contains("Apply")`,
		`button:contains("Submit")`,
		`button:contains("Send")`,
		`.apply-button`,
		`#submit-application`,
	}

	for _, selector := range submitSelectors {
		if err := chromedp.Run(browserCtx,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Click(selector, chromedp.ByQuery),
		); err == nil {
			submitted = true
			slog.Info("Clicked submit button", "selector", selector)
			break
		}
	}

	// Wait for confirmation page
	chromedp.Run(browserCtx, chromedp.Sleep(3*time.Second))

	// Take final screenshot
	var finalScreenshot []byte
	chromedp.Run(browserCtx, chromedp.CaptureScreenshot(&finalScreenshot))
	if len(finalScreenshot) > 0 {
		screenshot = finalScreenshot
	}

	if !submitted {
		return &ApplyResult{
			Success:    false,
			Screenshot: screenshot,
			Message:    "Could not find submit button",
		}, nil
	}

	return &ApplyResult{
		Success:    true,
		Screenshot: screenshot,
		Message:    "Application submitted successfully",
	}, nil
}

// fillField fills a single form field.
func (a *Automation) fillField(ctx context.Context, response models.FormResponse) error {
	// Try different selectors
	selectors := []string{
		fmt.Sprintf(`[name="%s"]`, response.FieldName),
		fmt.Sprintf(`#%s`, response.FieldName),
		fmt.Sprintf(`[id="%s"]`, response.FieldName),
		fmt.Sprintf(`[placeholder*="%s" i]`, response.FieldName),
	}

	for _, selector := range selectors {
		// Check if element exists
		var nodes []*cdp.Node
		if err := chromedp.Run(ctx, chromedp.Nodes(selector, &nodes, chromedp.ByQuery)); err != nil || len(nodes) == 0 {
			continue
		}

		// Get element type
		var tagName string
		chromedp.Run(ctx, chromedp.Evaluate(fmt.Sprintf(`document.querySelector('%s').tagName`, selector), &tagName))

		switch strings.ToLower(tagName) {
		case "select":
			// Handle select dropdown
			if err := chromedp.Run(ctx,
				chromedp.SetValue(selector, response.Value, chromedp.ByQuery),
			); err == nil {
				return nil
			}
		case "textarea":
			// Handle textarea
			if err := chromedp.Run(ctx,
				chromedp.Clear(selector, chromedp.ByQuery),
				chromedp.SendKeys(selector, response.Value, chromedp.ByQuery),
			); err == nil {
				return nil
			}
		default:
			// Handle input
			if err := chromedp.Run(ctx,
				chromedp.Clear(selector, chromedp.ByQuery),
				chromedp.SendKeys(selector, response.Value, chromedp.ByQuery),
			); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("could not fill field: %s", response.FieldName)
}

// DetectFormFields scans a page and returns detected form fields.
func (a *Automation) DetectFormFields(ctx context.Context, pageURL string) ([]models.FormField, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	if a.chromePath != "" {
		opts = append(opts, chromedp.ExecPath(a.chromePath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, 60*time.Second)
	defer cancel()

	var fields []models.FormField

	// Navigate and extract form fields
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(pageURL),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return nil, fmt.Errorf("failed to load page: %w", err)
	}

	// Extract fields using JavaScript
	var fieldsJSON string
	script := `
		(() => {
			const fields = [];
			document.querySelectorAll('input, textarea, select').forEach(el => {
				const field = {
					name: el.name || el.id || '',
					type: el.type || el.tagName.toLowerCase(),
					label: '',
					required: el.required,
					placeholder: el.placeholder || '',
					options: []
				};
				// Try to find label
				const label = document.querySelector('label[for="' + el.id + '"]');
				if (label) field.label = label.textContent.trim();
				// Get options for select
				if (el.tagName === 'SELECT') {
					Array.from(el.options).forEach(opt => {
						if (opt.value) field.options.push(opt.text);
					});
				}
				if (field.name) fields.push(field);
			});
			return JSON.stringify(fields);
		})()
	`

	if err := chromedp.Run(browserCtx,
		chromedp.Evaluate(script, &fieldsJSON),
	); err != nil {
		return nil, fmt.Errorf("failed to extract fields: %w", err)
	}

	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return nil, fmt.Errorf("failed to parse fields: %w", err)
	}

	slog.Info("Detected form fields", "count", len(fields), "url", pageURL)
	return fields, nil
}
