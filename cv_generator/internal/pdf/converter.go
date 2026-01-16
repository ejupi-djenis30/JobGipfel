package pdf

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Converter handles HTML to PDF conversion.
type Converter struct {
	chromePath string
}

// NewConverter creates a new PDF converter.
func NewConverter(chromePath string) *Converter {
	return &Converter{
		chromePath: chromePath,
	}
}

// ConvertHTMLToPDF converts HTML content to a PDF byte slice.
func (c *Converter) ConvertHTMLToPDF(ctx context.Context, html string) ([]byte, error) {
	start := time.Now()

	// Create browser context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	// Use custom Chrome path if specified
	if c.chromePath != "" {
		opts = append(opts, chromedp.ExecPath(c.chromePath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	browserCtx, cancel = context.WithTimeout(browserCtx, 60*time.Second)
	defer cancel()

	var pdfBuf []byte

	// Navigate to HTML content and generate PDF
	err := chromedp.Run(browserCtx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}

			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		// Wait for content to render
		chromedp.Sleep(500*time.Millisecond),
		// Generate PDF
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // A4 width in inches
				WithPaperHeight(11.69). // A4 height in inches
				WithMarginTop(0.4).
				WithMarginBottom(0.4).
				WithMarginLeft(0.4).
				WithMarginRight(0.4).
				WithPreferCSSPageSize(true).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBuf = buf
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("PDF generation failed: %w", err)
	}

	slog.Debug("PDF conversion completed",
		"duration_ms", time.Since(start).Milliseconds(),
		"pdf_size_bytes", len(pdfBuf),
	)

	return pdfBuf, nil
}
