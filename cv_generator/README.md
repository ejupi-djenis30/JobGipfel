# CV Generator Service

Microservice for generating professional CVs/Resumes using AI-powered HTML/CSS generation and PDF conversion.

## Features

- **AI-Powered Design**: Gemini generates beautiful, styled HTML/CSS
- **Multiple Styles**: Modern, Minimalist, Classic, Creative
- **Customizable**: Color schemes, sections, photo inclusion
- **PDF Export**: High-quality A4 PDFs with proper pagination
- **Multi-Language**: Generate CVs in English, German, French, Italian, Spanish

## Prerequisites

- **Go 1.23+**
- **Chrome/Chromium** for PDF generation
- **Running auth_service** with user profile data

## Quick Start

1. **Copy environment file**:
   ```bash
   cp .env.example .env
   ```

2. **Configure** (required):
   - `GEMINI_API_KEY`: Your Gemini API key
   - `AUTH_SERVICE_URL`: URL of auth_service (default: http://localhost:8082)

3. **Start server**:
   ```bash
   go run ./cmd/server
   ```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/api/v1/cv/generate` | Generate CV as PDF |
| POST | `/api/v1/cv/preview` | Generate CV preview (HTML) |
| GET | `/api/v1/cv/styles` | List available styles |
| GET | `/api/v1/cv/options` | Get all customization options |

## Generate CV Request

```json
{
  "style": "modern",
  "include_photo": true,
  "color_scheme": "blue",
  "sections": {
    "summary": true,
    "experiences": true,
    "education": true,
    "skills": true,
    "certifications": true
  },
  "max_experiences": 5,
  "max_education": 3,
  "max_skills": 15,
  "language": "en",
  "custom_instructions": "Make it concise and impactful"
}
```

## Example Usage

```bash
# Generate PDF
curl -X POST http://localhost:8083/api/v1/cv/generate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"style": "modern", "color_scheme": "blue"}' \
  --output resume.pdf

# Preview HTML
curl -X POST http://localhost:8083/api/v1/cv/preview \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"style": "minimalist"}'
```

## Styles

| Style | Description |
|-------|-------------|
| `modern` | Clean, contemporary with subtle accents |
| `minimalist` | Ultra-clean, lots of whitespace |
| `classic` | Traditional professional layout |
| `creative` | Bold colors, unique layouts |

## Color Schemes

`blue`, `green`, `dark`, `neutral`, `purple`, `red`
