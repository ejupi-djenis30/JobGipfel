# Job Scrapper

A CLI-based microservice for scraping job listings from [job-room.ch](https://www.job-room.ch) and storing them directly in PostgreSQL. Designed to be run as a scheduled cron job.

## Features

- **Smart Scraping**: Full or incremental strategies with automatic stop on unchanged jobs
- **Advanced Filtering**: Canton, workload, contract type, keywords, date range
- **Polite Mode**: Configurable delays (2-5s) with User-Agent rotation
- **Normalized Storage**: PostgreSQL with normalized tables; companies and locations are deduplicated
- **Run Telemetry**: Track scraping progress with detailed metrics
- **Production Ready**: Comprehensive logging, error handling, and retry logic
- **CLI Only**: No REST API, designed for scheduled execution

## Quick Start

### Prerequisites

- Go 1.23+
- PostgreSQL 14+
- Access to your PostgreSQL database

### Installation

```bash
# Clone the repository
cd scrapper

# Download dependencies
go mod download

# Build the binary
go build -o scrapper ./cmd/scrapper
```

### Configuration

Copy the example environment file and configure your database:

```bash
cp .env.example .env
```

Edit `.env` with your PostgreSQL connection string:

```env
DATABASE_URL=postgres://user:password@host:5432/database?sslmode=require
```

### Run Migrations

```bash
./scrapper migrate
```

### Run a Scrape

```bash
# Scrape jobs from Zurich canton
./scrapper scrape --cantons ZH --max-pages 3

# Full scrape of all cantons (may take a while)
./scrapper scrape --polite
```

## CLI Commands

```bash
scrapper scrape [options]    # Run a scraping job
scrapper jobs [options]      # List jobs from database
scrapper runs [options]      # List scrape runs
scrapper migrate [options]   # Run database migrations
scrapper version             # Show version information
scrapper help                # Show help
```

### Scrape Options

| Flag | Default | Description |
|------|---------|-------------|
| `--database` | `$DATABASE_URL` | PostgreSQL connection string |
| `--strategy` | `full` | `full` or `incremental` |
| `--max-pages` | `0` | Pages to scrape (0 = unlimited) |
| `--start-page` | `0` | Page to start from |
| `--keywords` | | Search keywords |
| `--cantons` | | Comma-separated cantons (e.g., ZH,BE,GE) |
| `--days-back` | `60` | Jobs published within N days |
| `--workload-min` | `10` | Minimum workload % |
| `--workload-max` | `100` | Maximum workload % |
| `--permanent` | | `true`=permanent, `false`=temporary |
| `--polite` | `true` | Enable delays between requests |

### Jobs Options

| Flag | Default | Description |
|------|---------|-------------|
| `--database` | `$DATABASE_URL` | PostgreSQL connection string |
| `--limit` | `20` | Number of jobs to list |
| `--offset` | `0` | Pagination offset |
| `--json` | `false` | Output as JSON |

### Runs Options

| Flag | Default | Description |
|------|---------|-------------|
| `--database` | `$DATABASE_URL` | PostgreSQL connection string |
| `--limit` | `20` | Number of runs to list |
| `--json` | `false` | Output as JSON |

### Migrate Options

| Flag | Default | Description |
|------|---------|-------------|
| `--database` | `$DATABASE_URL` | PostgreSQL connection string |
| `--direction` | `up` | `up` or `down` |
| `--steps` | `0` | Number of migrations (0 = all) |
| `--force` | `-1` | Force migration version (recovery) |

## Scrape Strategies

### Full Strategy

Scrapes all available pages up to `--max-pages`. Use for initial database population.

```bash
./scrapper scrape --strategy full --max-pages 100 --cantons ZH
```

### Incremental Strategy

Stops when it encounters a job that hasn't been updated since the last scrape. Use for daily updates.

```bash
./scrapper scrape --strategy incremental --days-back 7
```

## Cron Job Examples

### Linux/macOS

```bash
# Daily incremental scrape at 2 AM
0 2 * * * cd /path/to/scrapper && ./scrapper scrape --strategy incremental --days-back 7 >> /var/log/scrapper.log 2>&1

# Weekly full scrape on Sunday at 3 AM
0 3 * * 0 cd /path/to/scrapper && ./scrapper scrape --strategy full --max-pages 200 >> /var/log/scrapper.log 2>&1
```

### Windows Task Scheduler

Create a scheduled task that runs:

```powershell
cd C:\path\to\scrapper
.\scrapper.exe scrape --strategy incremental --days-back 7
```

## Per-Canton Scraping

For large-scale scraping, scrape each canton separately to avoid API limits:

```powershell
# PowerShell script for all cantons
$cantons = @("AG", "AI", "AR", "BE", "BL", "BS", "FR", "GE", "GL", "GR", 
             "JU", "LU", "NE", "NW", "OW", "SG", "SH", "SO", "SZ", "TG", 
             "TI", "UR", "VD", "VS", "ZG", "ZH")

foreach ($canton in $cantons) {
    Write-Host "Scraping canton: $canton"
    .\scrapper.exe scrape --cantons $canton --strategy incremental --days-back 7
}
```

## Database Schema

The database uses a normalized schema with the following tables:

| Table | Description |
|-------|-------------|
| `companies` | Employer information (deduplicated) |
| `locations` | Job locations (deduplicated) |
| `jobs` | Main job listings |
| `employments` | Employment terms (1:1 with jobs) |
| `publications` | Publication settings (1:1 with jobs) |
| `apply_channels` | Application methods (1:1 with jobs) |
| `job_descriptions` | Titles/descriptions per language (1:many) |
| `occupations` | Occupation codes (1:many) |
| `scrape_runs` | Telemetry for scrape runs |

### Flexible Schema

The `jobs` table includes a `source` field that supports:
- `jobroom`: Jobs scraped from job-room.ch
- `platform`: Jobs registered directly through your platform

This allows you to use the same database for both scraped and platform-native jobs.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | | PostgreSQL connection string (required) |
| `LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `LOG_FORMAT` | `text` | `text` or `json` |
| `SCRAPER_DELAY_MIN_MS` | `2000` | Minimum delay between requests (ms) |
| `SCRAPER_DELAY_MAX_MS` | `5000` | Maximum delay between requests (ms) |
| `SCRAPER_DEFAULT_MAX_PAGES` | `0` | Default max pages (0 = unlimited) |
| `SCRAPER_DEFAULT_DAYS_BACK` | `60` | Default days back filter |

## Swiss Canton Codes

```
AG (Aargau)          AI (Appenzell I.Rh.)   AR (Appenzell A.Rh.)
BE (Bern)            BL (Basel-Land)        BS (Basel-Stadt)
FR (Fribourg)        GE (Geneva)            GL (Glarus)
GR (Graubünden)      JU (Jura)              LU (Lucerne)
NE (Neuchâtel)       NW (Nidwalden)         OW (Obwalden)
SG (St. Gallen)      SH (Schaffhausen)      SO (Solothurn)
SZ (Schwyz)          TG (Thurgau)           TI (Ticino)
UR (Uri)             VD (Vaud)              VS (Valais)
ZG (Zug)             ZH (Zurich)
```

## Project Structure

```
scrapper/
├── cmd/scrapper/
│   └── main.go              # CLI entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Environment configuration
│   ├── db/
│   │   ├── postgres.go      # Database connection
│   │   └── migrate.go       # Migration runner
│   ├── logger/
│   │   └── logger.go        # Structured logging
│   ├── models/
│   │   ├── job.go           # Domain models
│   │   └── filters.go       # Scrape filters
│   ├── scraper/
│   │   ├── client.go        # HTTP client
│   │   └── runner.go        # Scrape orchestration
│   └── store/
│       └── store.go         # Database operations
├── migrations/
│   ├── 001_create_companies.up.sql
│   ├── 002_create_locations.up.sql
│   ├── 003_create_jobs.up.sql
│   ├── 004_create_child_tables.up.sql
│   └── 005_create_scrape_runs.up.sql
├── .env.example
├── .gitignore
├── go.mod
└── README.md
```

## Development

### Build

```bash
go build -o scrapper ./cmd/scrapper
```

### Test

```bash
go test ./... -v
```

### Run Locally

```bash
# Set environment variables
export DATABASE_URL="postgres://localhost:5432/jobgipfel?sslmode=disable"
export LOG_LEVEL=DEBUG

# Run migrations
go run ./cmd/scrapper migrate

# Run a test scrape
go run ./cmd/scrapper scrape --max-pages 1 --cantons ZH
```

## Troubleshooting

### Migration Errors

If a migration fails halfway, you may need to force the version:

```bash
./scrapper migrate --force 3
./scrapper migrate --direction up
```

### API Rate Limiting

If you receive 412 errors, the API has rate-limited you. Solutions:
1. Increase delay: Set `SCRAPER_DELAY_MIN_MS=5000` and `SCRAPER_DELAY_MAX_MS=10000`
2. Scrape per-canton instead of all at once
3. Wait and retry later

### Connection Issues

Verify your PostgreSQL connection:

```bash
psql "$DATABASE_URL" -c "SELECT 1"
```

## License

MIT License - see [LICENSE](../LICENSE) for details.
