package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"auth_service/internal/auth"
	"auth_service/internal/models"
)

// Store handles database operations.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new Store instance.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// DB returns the underlying database connection.
func (s *Store) DB() *sqlx.DB {
	return s.db
}

// ==================== User Operations ====================

// GetUserByID retrieves a user by ID.
func (s *Store) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE email = $1", email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByGoogleID retrieves a user by Google ID.
func (s *Store) GetUserByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	var user models.User
	err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE google_id = $1", googleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// GetUserByLinkedInID retrieves a user by LinkedIn ID.
func (s *Store) GetUserByLinkedInID(ctx context.Context, linkedInID string) (*models.User, error) {
	var user models.User
	err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE linkedin_id = $1", linkedInID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

// CreateUser creates a new user.
func (s *Store) CreateUser(ctx context.Context, email string, emailVerified bool, googleID, linkedInID, avatarURL *string) (*models.User, error) {
	var user models.User
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO users (email, email_verified, google_id, linkedin_id, avatar_url, status, last_login_at)
		VALUES ($1, $2, $3, $4, $5, 'active', NOW())
		RETURNING *`,
		email, emailVerified, googleID, linkedInID, avatarURL,
	).StructScan(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &user, nil
}

// UpdateUserLastLogin updates the last login timestamp.
func (s *Store) UpdateUserLastLogin(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1",
		userID,
	)
	return err
}

// UpdateUserGoogleID updates the user's Google ID.
func (s *Store) UpdateUserGoogleID(ctx context.Context, userID uuid.UUID, googleID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET google_id = $1, updated_at = NOW() WHERE id = $2",
		googleID, userID,
	)
	return err
}

// UpdateUserLinkedInID updates the user's LinkedIn ID.
func (s *Store) UpdateUserLinkedInID(ctx context.Context, userID uuid.UUID, linkedInID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE users SET linkedin_id = $1, updated_at = NOW() WHERE id = $2",
		linkedInID, userID,
	)
	return err
}

// ==================== Refresh Token Operations ====================

// SaveRefreshToken stores a refresh token.
func (s *Store) SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

// GetRefreshToken retrieves a refresh token by hash.
func (s *Store) GetRefreshToken(ctx context.Context, tokenHash string) (uuid.UUID, time.Time, error) {
	var result struct {
		UserID    uuid.UUID `db:"user_id"`
		ExpiresAt time.Time `db:"expires_at"`
	}
	err := s.db.GetContext(ctx, &result, `
		SELECT user_id, expires_at FROM refresh_tokens 
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		tokenHash,
	)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}
	return result.UserID, result.ExpiresAt, nil
}

// RevokeRefreshToken revokes a refresh token.
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE token_hash = $1",
		tokenHash,
	)
	return err
}

// RevokeAllUserTokens revokes all refresh tokens for a user.
func (s *Store) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL",
		userID,
	)
	return err
}

// ==================== Profile Operations ====================

// GetProfile retrieves a user's profile.
func (s *Store) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	var profile models.Profile
	err := s.db.GetContext(ctx, &profile, "SELECT * FROM profiles WHERE user_id = $1", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &profile, nil
}

