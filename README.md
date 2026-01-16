# JobGipfel 🏔️

AI-powered job application platform for the Swiss job market.

## Overview

JobGipfel automates job searching, CV generation, and application submission using AI. It scrapes jobs from job-room.ch, matches them to your profile, generates personalized CVs and cover letters, and can automatically apply on your behalf.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                           Frontend (TBD)                            │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          API Gateway (TBD)                          │
└─────────────────────────────────────────────────────────────────────┘
         │           │           │           │           │           │
         ▼           ▼           ▼           ▼           ▼           ▼
┌─────────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│    Auth     │ │    CV    │ │AutoApply │ │  Search  │ │ Matching │ │Analytics │
│   :8082     │ │Generator │ │  :8084   │ │  :8085   │ │  :8086   │ │  :8087   │
│             │ │  :8083   │ │          │ │          │ │          │ │          │
└─────────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘
         │           │           │           │           │           │
         └───────────┴───────────┴───────────┴───────────┴───────────┘
                                    │
                                    ▼
                        ┌───────────────────┐
                        │    PostgreSQL     │
                        │      :5432        │
                        └───────────────────┘
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| **scrapper** | CLI | Scrapes jobs from job-room.ch |
| **auth_service** | 8082 | OAuth, JWT, user profiles, career data |
| **cv_generator** | 8083 | AI-powered CV generation → PDF |
| **autoapply_service** | 8084 | Automated job applications |
| **job_search** | 8085 | AI-powered job search with filters |
| **matching_service** | 8086 | Job recommendations & scoring |
| **analytics_service** | 8087 | Dashboard & market insights |

## Tech Stack

- **Backend**: Go 1.23
- **Database**: PostgreSQL 16 with pgvector
- **AI**: Google Gemini (gemini-2.0-flash, text-embedding-004)
- **PDF**: chromedp (headless Chrome)
- **OAuth**: Google, LinkedIn

## Quick Start

### Using Docker (Recommended)

```bash
# Start all services with database
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

### Manual Setup

1. **Start PostgreSQL**:
   ```bash
   docker run -d \
     --name jobgipfel-db \
     -e POSTGRES_PASSWORD=postgres \
     -p 5432:5432 \
     pgvector/pgvector:pg16
   ```

2. **Configure services**:
   ```bash
   for dir in auth_service cv_generator autoapply_service job_search matching_service analytics_service; do
     cp $dir/.env.example $dir/.env
   done
   ```

3. **Run migrations**:
   ```bash
   cd auth_service && go run ./cmd/server migrate
   cd ../autoapply_service && go run ./cmd/server migrate
   cd ../scrapper && go run ./cmd/scrapper migrate
   ```

4. **Start services**:
   ```bash
   # Start each in a separate terminal
   cd auth_service && go run ./cmd/server
   cd cv_generator && go run ./cmd/server
   cd autoapply_service && go run ./cmd/server
   cd job_search && go run ./cmd/server
   cd matching_service && go run ./cmd/server
   cd analytics_service && go run ./cmd/server
   ```

## Environment Variables

Create `.env` files from `.env.example` in each service. Key variables:

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `GEMINI_API_KEY` | Google AI API key |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |
| `JWT_SECRET` | Secret for JWT signing |

## API Examples

### Search Jobs
```bash
curl -X POST http://localhost:8085/api/v1/jobs/search \
  -H "Content-Type: application/json" \
  -d '{"query": "Python developer Zurich", "use_ai": true}'
```

### Generate CV
```bash
curl -X POST http://localhost:8083/api/v1/cv/generate \
  -H "Authorization: Bearer <token>" \
  -d '{"style": "modern", "color_scheme": "blue"}' \
  --output resume.pdf
```

### Auto-Apply
```bash
curl -X POST http://localhost:8084/api/v1/apply/email \
  -H "Authorization: Bearer <token>" \
  -d '{
    "job_title": "Software Engineer",
    "company_name": "Google",
    "recipient_email": "jobs@google.com"
  }'
```

### Get Job Matches
```bash
curl http://localhost:8086/api/v1/matches \
  -H "Authorization: Bearer <token>"
```

### Get Dashboard
```bash
curl http://localhost:8087/api/v1/dashboard \
  -H "Authorization: Bearer <token>"
```

## Development

### Build All Services
```bash
./scripts/build-all.sh
```

### Run Tests
```bash
./scripts/test-all.sh
```

## License

MIT
