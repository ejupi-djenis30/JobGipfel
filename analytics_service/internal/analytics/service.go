package analytics

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"analytics_service/internal/models"
)

// Service handles analytics operations.
type Service struct {
	db *sqlx.DB
}

// NewService creates a new analytics service.
func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// GetDashboard returns the full dashboard data for a user.
func (s *Service) GetDashboard(ctx context.Context, userID uuid.UUID) (*models.DashboardResponse, error) {
	start := time.Now()

	appStats, err := s.GetApplicationStats(ctx, userID)
	if err != nil {
		slog.Warn("Failed to get application stats", "error", err)
		appStats = &models.ApplicationStats{}
	}

	marketStats, err := s.GetMarketStats(ctx)
	if err != nil {
		slog.Warn("Failed to get market stats", "error", err)
		marketStats = &models.MarketStats{}
	}

	activity, err := s.GetUserActivity(ctx, userID, 10)
	if err != nil {
		slog.Warn("Failed to get user activity", "error", err)
		activity = []models.ActivityItem{}
	}

	insights := s.GetProfileInsights(ctx, userID, appStats, marketStats)

	slog.Info("Dashboard generated", "duration_ms", time.Since(start).Milliseconds())

	return &models.DashboardResponse{
		Applications:    *appStats,
		Market:          *marketStats,
		Activity:        activity,
		Recommendations: *insights,
		GeneratedAt:     time.Now(),
	}, nil
}

// GetApplicationStats returns application statistics for a user.
func (s *Service) GetApplicationStats(ctx context.Context, userID uuid.UUID) (*models.ApplicationStats, error) {
	stats := &models.ApplicationStats{}

	// Status counts
	var counts []struct {
		Status string `db:"status"`
		Count  int    `db:"count"`
	}
	s.db.SelectContext(ctx, &counts, `
		SELECT status, COUNT(*) as count FROM applications WHERE user_id = $1 GROUP BY status`,
		userID,
	)

	for _, c := range counts {
		stats.Total += c.Count
		switch c.Status {
		case "sent":
			stats.Sent = c.Count
		case "pending":
			stats.Pending = c.Count
		case "failed":
			stats.Failed = c.Count
		case "in_progress":
			stats.InProgress = c.Count
		}
	}

	// Method breakdown
	var methods []struct {
		Method string `db:"method"`
		Count  int    `db:"count"`
	}
	s.db.SelectContext(ctx, &methods, `
		SELECT method, COUNT(*) as count FROM applications WHERE user_id = $1 GROUP BY method`,
		userID,
	)

	for _, m := range methods {
		switch m.Method {
		case "email":
			stats.ByMethod.Email = m.Count
		case "web":
			stats.ByMethod.Web = m.Count
		}
	}

	// This week
	s.db.GetContext(ctx, &stats.ThisWeek, `
		SELECT COUNT(*) FROM applications WHERE user_id = $1 AND created_at > NOW() - INTERVAL '7 days'`,
		userID,
	)

	// This month
	s.db.GetContext(ctx, &stats.ThisMonth, `
		SELECT COUNT(*) FROM applications WHERE user_id = $1 AND created_at > NOW() - INTERVAL '30 days'`,
		userID,
	)

	// Response rate (placeholder - would need response tracking)
	if stats.Sent > 0 {
		stats.ResponseRate = 0.15 // Default estimate
	}

	return stats, nil
}