// UpsertProfile creates or updates a user's profile.
func (s *Store) UpsertProfile(ctx context.Context, userID uuid.UUID, req *models.ProfileRequest) (*models.Profile, error) {
	var profile models.Profile
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO profiles (
			user_id, first_name, last_name, headline, summary,
			phone, website, linkedin_url, github_url,
			city, country, postal_code,
			preferred_job_titles, preferred_locations,
			salary_expectation_min, salary_expectation_max, salary_currency,
			work_authorization, willing_to_relocate
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (user_id) DO UPDATE SET
			first_name = COALESCE(EXCLUDED.first_name, profiles.first_name),
			last_name = COALESCE(EXCLUDED.last_name, profiles.last_name),
			headline = COALESCE(EXCLUDED.headline, profiles.headline),
			summary = COALESCE(EXCLUDED.summary, profiles.summary),
			phone = COALESCE(EXCLUDED.phone, profiles.phone),
			website = COALESCE(EXCLUDED.website, profiles.website),
			linkedin_url = COALESCE(EXCLUDED.linkedin_url, profiles.linkedin_url),
			github_url = COALESCE(EXCLUDED.github_url, profiles.github_url),
			city = COALESCE(EXCLUDED.city, profiles.city),
			country = COALESCE(EXCLUDED.country, profiles.country),
			postal_code = COALESCE(EXCLUDED.postal_code, profiles.postal_code),
			preferred_job_titles = COALESCE(EXCLUDED.preferred_job_titles, profiles.preferred_job_titles),
			preferred_locations = COALESCE(EXCLUDED.preferred_locations, profiles.preferred_locations),
			salary_expectation_min = COALESCE(EXCLUDED.salary_expectation_min, profiles.salary_expectation_min),
			salary_expectation_max = COALESCE(EXCLUDED.salary_expectation_max, profiles.salary_expectation_max),
			salary_currency = COALESCE(EXCLUDED.salary_currency, profiles.salary_currency),
			work_authorization = COALESCE(EXCLUDED.work_authorization, profiles.work_authorization),
			willing_to_relocate = COALESCE(EXCLUDED.willing_to_relocate, profiles.willing_to_relocate),
			updated_at = NOW()
		RETURNING *`,
		userID,
		req.FirstName, req.LastName, req.Headline, req.Summary,
		req.Phone, req.Website, req.LinkedInURL, req.GithubURL,
		req.City, req.Country, req.PostalCode,
		pq.Array(req.PreferredJobTitles), pq.Array(req.PreferredLocations),
		req.SalaryExpectationMin, req.SalaryExpectationMax, req.SalaryCurrency,
		req.WorkAuthorization, nilBool(req.WillingToRelocate),
	).StructScan(&profile)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert profile: %w", err)
	}
	return &profile, nil
}

// UpdateProfileImportData updates the import source and raw data.
func (s *Store) UpdateProfileImportData(ctx context.Context, userID uuid.UUID, importedFrom string, rawData []byte) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE profiles SET imported_from = $1, raw_import_data = $2, updated_at = NOW()
		WHERE user_id = $3`,
		importedFrom, rawData, userID,
	)
	return err
}

// ==================== Experience Operations ====================

// ListExperiences returns all experiences for a user.
func (s *Store) ListExperiences(ctx context.Context, userID uuid.UUID) ([]models.Experience, error) {
	var experiences []models.Experience
	err := s.db.SelectContext(ctx, &experiences,
		"SELECT * FROM experiences WHERE user_id = $1 ORDER BY is_current DESC, start_date DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list experiences: %w", err)
	}
	return experiences, nil
}

// GetExperience retrieves an experience by ID.
func (s *Store) GetExperience(ctx context.Context, id uuid.UUID) (*models.Experience, error) {
	var exp models.Experience
	err := s.db.GetContext(ctx, &exp, "SELECT * FROM experiences WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get experience: %w", err)
	}
	return &exp, nil
}

