# Matching Service

AI-powered job matching and recommendations based on user profiles.

## Features

- **Skills Matching**: Compare user skills with job requirements
- **AI Scoring**: Gemini evaluates profile-job fit
- **Match Breakdown**: Detailed scoring by category
- **Profile Analysis**: AI insights and improvement suggestions

## Quick Start

```bash
cp .env.example .env
# Set DATABASE_URL and GEMINI_API_KEY
go run ./cmd/server
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/matches` | Get matched jobs for user |
| GET | `/api/v1/matches/:job_id/score` | Get fit score for specific job |
| POST | `/api/v1/profile/analyze` | Analyze profile and get suggestions |

## Example

```bash
# Get matched jobs
curl http://localhost:8086/api/v1/matches \
  -H "Authorization: Bearer <token>"

# Get score for specific job
curl http://localhost:8086/api/v1/matches/abc123/score \
  -H "Authorization: Bearer <token>"
```

## Match Score Components

| Component | Weight | Description |
|-----------|--------|-------------|
| Skills | 40% | Matching skills count |
| Experience | 25% | Years of experience alignment |
| Location | 15% | Geographic match |
| Workload | 10% | Preferred workload percentage |
| Semantic | 10% | AI embedding similarity |