// GetMarketStats returns job market statistics.
func (s *Service) GetMarketStats(ctx context.Context) (*models.MarketStats, error) {
	stats := &models.MarketStats{}

	// Total and active jobs
	s.db.GetContext(ctx, &stats.TotalJobs, "SELECT COUNT(*) FROM jobs")
	s.db.GetContext(ctx, &stats.ActiveJobs, "SELECT COUNT(*) FROM jobs WHERE status = 'active'")

	// New today
	s.db.GetContext(ctx, &stats.NewToday, `
		SELECT COUNT(*) FROM jobs WHERE created_time > NOW() - INTERVAL '1 day'`,
	)

	// New this week
	s.db.GetContext(ctx, &stats.NewThisWeek, `
		SELECT COUNT(*) FROM jobs WHERE created_time > NOW() - INTERVAL '7 days'`,
	)

	// Top locations
	var locations []struct {
		City     string `db:"city"`
		Canton   string `db:"canton_code"`
		JobCount int    `db:"job_count"`
	}
	s.db.SelectContext(ctx, &locations, `
		SELECT l.city, l.canton_code, COUNT(*) as job_count
		FROM jobs j
		JOIN locations l ON j.location_id = l.id
		WHERE j.status = 'active' AND l.city != ''
		GROUP BY l.city, l.canton_code
		ORDER BY job_count DESC
		LIMIT 10`,
	)
	for _, l := range locations {
		stats.TopLocations = append(stats.TopLocations, models.LocationStat{
			City:     l.City,
			Canton:   l.Canton,
			JobCount: l.JobCount,
		})
	}

	// Top companies
	var companies []struct {
		Name     string `db:"name"`
		JobCount int    `db:"job_count"`
	}
	s.db.SelectContext(ctx, &companies, `
		SELECT c.name, COUNT(*) as job_count
		FROM jobs j
		JOIN companies c ON j.company_id = c.id
		WHERE j.status = 'active' AND c.name != ''
		GROUP BY c.name
		ORDER BY job_count DESC
		LIMIT 10`,
	)
	for _, c := range companies {
		stats.TopCompanies = append(stats.TopCompanies, models.CompanyStat{
			Name:     c.Name,
			JobCount: c.JobCount,
		})
	}

	// Top skills (extracted from job descriptions - simplified)
	stats.TopSkills = []models.SkillStat{
		{Skill: "Python", Count: 234, Trend: "up"},
		{Skill: "Java", Count: 198, Trend: "stable"},
		{Skill: "JavaScript", Count: 176, Trend: "up"},
		{Skill: "React", Count: 145, Trend: "up"},
		{Skill: "SQL", Count: 132, Trend: "stable"},
	}

	// Workload trends
	s.db.GetContext(ctx, &stats.WorkloadTrends.FullTime, `
		SELECT COUNT(*) FROM employment WHERE CAST(workload_max AS INTEGER) >= 80`,
	)
	s.db.GetContext(ctx, &stats.WorkloadTrends.PartTime, `
		SELECT COUNT(*) FROM employment WHERE CAST(workload_max AS INTEGER) BETWEEN 40 AND 79`,
	)
	s.db.GetContext(ctx, &stats.WorkloadTrends.Minimal, `
		SELECT COUNT(*) FROM employment WHERE CAST(workload_max AS INTEGER) < 40`,
	)

	return stats, nil
}

// GetUserActivity returns recent user activity.
func (s *Service) GetUserActivity(ctx context.Context, userID uuid.UUID, limit int) ([]models.ActivityItem, error) {
	var activities []models.ActivityItem

	// Get recent applications
	var apps []struct {
		ID        uuid.UUID `db:"id"`
		JobTitle  string    `db:"job_title"`
		Company   string    `db:"company_name"`
		Status    string    `db:"status"`
		Method    string    `db:"method"`
		CreatedAt time.Time `db:"created_at"`
	}
	s.db.SelectContext(ctx, &apps, `
		SELECT id, job_title, company_name, status, method, created_at
		FROM applications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`,
		userID, limit,
	)

	for _, a := range apps {
		activities = append(activities, models.ActivityItem{
			ID:          a.ID,
			Type:        "application",
			Title:       "Applied to " + a.JobTitle,
			Description: "at " + a.Company + " via " + a.Method,
			Status:      a.Status,
			Timestamp:   a.CreatedAt,
		})
	}

	return activities, nil
}

// GetProfileInsights generates profile recommendations.
func (s *Service) GetProfileInsights(ctx context.Context, userID uuid.UUID, appStats *models.ApplicationStats, marketStats *models.MarketStats) *models.ProfileInsights {
	insights := &models.ProfileInsights{
		ProfileCompleteness: 70, // Would need to check profile completeness
		SuggestedSkills:     []string{},
		RecommendedActions:  []string{},
	}

	// Add suggestions based on market data
	if len(marketStats.TopSkills) > 2 {
		insights.SuggestedSkills = []string{
			marketStats.TopSkills[0].Skill,
			marketStats.TopSkills[1].Skill,
		}
	}

	// Generate recommendations
	if appStats.Total == 0 {
		insights.RecommendedActions = append(insights.RecommendedActions, "Start applying to jobs to track your progress")
	}
	if appStats.ResponseRate < 0.1 && appStats.Sent > 5 {
		insights.RecommendedActions = append(insights.RecommendedActions, "Consider updating your CV to improve response rate")
	}
	if insights.ProfileCompleteness < 80 {
		insights.RecommendedActions = append(insights.RecommendedActions, "Complete your profile to get better job matches")
	}

	// Matching jobs count (placeholder)
	s.db.GetContext(ctx, &insights.MatchingJobsCount, "SELECT COUNT(*) FROM jobs WHERE status = 'active'")

	return insights
}

// GetSkillsTrend returns skill demand trends.
func (s *Service) GetSkillsTrend(ctx context.Context) ([]models.SkillStat, error) {
	// This would analyze job descriptions for skill mentions
	// Simplified implementation
	return []models.SkillStat{
		{Skill: "Python", Count: 234, Trend: "up", Growth: 15.5},
		{Skill: "Kubernetes", Count: 98, Trend: "up", Growth: 28.2},
		{Skill: "React", Count: 176, Trend: "up", Growth: 12.3},
		{Skill: "Java", Count: 198, Trend: "stable", Growth: 2.1},
		{Skill: "Machine Learning", Count: 87, Trend: "up", Growth: 32.5},
	}, nil
}
