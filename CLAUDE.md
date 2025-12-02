# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Postulator is a desktop application for automated WordPress blog post creation using AI providers. It's built with a Go backend (Wails framework) and a Next.js/React frontend. The app runs as a desktop application with system tray integration.

## Build Commands

### Full Application
```bash
wails dev          # Development mode with hot reload
wails build        # Production build
```

### Backend (Go)
```bash
make test          # Run all Go tests
make test-verbose  # Run tests with verbose output
make test-coverage # Run tests with coverage report
make go-lint       # Run golangci-lint
make go-fmt        # Format with gofumpt
```

### Frontend (Next.js)
```bash
cd frontend
npm run dev        # Development server
npm run build      # Production build
npm run lint       # ESLint
```

## Architecture

### Dependency Injection
The application uses **uber-go/fx** for dependency injection. All modules are organized in three main fx.Module definitions:
- `internal/infra/module.go` - Infrastructure layer (database, WordPress client, secrets, events)
- `internal/domain/module.go` - Domain services and repositories
- `internal/handlers/module.go` - Handler layer exposed to frontend

### Handler Bindings
Handlers in `internal/handlers/` are bound to the Wails runtime and exposed to the frontend. The `App.GetBinds()` method returns all handlers that become callable from JavaScript.

### Job Execution Pipeline
The core feature is the job execution pipeline (`internal/domain/jobs/execution/`):
- Uses a **command pattern** with a state machine
- Pipeline builder pattern in `pipeline/orchestrator.go`
- Commands are in `commands/` subdirectories organized by phase (validation, selection, generation, publishing, tracking)
- State transitions: Initialized -> TopicSelected -> CategorySelected -> ExecutionCreated -> PromptRendered -> ContentGenerated -> OutputValidated -> Published -> Completed
- Supports retry strategies with exponential backoff
- Error handling via fault package with actions: Retry, Fail, Pause, Recover, Continue

### Domain Structure
Each domain follows the pattern:
- `model.go` - Domain models
- `repository.go` - Data access interface and implementation
- `service.go` - Business logic

Key domains: articles, categories, jobs, topics, prompts, providers, sites, stats, healthcheck, linking

### AI Providers
AI integration in `internal/infra/ai/` supports:
- OpenAI
- Anthropic
- Google

Factory pattern creates clients based on provider type via `CreateClient()`.

### Frontend-Backend Communication
Wails generates TypeScript bindings in `frontend/src/wailsjs/` from Go handler methods. Frontend calls Go methods directly via these generated bindings.

### Database
Uses SQLite via `ncruces/go-sqlite3`. Migrations are in `internal/infra/database/migrator/migrations/`. The `pkg/dbx/` package provides query building utilities with Squirrel.

### Event Bus
Global event bus (`internal/infra/events/bus.go`) for decoupled communication, particularly for pipeline events that update UI.

## Key Packages

- `pkg/di/` - Custom typed DI container utilities
- `pkg/dbx/` - Database query builder helpers
- `pkg/errors/` - Custom error types
- `pkg/ctx/` - Context utilities
- `pkg/logger/` - Zerolog-based structured logging
