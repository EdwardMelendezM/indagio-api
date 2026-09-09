.PHONY: help swagger build run test clean install-tools

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

BINARY_NAME := foro-api
GO := go
SWAG := swag

help: ## Display this help screen
	@echo "$(BLUE)Hilos Backend - Build Commands$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}'
	@echo ""

install-tools: ## Install CLI tools (swag)
	@echo "$(BLUE)Installing tools...$(NC)"
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install github.com/swaggo/swag@latest
	@echo "$(GREEN)✅ Tools installed$(NC)"

swagger: ## Generate Swagger documentation
	@echo "$(BLUE)Generating Swagger documentation...$(NC)"
	$(SWAG) init -d . -o internal/docs
	@echo "$(GREEN)✅ Swagger generated successfully!$(NC)"
	@echo "   View at: http://localhost:8080/swagger/index.html"

build: swagger ## Build the application
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	$(GO) build -o $(BINARY_NAME) .
	@echo "$(GREEN)✅ Build complete: ./$(BINARY_NAME)$(NC)"

run: ## Run the application (with swagger generation)
	@echo "$(BLUE)Starting $(BINARY_NAME)...$(NC)"
	$(GO) run main.go

run-build: build ## Build and run the application
	./$(BINARY_NAME)

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	$(GO) test -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report: coverage.html$(NC)"

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning up...$(NC)"
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "$(GREEN)✅ Clean complete$(NC)"

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GO) fmt ./...
	@echo "$(GREEN)✅ Formatting complete$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	$(GO) vet ./...
	@echo "$(GREEN)✅ Vet complete$(NC)"

deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

lint: fmt vet ## Run formatter and vet
	@echo "$(GREEN)✅ Linting complete$(NC)"

dev: swagger ## Develop mode (watch and rebuild)
	@echo "$(BLUE)Swagger generated!$(NC)"
	@echo "$(YELLOW)Run 'go run main.go' to start the server$(NC)"

# Database commands (optional)
db-migrate-up: ## Run database migrations up
	@echo "$(BLUE)Running migrations...$(NC)"
	migrate -path db/migrations -database "$(DATABASE_URL)" up
	@echo "$(GREEN)✅ Migrations complete$(NC)"

db-migrate-down: ## Rollback database migrations
	@echo "$(BLUE)Rolling back migrations...$(NC)"
	migrate -path db/migrations -database "$(DATABASE_URL)" down
	@echo "$(GREEN)✅ Rollback complete$(NC)"

docker-build: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(NC)"
	docker build -t foro-unsaac-api:latest .
	@echo "$(GREEN)✅ Docker image built$(NC)"

docker-run: ## Run Docker container
	@echo "$(BLUE)Running Docker container...$(NC)"
	docker run -p 8080:8080 foro-unsaac-api:latest
	@echo "$(GREEN)✅ Container running on http://localhost:8080$(NC)"

.DEFAULT_GOAL := help

# Create mocks for authentication domain
create_mocks:
	mockery --dir=internal/domain --name=AuthUsecase      --filename=auth_usecase_mock.go      --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=UserRepository   --filename=user_repository_mock.go   --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=OTPRepository    --filename=otp_repository_mock.go    --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=EmailService     --filename=email_service_mock.go     --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=TokenService     --filename=token_service_mock.go     --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=PasswordService  --filename=password_service_mock.go  --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=AdminUserRepository --filename=admin_user_repository_mock.go --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=AdminUsecase --filename=admin_usecase_mock.go --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=ModerationUsecase --filename=moderation_usecase_mock.go --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=UserRepository --filename=user_repository_mock.go --output=internal/domain/mocks --outpkg=mocks
	# Sticker feature (PR 1)
	mockery --dir=internal/domain --name=StickerRepository  --filename=sticker_repository_mock.go  --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=StickerUsecase     --filename=sticker_usecase_mock.go     --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=StickerJobQueue    --filename=sticker_job_queue_mock.go   --output=internal/domain/mocks --outpkg=mocks
	# Avatar borders feature (PR 4)
	mockery --dir=internal/domain --name=AvatarBorderRepository --filename=avatar_border_repository_mock.go --output=internal/domain/mocks --outpkg=mocks
	mockery --dir=internal/domain --name=AvatarBorderUsecase    --filename=avatar_border_usecase_mock.go    --output=internal/domain/mocks --outpkg=mocks




.PHONY: create_auth_mocks


