# Auth Service

Authentication and profile management microservice for the JobGipfel platform.

## Features

- **OAuth Authentication**: Google and LinkedIn OAuth2 login
- **JWT Tokens**: Access and refresh token management
- **Career Profiles**: Store personal info, experiences, education, and skills
- **CV Parsing**: Import profile data from resumes using Gemini AI
- **Resume Export**: Export all profile data for CV generation

## Quick Start

1. **Copy environment file**:
   ```bash
   cp .env.example .env
   ```

2. **Configure environment variables** (required):
   - `DATABASE_URL`: PostgreSQL connection string
   - `JWT_SECRET`: Secret key for JWT (min 32 chars)
   - OAuth credentials (Google and/or LinkedIn)

3. **Run migrations**:
   ```bash
   go run ./cmd/server migrate
   ```

4. **Start server**:
   ```bash
   go run ./cmd/server
   ```

## API Endpoints

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/auth/google` | Start Google OAuth |
| GET | `/api/v1/auth/google/callback` | Google OAuth callback |
| GET | `/api/v1/auth/linkedin` | Start LinkedIn OAuth |
| GET | `/api/v1/auth/linkedin/callback` | LinkedIn OAuth callback |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Logout |

### Profile (requires auth)
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/me` | Get current user |
| GET | `/api/v1/profile` | Get profile |
| PUT | `/api/v1/profile` | Update profile |
| GET/POST/PUT/DELETE | `/api/v1/experiences` | Work experiences |
| GET/POST/PUT/DELETE | `/api/v1/education` | Education entries |
| GET/POST/PUT/DELETE | `/api/v1/skills` | Skills |
| POST | `/api/v1/import/cv` | Import from CV |
| GET | `/api/v1/export/resume-data` | Export all data |

## OAuth Setup

### Google
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create OAuth 2.0 credentials
3. Add redirect URI: `http://localhost:8082/api/v1/auth/google/callback`

### LinkedIn
1. Go to [LinkedIn Developers](https://www.linkedin.com/developers/)
2. Create an app with "Sign In with LinkedIn using OpenID Connect"
3. Add redirect URI: `http://localhost:8082/api/v1/auth/linkedin/callback`

## CV Import

Send CV content to parse and import:

```bash
curl -X POST http://localhost:8082/api/v1/import/cv \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "file_content": "<CV text content>",
    "file_name": "resume.pdf"
  }'
```

## Commands

```bash
# Start server
go run ./cmd/server

# Run migrations
go run ./cmd/server migrate

# Rollback migrations
go run ./cmd/server migrate --direction down

# Show version
go run ./cmd/server version
```
