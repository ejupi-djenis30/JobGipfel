@echo off
REM Start all services in separate windows

echo Starting JobGipfel services...

start "Auth Service :8082" cmd /k "cd auth_service && go run ./cmd/server"
start "CV Generator :8083" cmd /k "cd cv_generator && go run ./cmd/server"
start "AutoApply :8084" cmd /k "cd autoapply_service && go run ./cmd/server"
start "Job Search :8085" cmd /k "cd job_search && go run ./cmd/server"
start "Matching :8086" cmd /k "cd matching_service && go run ./cmd/server"
start "Analytics :8087" cmd /k "cd analytics_service && go run ./cmd/server"

echo.
echo All services starting...
echo.
echo Ports:
echo   Auth:        http://localhost:8082
echo   CV Gen:      http://localhost:8083
echo   AutoApply:   http://localhost:8084
echo   Search:      http://localhost:8085
echo   Matching:    http://localhost:8086
echo   Analytics:   http://localhost:8087
