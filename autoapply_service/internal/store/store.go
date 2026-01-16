package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"autoapply_service/internal/models"
)

// Store handles database operations.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new Store instance.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// CreateApplication creates a new application record.
func (s *Store) CreateApplication(ctx context.Context, userID uuid.UUID, req interface{}, method models.ApplicationMethod) (*models.Application, error) {
	app := &models.Application{
		UserID: userID,
		Method: method,
		Status: models.StatusPending,
	}

	switch r := req.(type) {
	case *models.EmailApplicationRequest:
		app.JobTitle = r.JobTitle
		app.CompanyName = r.CompanyName
		app.RecipientEmail = sql.NullString{String: r.RecipientEmail, Valid: true}
		if r.JobURL != "" {
			app.JobURL = sql.NullString{String: r.JobURL, Valid: true}
		}
		if r.JobID != "" {
			app.JobID = sql.NullString{String: r.JobID, Valid: true}
		}
	case *models.WebApplicationRequest:
		app.JobTitle = r.JobTitle
		app.CompanyName = r.CompanyName
		app.JobURL = sql.NullString{String: r.JobURL, Valid: true}
		if r.JobID != "" {
			app.JobID = sql.NullString{String: r.JobID, Valid: true}
		}
	}

	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO applications (user_id, job_id, job_title, company_name, job_url, method, status, recipient_email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING *`,
		app.UserID, app.JobID, app.JobTitle, app.CompanyName, app.JobURL, app.Method, app.Status, app.RecipientEmail,
	).StructScan(app)
	if err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	return app, nil
}

// UpdateApplicationStatus updates the status of an application.
func (s *Store) UpdateApplicationStatus(ctx context.Context, id uuid.UUID, status models.ApplicationStatus, errorMsg string) error {
	var errPtr *string
	if errorMsg != "" {
		errPtr = &errorMsg
	}

	var sentAt *time.Time
	if status == models.StatusSent {
		now := time.Now()
		sentAt = &now
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE applications SET status = $1, error_message = $2, sent_at = $3, updated_at = NOW()
		WHERE id = $4`,
		status, errPtr, sentAt, id,
	)
	return err
}

// UpdateApplicationCoverLetter updates the cover letter of an application.
func (s *Store) UpdateApplicationCoverLetter(ctx context.Context, id uuid.UUID, coverLetter string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE applications SET cover_letter = $1, updated_at = NOW() WHERE id = $2`,
		coverLetter, id,
	)
	return err
}

// UpdateApplicationScreenshot updates the screenshot URL.
func (s *Store) UpdateApplicationScreenshot(ctx context.Context, id uuid.UUID, screenshotURL string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE applications SET screenshot_url = $1, updated_at = NOW() WHERE id = $2`,
		screenshotURL, id,
	)
	return err
}

// UpdateApplicationCV updates the CV data for an application.
func (s *Store) UpdateApplicationCV(ctx context.Context, id uuid.UUID, cvBase64, style, colorScheme string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE applications SET cv_data = $1, cv_style = $2, cv_color_scheme = $3, updated_at = NOW() WHERE id = $4`,
		cvBase64, style, colorScheme, id,
	)
	return err
}

// GetApplicationCV retrieves the CV data for an application.
func (s *Store) GetApplicationCV(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var cvData sql.NullString
	err := s.db.GetContext(ctx, &cvData, "SELECT cv_data FROM applications WHERE id = $1", id)
	if err != nil || !cvData.Valid {
		return nil, err
	}
	return []byte(cvData.String), nil
}

// IncrementRetryCount increments the retry count.
func (s *Store) IncrementRetryCount(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE applications SET retry_count = retry_count + 1, status = 'retrying', updated_at = NOW()
		WHERE id = $1`,
		id,
	)
	return err
}

// GetApplication retrieves an application by ID.
func (s *Store) GetApplication(ctx context.Context, id uuid.UUID) (*models.Application, error) {
	var app models.Application
	err := s.db.GetContext(ctx, &app, "SELECT * FROM applications WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}
	return &app, nil
}

// ListApplications returns applications for a user.
func (s *Store) ListApplications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Application, error) {
	var apps []models.Application
	err := s.db.SelectContext(ctx, &apps, `
		SELECT * FROM applications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}
	return apps, nil
}

// CountApplications counts applications for a user.
func (s *Store) CountApplications(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM applications WHERE user_id = $1", userID)
	return count, err
}

// CountApplicationsInLastHour counts applications in the last hour (for rate limiting).
func (s *Store) CountApplicationsInLastHour(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM applications 
		WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 hour'`,
		userID,
	)
	return count, err
}