// CreateExperience creates a new experience.
func (s *Store) CreateExperience(ctx context.Context, userID uuid.UUID, req *models.ExperienceRequest) (*models.Experience, error) {
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	var endDate sql.NullTime
	if req.EndDate != nil && *req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			endDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	var exp models.Experience
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO experiences (
			user_id, title, company_name, company_linkedin_url, company_logo_url,
			employment_type, location, location_type,
			start_date, end_date, is_current,
			description, achievements, skills_used, display_order
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING *`,
		userID, req.Title, req.CompanyName, req.CompanyLinkedInURL, req.CompanyLogoURL,
		req.EmploymentType, req.Location, req.LocationType,
		startDate, endDate, req.IsCurrent,
		req.Description, pq.Array(req.Achievements), pq.Array(req.SkillsUsed), nilInt(req.DisplayOrder),
	).StructScan(&exp)
	if err != nil {
		return nil, fmt.Errorf("failed to create experience: %w", err)
	}
	return &exp, nil
}

// UpdateExperience updates an experience.
func (s *Store) UpdateExperience(ctx context.Context, id uuid.UUID, req *models.ExperienceRequest) (*models.Experience, error) {
	startDate, _ := time.Parse("2006-01-02", req.StartDate)
	var endDate sql.NullTime
	if req.EndDate != nil && *req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			endDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	var exp models.Experience
	err := s.db.QueryRowxContext(ctx, `
		UPDATE experiences SET
			title = $1, company_name = $2, company_linkedin_url = $3, company_logo_url = $4,
			employment_type = $5, location = $6, location_type = $7,
			start_date = $8, end_date = $9, is_current = $10,
			description = $11, achievements = $12, skills_used = $13, display_order = $14,
			updated_at = NOW()
		WHERE id = $15
		RETURNING *`,
		req.Title, req.CompanyName, req.CompanyLinkedInURL, req.CompanyLogoURL,
		req.EmploymentType, req.Location, req.LocationType,
		startDate, endDate, req.IsCurrent,
		req.Description, pq.Array(req.Achievements), pq.Array(req.SkillsUsed), nilInt(req.DisplayOrder),
		id,
	).StructScan(&exp)
	if err != nil {
		return nil, fmt.Errorf("failed to update experience: %w", err)
	}
	return &exp, nil
}

// DeleteExperience deletes an experience.
func (s *Store) DeleteExperience(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM experiences WHERE id = $1", id)
	return err
}

// ==================== Education Operations ====================

// ListEducation returns all education entries for a user.
func (s *Store) ListEducation(ctx context.Context, userID uuid.UUID) ([]models.Education, error) {
	var education []models.Education
	err := s.db.SelectContext(ctx, &education,
		"SELECT * FROM education WHERE user_id = $1 ORDER BY is_current DESC, end_date DESC NULLS FIRST",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list education: %w", err)
	}
	return education, nil
}

// GetEducation retrieves an education entry by ID.
func (s *Store) GetEducation(ctx context.Context, id uuid.UUID) (*models.Education, error) {
	var edu models.Education
	err := s.db.GetContext(ctx, &edu, "SELECT * FROM education WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get education: %w", err)
	}
	return &edu, nil
}

// CreateEducation creates a new education entry.
func (s *Store) CreateEducation(ctx context.Context, userID uuid.UUID, req *models.EducationRequest) (*models.Education, error) {
	var startDate, endDate sql.NullTime
	if req.StartDate != nil && *req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			startDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.EndDate != nil && *req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			endDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	var edu models.Education
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO education (
			user_id, institution_name, institution_logo_url,
			degree, field_of_study, grade,
			start_date, end_date, is_current,
			description, activities, display_order
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING *`,
		userID, req.InstitutionName, req.InstitutionLogoURL,
		req.Degree, req.FieldOfStudy, req.Grade,
		startDate, endDate, req.IsCurrent,
		req.Description, pq.Array(req.Activities), nilInt(req.DisplayOrder),
	).StructScan(&edu)
	if err != nil {
		return nil, fmt.Errorf("failed to create education: %w", err)
	}
	return &edu, nil
}

// UpdateEducation updates an education entry.
func (s *Store) UpdateEducation(ctx context.Context, id uuid.UUID, req *models.EducationRequest) (*models.Education, error) {
	var startDate, endDate sql.NullTime
	if req.StartDate != nil && *req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
			startDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.EndDate != nil && *req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", *req.EndDate); err == nil {
			endDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	var edu models.Education
	err := s.db.QueryRowxContext(ctx, `
		UPDATE education SET
			institution_name = $1, institution_logo_url = $2,
			degree = $3, field_of_study = $4, grade = $5,
			start_date = $6, end_date = $7, is_current = $8,
			description = $9, activities = $10, display_order = $11,
			updated_at = NOW()
		WHERE id = $12
		RETURNING *`,
		req.InstitutionName, req.InstitutionLogoURL,
		req.Degree, req.FieldOfStudy, req.Grade,
		startDate, endDate, req.IsCurrent,
		req.Description, pq.Array(req.Activities), nilInt(req.DisplayOrder),
		id,
	).StructScan(&edu)
	if err != nil {
		return nil, fmt.Errorf("failed to update education: %w", err)
	}
	return &edu, nil
}

