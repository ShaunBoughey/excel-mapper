# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Excel Mapper is a Go service for mapping and transforming fields between Excel (XLSX) and CSV files. It provides both a web UI (served at /) and REST API endpoints.

## Common Commands

```bash
# Run the application
export API_KEYS="your-api-key-1,your-api-key-2" && go run main.go

# Run all tests
go test ./...

# Run a specific test
go test -v -run TestName

# Run tests with race detection
go test -race ./...

# Format code
go fmt ./...

# Install/update dependencies
go mod tidy

# Regenerate Swagger docs (requires swag CLI)
swag init
```

## Architecture

**Backend**: Standard library `net/http` server running on port 8080.

**API Endpoints** (under `/api/v1/`):
- `GET /config` - Returns field configuration from `config/field_config.json`
- `POST /process` - Processes uploaded files with field mappings, outputs XLSX/CSV/Markdown

**Authentication**: API key via `X-API-Key` header. Keys loaded from `API_KEYS` environment variable (comma-separated).

**Key Dependencies**:
- `github.com/xuri/excelize/v2` - Excel file handling
- `github.com/swaggo/http-swagger` - Swagger UI

## Key Files

- `main.go` - HTTP server setup, handlers, and core file processing logic
- `main_test.go` - Integration and unit tests
- `auth/auth.go` - API key authentication middleware
- `config/field_config.go` - Field configuration logic
- `config/field_config.json` - Field definitions (name, displayName, isMandatory)
- `ui/` - Frontend assets for web interface

## Environment Setup

```bash
export API_KEYS="your-api-key-1,your-api-key-2"
```

## Testing Patterns

Tests use `httptest.NewRequest` and `httptest.NewRecorder` for handler testing. See `main_test.go` for examples of multipart form file upload tests.

## Git Workflow

**CRITICAL: Always create a branch BEFORE making any commits!**

### Proper Workflow for Changes

1. **Create branch FIRST** (before any code changes):
   ```bash
   git checkout -b feature/descriptive-name
   ```

2. Make your changes and commits on the branch

3. Push the branch:
   ```bash
   git push -u origin feature/descriptive-name
   ```

4. Create a Pull Request from the branch to main

### ❌ NEVER Do This:
```bash
# Don't commit directly to main!
git checkout main
git add .
git commit -m "changes"  # ❌ WRONG - now main has commits it shouldn't
```

### ✅ Always Do This:
```bash
# Create branch first, then commit
git checkout -b feature/my-changes  # ✅ CORRECT - branch first!
git add .
git commit -m "changes"
git push -u origin feature/my-changes
```

### Recovery from Mistakes

If commits were accidentally made to main:
```bash
# Reset local main
git checkout main
git reset --hard origin/main

# Force push to clean remote (use with caution!)
git push --force origin main

# Your commits are safe on the feature branch
git checkout feature/my-changes
```
