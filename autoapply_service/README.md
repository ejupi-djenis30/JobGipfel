# AutoApply Service

Microservice for automatically applying to jobs via email and web forms using AI-powered personalization.

## Features

- **Email Applications**: Send personalized applications via SMTP
- **Web Form Automation**: Fill and submit application forms using headless Chrome
- **AI Cover Letters**: Gemini generates tailored cover letters
- **Intelligent Form Filling**: AI analyzes forms and generates appropriate responses
- **Rate Limiting**: Configurable limits to avoid detection
- **Application Tracking**: Store and monitor application status

## Prerequisites

- **Go 1.23+**
- **PostgreSQL** (same DB as auth_service)
- **Chrome/Chromium** for web automation
- **Running auth_service** with user profile data

## Quick Start

1. **Copy environment file**:
   ```bash
   cp .env.example .env
   ```

2. **Configure** (required):
   - `DATABASE_URL`: PostgreSQL connection string
   - `GEMINI_API_KEY`: Your Gemini API key
   - `SMTP_*`: Email configuration

3. **Run migrations**:
   ```bash
   go run ./cmd/server migrate
   ```

4. **Start server**:
   ```bash
   go run ./cmd/server
   ```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/apply/email` | Apply via email |
| POST | `/api/v1/apply/web` | Apply via web form |
| GET | `/api/v1/applications` | List applications |
| GET | `/api/v1/applications/:id` | Get application details |
| POST | `/api/v1/cover-letter/generate` | Generate cover letter only |

## Email Application

```bash
curl -X POST http://localhost:8084/api/v1/apply/email \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "job_title": "Software Engineer",
    "company_name": "Google",
    "recipient_email": "jobs@google.com",
    "job_description": "We are looking for..."
  }'
```

## Web Application

```bash
curl -X POST http://localhost:8084/api/v1/apply/web \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "job_title": "Software Engineer",
    "company_name": "Google",
    "job_url": "https://careers.google.com/apply/123",
    "job_description": "We are looking for..."
  }'
```

## Rate Limiting

Default: 20 applications per hour per user. Configure via `RATE_LIMIT_PER_HOUR`.