// DeleteEducation deletes an education entry.
func (s *Store) DeleteEducation(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM education WHERE id = $1", id)
	return err
}

// ==================== Skills Operations ====================

// ListSkills returns all skills for a user.
func (s *Store) ListSkills(ctx context.Context, userID uuid.UUID) ([]models.Skill, error) {
	var skills []models.Skill
	err := s.db.SelectContext(ctx, &skills,
		"SELECT * FROM skills WHERE user_id = $1 ORDER BY category, name",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}
	return skills, nil
}

// GetSkill retrieves a skill by ID.
func (s *Store) GetSkill(ctx context.Context, id uuid.UUID) (*models.Skill, error) {
	var skill models.Skill
	err := s.db.GetContext(ctx, &skill, "SELECT * FROM skills WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get skill: %w", err)
	}
	return &skill, nil
}

// CreateSkill creates a new skill.
func (s *Store) CreateSkill(ctx context.Context, userID uuid.UUID, req *models.SkillRequest) (*models.Skill, error) {
	var issueDate, expiryDate sql.NullTime
	if req.IssueDate != nil && *req.IssueDate != "" {
		if t, err := time.Parse("2006-01-02", *req.IssueDate); err == nil {
			issueDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		if t, err := time.Parse("2006-01-02", *req.ExpiryDate); err == nil {
			expiryDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	var skill models.Skill
	err := s.db.QueryRowxContext(ctx, `
		INSERT INTO skills (
			user_id, name, category, proficiency_level, years_of_experience,
			is_certification, issuing_organization, issue_date, expiry_date, credential_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, name) DO UPDATE SET
			category = EXCLUDED.category,
			proficiency_level = EXCLUDED.proficiency_level,
			years_of_experience = EXCLUDED.years_of_experience,
			is_certification = EXCLUDED.is_certification,
			issuing_organization = EXCLUDED.issuing_organization,
			issue_date = EXCLUDED.issue_date,
			expiry_date = EXCLUDED.expiry_date,
			credential_url = EXCLUDED.credential_url,
			updated_at = NOW()
		RETURNING *`,
		userID, req.Name, req.Category, req.ProficiencyLevel, req.YearsOfExperience,
		req.IsCertification, req.IssuingOrganization, issueDate, expiryDate, req.CredentialURL,
	).StructScan(&skill)
	if err != nil {
		return nil, fmt.Errorf("failed to create skill: %w", err)
	}
	return &skill, nil
}

// UpdateSkill updates a skill.
func (s *Store) UpdateSkill(ctx context.Context, id uuid.UUID, req *models.SkillRequest) (*models.Skill, error) {
	var issueDate, expiryDate sql.NullTime
	if req.IssueDate != nil && *req.IssueDate != "" {
		if t, err := time.Parse("2006-01-02", *req.IssueDate); err == nil {
			issueDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		if t, err := time.Parse("2006-01-02", *req.ExpiryDate); err == nil {
			expiryDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	var skill models.Skill
	err := s.db.QueryRowxContext(ctx, `
		UPDATE skills SET
			name = $1, category = $2, proficiency_level = $3, years_of_experience = $4,
			is_certification = $5, issuing_organization = $6, issue_date = $7, expiry_date = $8, credential_url = $9,
			updated_at = NOW()
		WHERE id = $10
		RETURNING *`,
		req.Name, req.Category, req.ProficiencyLevel, req.YearsOfExperience,
		req.IsCertification, req.IssuingOrganization, issueDate, expiryDate, req.CredentialURL,
		id,
	).StructScan(&skill)
	if err != nil {
		return nil, fmt.Errorf("failed to update skill: %w", err)
	}
	return &skill, nil
}

// DeleteSkill deletes a skill.
func (s *Store) DeleteSkill(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM skills WHERE id = $1", id)
	return err
}

// ==================== Bulk Import Operations ====================

// BulkCreateExperiences creates multiple experiences (for CV import).
func (s *Store) BulkCreateExperiences(ctx context.Context, userID uuid.UUID, experiences []models.ParsedExperience, importedFrom string) error {
	for i, exp := range experiences {
		startDate := parseFlexibleDate(exp.StartDate)
		var endDate sql.NullTime
		if exp.EndDate != "" && !exp.IsCurrent {
			if t := parseFlexibleDate(exp.EndDate); !t.IsZero() {
				endDate = sql.NullTime{Time: t, Valid: true}
			}
		}

		_, err := s.db.ExecContext(ctx, `
			INSERT INTO experiences (
				user_id, title, company_name, employment_type, location,
				start_date, end_date, is_current, description, achievements, imported_from, display_order
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			userID, exp.Title, exp.CompanyName, nilString(exp.EmploymentType), nilString(exp.Location),
			startDate, endDate, exp.IsCurrent, nilString(exp.Description), pq.Array(exp.Achievements), importedFrom, i,
		)
		if err != nil {
			return fmt.Errorf("failed to create experience: %w", err)
		}
	}
	return nil
}

// BulkCreateEducation creates multiple education entries (for CV import).
func (s *Store) BulkCreateEducation(ctx context.Context, userID uuid.UUID, education []models.ParsedEducation, importedFrom string) error {
	for i, edu := range education {
		var startDate, endDate sql.NullTime
		if edu.StartDate != "" {
			if t := parseFlexibleDate(edu.StartDate); !t.IsZero() {
				startDate = sql.NullTime{Time: t, Valid: true}
			}
		}
		if edu.EndDate != "" {
			if t := parseFlexibleDate(edu.EndDate); !t.IsZero() {
				endDate = sql.NullTime{Time: t, Valid: true}
			}
		}

		_, err := s.db.ExecContext(ctx, `
			INSERT INTO education (
				user_id, institution_name, degree, field_of_study, grade,
				start_date, end_date, activities, imported_from, display_order
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			userID, edu.InstitutionName, nilString(edu.Degree), nilString(edu.FieldOfStudy), nilString(edu.Grade),
			startDate, endDate, pq.Array(edu.Activities), importedFrom, i,
		)
		if err != nil {
			return fmt.Errorf("failed to create education: %w", err)
		}
	}
	return nil
}

// BulkCreateSkills creates multiple skills (for CV import).
func (s *Store) BulkCreateSkills(ctx context.Context, userID uuid.UUID, skills []models.ParsedSkill, importedFrom string) error {
	for _, skill := range skills {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO skills (user_id, name, category, proficiency_level, imported_from)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (user_id, name) DO NOTHING`,
			userID, skill.Name, nilString(skill.Category), nilString(skill.ProficiencyLevel), importedFrom,
		)
		if err != nil {
			return fmt.Errorf("failed to create skill: %w", err)
		}
	}
	return nil
}

// ==================== Resume Data Export ====================

// GetResumeData retrieves all profile data for CV generation.
func (s *Store) GetResumeData(ctx context.Context, userID uuid.UUID) (*models.ResumeData, error) {
	user, err := s.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	profile, _ := s.GetProfile(ctx, userID)
	experiences, _ := s.ListExperiences(ctx, userID)
	education, _ := s.ListEducation(ctx, userID)
	skills, _ := s.ListSkills(ctx, userID)

	return &models.ResumeData{
		User:        user.ToResponse(),
		Profile:     profile,
		Experiences: experiences,
		Education:   education,
		Skills:      skills,
	}, nil
}

// ==================== Helper Functions ====================

func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nilInt(i *int) *int {
	return i
}

func nilBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func parseFlexibleDate(s string) time.Time {
	formats := []string{
		"2006-01-02",
		"2006-01",
		"2006",
		"01/2006",
		"Jan 2006",
		"January 2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// Ensure auth package is used
var _ = auth.HashRefreshToken
