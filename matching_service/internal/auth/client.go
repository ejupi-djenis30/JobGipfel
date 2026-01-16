package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"matching_service/internal/models"
)

// Client for auth_service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new auth client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetUserProfile fetches user profile with skills and experience.
func (c *Client) GetUserProfile(ctx context.Context, accessToken string) (*models.UserProfile, error) {
	url := fmt.Sprintf("%s/api/v1/export/resume-data", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("auth_service error: %s - %s", resp.Status, string(body))
	}

	var data struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		Profile *struct {
			FirstName *string `json:"first_name"`
			LastName  *string `json:"last_name"`
			Headline  *string `json:"headline"`
			Summary   *string `json:"summary"`
			City      *string `json:"city"`
			Country   *string `json:"country"`
		} `json:"profile"`
		Skills []struct {
			Name     string  `json:"name"`
			Category *string `json:"category"`
		} `json:"skills"`
		Experiences []struct {
			Title       string  `json:"title"`
			CompanyName string  `json:"company_name"`
			StartDate   string  `json:"start_date"`
			EndDate     *string `json:"end_date"`
			IsCurrent   bool    `json:"is_current"`
			Description *string `json:"description"`
		} `json:"experiences"`
		Education []struct {
			InstitutionName string  `json:"institution_name"`
			Degree          *string `json:"degree"`
			FieldOfStudy    *string `json:"field_of_study"`
		} `json:"education"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode profile: %w", err)
	}

	profile := &models.UserProfile{}

	if data.Profile != nil {
		if data.Profile.FirstName != nil {
			profile.FirstName = *data.Profile.FirstName
		}
		if data.Profile.LastName != nil {
			profile.LastName = *data.Profile.LastName
		}
		if data.Profile.Headline != nil {
			profile.Headline = *data.Profile.Headline
		}
		if data.Profile.Summary != nil {
			profile.Summary = *data.Profile.Summary
		}
		if data.Profile.City != nil {
			profile.City = *data.Profile.City
		}
		if data.Profile.Country != nil {
			profile.Country = *data.Profile.Country
		}
	}

	for _, s := range data.Skills {
		skill := models.Skill{Name: s.Name}
		if s.Category != nil {
			skill.Category = *s.Category
		}
		profile.Skills = append(profile.Skills, skill)
	}

	for _, e := range data.Experiences {
		exp := models.Experience{
			Title:       e.Title,
			CompanyName: e.CompanyName,
			StartDate:   e.StartDate,
			EndDate:     e.EndDate,
			IsCurrent:   e.IsCurrent,
		}
		if e.Description != nil {
			exp.Description = *e.Description
		}
		profile.Experiences = append(profile.Experiences, exp)
	}

	for _, e := range data.Education {
		edu := models.Education{
			InstitutionName: e.InstitutionName,
		}
		if e.Degree != nil {
			edu.Degree = *e.Degree
		}
		if e.FieldOfStudy != nil {
			edu.FieldOfStudy = *e.FieldOfStudy
		}
		profile.Education = append(profile.Education, edu)
	}

	return profile, nil
}

// HealthCheck checks if auth_service is available.
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
		return fmt.Errorf("auth_service unhealthy: %s", resp.Status)
	}

	return nil
}
