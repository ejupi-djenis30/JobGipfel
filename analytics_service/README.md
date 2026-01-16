# Analytics Service

Application tracking, statistics, and job market insights dashboard.

## Features

- **Application Stats**: Track sent, pending, failed applications
- **Market Insights**: Job trends, top locations, trending skills
- **Activity Timeline**: Recent user actions
- **Profile Recommendations**: Suggestions to improve job matches

## Quick Start

```bash
cp .env.example .env
# Set DATABASE_URL
go run ./cmd/server
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/dashboard` | Full dashboard data |
| GET | `/api/v1/stats/applications` | Application statistics |
| GET | `/api/v1/stats/market` | Job market insights |
| GET | `/api/v1/stats/skills` | Trending skills |
| GET | `/api/v1/activity` | User activity timeline |

## Example Response

```json
{
  "applications": {
    "total": 45,
    "sent": 40,
    "pending": 3,
    "failed": 2,
    "by_method": {"email": 30, "web": 15},
    "this_week": 5,
    "this_month": 20
  },
  "market": {
    "total_jobs": 15420,
    "new_today": 234,
    "top_locations": [
      {"city": "Zürich", "job_count": 2340}
    ],
    "top_skills": [
      {"skill": "Python", "count": 234, "trend": "up"}
    ]
  },
  "recommendations": {
    "profile_completeness": 85,
    "suggested_skills": ["Kubernetes", "AWS"]
  }
}
```
