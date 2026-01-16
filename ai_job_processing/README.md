# AI Job Processing Service

A REST API microservice that uses Google Gemini AI to normalize job descriptions into structured format and/or translate them to multiple languages. Integrates with the scrapper service via a shared PostgreSQL database.

## Features

- **Job Normalization**: Extracts structured sections from raw job descriptions:
  - **Tasks**: What the employee will do (duties, responsibilities)
  - **Requirements**: Skills, experience, qualifications needed
  - **Offer**: Salary, benefits, perks, work environment

- **Multi-language Translation**: Translates content to configurable languages
  - Default: German (de), French (fr), Italian (it), English (en)
  - Supports both normalized content AND raw descriptions
  - **Can work without normalization** - just translate the original description

- **Smart Token Saving**: 
  - **Skip already-normalized jobs**
  - **Skip already-translated languages**
  - Only process what's missing

- **Database Integration**: 
  - Process jobs directly from the shared database by ID
  - Automatically save results back to database
  - List pending (un-normalized) jobs

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 14+ (shared database with scrapper)
- Google Gemini API key

### Installation

```bash
cd ai_job_processing
go mod download
go build -o server ./cmd/server
```

### Configuration

```bash
cp .env.example .env
# Edit with your GEMINI_API_KEY and DATABASE_URL
```

### Run

```bash
./server --migrate  # Run migrations first
./server            # Start server on :8081
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/languages` | List configured languages |
| GET | `/api/v1/pending` | List jobs pending normalization |
| **POST** | **`/api/v1/process/:id`** | **Normalize + translate job from DB** |
| **POST** | **`/api/v1/normalize/:id`** | **Normalize only (no translation) from DB** |
| **POST** | **`/api/v1/translate/:id`** | **Translate only (no normalization) from DB** |
| POST | `/api/v1/process` | Normalize + translate raw data |
| POST | `/api/v1/normalize` | Normalize only (raw data) |
| POST | `/api/v1/translate` | Translate only (raw data) |

## Three Processing Modes

### 1. Full Processing (Normalize + Translate)

**`POST /api/v1/process/:id`**

- Normalizes job description into tasks, requirements, offer
- Translates to all target languages
- Saves normalized content + translations to DB

```bash
curl -X POST http://localhost:8081/api/v1/process/job-123
```

### 2. Normalize Only (No Translation)

**`POST /api/v1/normalize/:id`**

- Creates tasks, requirements, offer fields
- Does NOT translate to other languages
- Use when you want structured content in the original language

```bash
curl -X POST http://localhost:8081/api/v1/normalize/job-123
```

**Response:**
```json
{
  "job_id": "job-123",
  "source_language": "de",
  "normalized": {
    "tasks": "- Entwicklung von Webanwendungen\n- Code Review",
    "requirements": "- 3+ Jahre Erfahrung\n- Go/Python",
    "offer": "- Wettbewerbsfähiges Gehalt\n- Remote-Arbeit"
  },
  "saved_to_db": true
}
```

### 3. Translation Only (No Normalization)

**`POST /api/v1/translate/:id`**

- Translates the **original job description** directly
- Does NOT create tasks/requirements/offer fields
- Use when you want to keep the original format

```bash
curl -X POST http://localhost:8081/api/v1/translate/job-123
```

## Smart Skip Logic

The service automatically skips work that's already been done:

### Skip Already Normalized
```bash
curl -X POST http://localhost:8081/api/v1/normalize/job-123
# Response: { "skipped": true, "skip_reason": "already normalized" }
```

### Skip Already Translated Languages
If you request DE, FR, IT, EN but DE and FR already exist:
```bash
curl -X POST http://localhost:8081/api/v1/translate/job-123
# Only translates IT and EN
# Response includes: { "skipped_languages": ["de", "fr"] }
```

### Force Reprocessing
```bash
curl -X POST http://localhost:8081/api/v1/process/job-123 \
  -d '{"force": true}'
```

## Integration with Scrapper

The scrapper can automatically call this service after saving each job.

### Scrapper Environment Variables

```env
# URL of this service
AI_SERVICE_URL=http://localhost:8081

# Processing mode: none, process, normalize, translate
AI_PROCESSING_MODE=process
```

### Available Modes

| Mode | Description |
|------|-------------|
| `none` | Don't call AI service (default) |
| `process` | Normalize + translate each job |
| `normalize` | Only extract tasks/requirements/offer |
| `translate` | Only translate original description |

### Example Workflow

```bash
# 1. Start AI service
cd ai_job_processing && ./server

# 2. Run scrapper with AI processing
cd scrapper
AI_SERVICE_URL=http://localhost:8081 AI_PROCESSING_MODE=process ./scrapper scrape --cantons ZH --max-pages 5
```

Output:
```
Starting scrape...
  Strategy: full
  AI processing: process
AI service connected successfully

Run 5 completed: COMPLETED
  Jobs inserted: 50
  AI jobs processed: 45
  AI jobs skipped: 5 (already done)
```

## Database Schema

Migration 006 adds:

| Column | Type | Description |
|--------|------|-------------|
| `tasks` | TEXT | Normalized tasks |
| `requirements` | TEXT | Normalized requirements |
| `offer` | TEXT | Normalized offer |
| `is_normalized` | BOOLEAN | True if normalized |
| `normalized_at` | TIMESTAMPTZ | When normalized |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8081` | HTTP server port |
| `DATABASE_URL` | | PostgreSQL connection string |
| `GEMINI_API_KEY` | | Google Gemini API key (required) |
| `GEMINI_MODEL` | `gemini-2.0-flash` | Gemini model |
| `TARGET_LANGUAGES` | `de,fr,it,en` | Translation languages |
| `LOG_LEVEL` | `INFO` | DEBUG, INFO, WARN, ERROR |
| `LOG_FORMAT` | `json` | json or text |

## Error Codes

- `INVALID_REQUEST` - Bad request body
- `JOB_NOT_FOUND` - Job ID not in database
- `NORMALIZATION_ERROR` - AI normalization failed
- `TRANSLATION_ERROR` - AI translation failed
- `DATABASE_ERROR` - Database operation failed

## License

MIT License
