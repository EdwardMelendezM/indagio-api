# Copilot instructions for indagio-api

## Project snapshot
- This repo is a Go API using Gin and a clean-architecture split under `internal/`.
- The module is `indagio-api` (`go.mod`), but some legacy files still import `foro-unsaac-backend/...` paths; treat that as a repo-specific migration issue rather than a valid current import pattern.
- The main architectural layers are `internal/domain`, `internal/usecase`, `internal/repository`, `internal/delivery`, and `middleware`.
- Storage, workers, and websocket infrastructure live under `internal/repository/storage`, `internal/delivery/worker`, and `internal/delivery/ws`.

## Build, test, and lint commands
### Standard commands
- `make help` — show the available project commands.
- `make build` — runs Swagger generation first and then builds the binary with `go build -o foro-api .`.
- `make run` — starts the app with `go run main.go`.
- `make test` — runs `go test -v ./...`.
- `make fmt` — runs `gofmt` across the repo.
- `make vet` — runs `go vet ./...`.
- `make lint` — does the repo's formatter + vet workflow.
- `make swagger` — generates Swagger docs via `swag init -d . -o internal/docs`.
- Direct Swagger generation: `swag init -d . -o internal/docs`.

### Single-test examples
Use package-scoped `go test` commands when narrowing to one area:
- `go test ./internal/usecase/auth -run TestAuthUsecase_Register -v`
- `go test ./internal/delivery/http/auth -run TestAuthHandler_Login_Success -v`
- `go test ./middleware -run TestOptionalAuth_ValidToken_InjectsUserIDAndRole -v`

### Important repo gotcha
- `go test ./...` is not a reliable validation command in the current checkout because there is a stale legacy file, `main-from-another-project.go`, that still imports `foro-unsaac-backend/...` packages.
- That file does not match the repo's declared module path and is a known copy-paste artifact from another project; do not treat it as the current app entrypoint.
- If the repo is mid-migration, validate the package you are changing instead of assuming the whole tree is green.

## High-level architecture
- `internal/config/config.go` is the single source of truth for environment variables and app configuration; prefer it for any new config access.
- `internal/domain` defines the business entities and repository interfaces (`UserRepository`, `OTPRepository`, `AuthUsecase`, etc.). This is the dependency boundary for the rest of the app.
- `internal/usecase/...` contains business logic and is the place where use-case orchestration happens.
- `internal/repository/...` contains infrastructure implementations (Postgres repositories and Cloudflare/R2 storage integrations).
- `internal/delivery/http/...` contains Gin handlers and DTOs for HTTP endpoints; the handlers should stay thin and delegate to the use case layer.
- `internal/delivery/ws/...` and `internal/delivery/worker/...` handle websocket presence/hub logic and asynchronous worker flows.
- `middleware/...` contains JWT/auth and authorization checks, including optional-auth logic and admin guards.
- `internal/utils/...` provides concrete services used by the use cases (JWT, password hashing, email, admin helpers).

## Key conventions specific to this repo
- Keep the dependency direction clean: handlers -> use cases -> domain interfaces; infrastructure implementations are behind the interfaces.
- Prefer constructing concrete services through constructors that return interfaces; the use-case layer is designed around dependency injection.
- Environment access is centralized in `internal/config/config.go`; avoid scattering `os.Getenv` calls elsewhere.
- Functionality is often tested beside the code under test, with `*_test.go` files colocated in the same package directories.
- This repo contains legacy drift: `main.go` is minimal/stub-like while `main-from-another-project.go` contains an older application scaffold. Treat those as clues to a partial migration, not as a stable canonical app layout.
- If you are working on a specific package, validate that package by path instead of assuming the full root build is healthy until the legacy import mismatch is resolved.

## Practical guidance for future Copilot sessions
- Start from the package you are editing, not from the repository root, when validating behavior.
- When the build fails with import errors mentioning `foro-unsaac-backend`, assume the issue is stale legacy code or a module mismatch rather than a new compiler problem in the feature you are touching.
- Keep edits local to the feature being addressed unless the task explicitly requires cleaning up the stale app skeleton or import path drift.
