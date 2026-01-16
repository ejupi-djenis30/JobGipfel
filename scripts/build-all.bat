@echo off
REM Build all services (Windows)

echo Building JobGipfel services...

cd auth_service
go mod tidy
go build -o auth_service.exe ./cmd/server
cd ..
echo [OK] auth_service

cd cv_generator
go mod tidy
go build -o cv_generator.exe ./cmd/server
cd ..
echo [OK] cv_generator

cd autoapply_service
go mod tidy
go build -o autoapply_service.exe ./cmd/server
cd ..
echo [OK] autoapply_service

cd job_search
go mod tidy
go build -o job_search.exe ./cmd/server
cd ..
echo [OK] job_search

cd matching_service
go mod tidy
go build -o matching_service.exe ./cmd/server
cd ..
echo [OK] matching_service

cd analytics_service
go mod tidy
go build -o analytics_service.exe ./cmd/server
cd ..
echo [OK] analytics_service

cd scrapper
go mod tidy
go build -o scrapper.exe ./cmd/scrapper
cd ..
echo [OK] scrapper

echo.
echo All services built successfully!
