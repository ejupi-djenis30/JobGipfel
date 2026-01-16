# Job Search Service

Advanced job search microservice with AI-powered natural language queries, semantic search, and comprehensive filtering.

## Features

- **AI-Powered Search**: Natural language query understanding with Gemini
- **Semantic Search**: Vector embeddings for similarity search
- **Advanced Filters**: Location, workload, employment type, company, date
- **Radius Search**: Find jobs within X km of coordinates
- **Full-Text Search**: PostgreSQL `tsvector` for keyword matching
- **Saved Searches**: Save and rerun searches
- **Personalized Recommendations**: Based on user profile

## Prerequisites

- **Go 1.23+**
- **PostgreSQL** with jobs data (from scrapper service)
- **Gemini API key** (optional, for AI features)

## Quick Start

1. **Copy environment file**:
   ```bash
   cp .env.example .env
   ```

2. **Configure**:
   - `DATABASE_URL`: PostgreSQL connection string
   - `GEMINI_API_KEY`: Optional, enables AI search

3. **Start server**:
   ```bash
   go run ./cmd/server
   ```

## API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/v1/jobs/search` | ❌ | Search jobs |
| GET | `/api/v1/jobs/:id` | ❌ | Get job details |
| GET | `/api/v1/jobs/filters` | ❌ | Get filter options |
| POST | `/api/v1/jobs/recommend` | ✅ | Get recommendations |
| POST | `/api/v1/saved-searches` | ✅ | Save a search |
| GET | `/api/v1/saved-searches` | ✅ | List saved searches |
| DELETE | `/api/v1/saved-searches/:id` | ✅ | Delete saved search |

## Search Request

```json
{
  "query": "Software Engineer Python",
  "use_ai": true,
  "filters": {
    "location": {
      "city": "Zürich",
      "canton": "ZH",
      "radius_km": 25,
      "lat": 47.3769,
      "lon": 8.5417
    },
    "workload": {
      "min": 80,
      "max": 100
    },
    "employment": {
      "permanent": true,
      "immediately": true
    },
    "company": {
      "names": ["Google"],
      "exclude_anonymous": true
    },
    "date": {
      "posted_after": "2024-01-01"
    }
  },
  "sort": {
    "field": "created_time",
    "order": "desc"
  },
  "pagination": {
    "page": 1,
    "per_page": 20
  }
}
```

## Example Usage

```bash
# Simple search
curl -X POST http://localhost:8085/api/v1/jobs/search \
  -H "Content-Type: application/json" \
  -d '{"query": "Software Developer"}'

# AI-enhanced search
curl -X POST http://localhost:8085/api/v1/jobs/search \
  -d '{"query": "I want a remote Python job in Zurich", "use_ai": true}'

# Get job details
curl http://localhost:8085/api/v1/jobs/abc123
```
