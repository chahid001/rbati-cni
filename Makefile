# ──────────────────────────────
#        CONFIGURATION
# ──────────────────────────────
APP_NAME      := rbati-cni
BIN_DIR       := bin
DEPLOY_DIR    := deploy
RELEASE_DIR   := release
BRANCH        := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT        := $(shell git rev-parse --short HEAD)
TAG           := $(shell git describe --tags --exact-match 2>/dev/null)

# The VERSION is now overridable from the command line, with a default
VERSION       ?= development

RELEASE_NAME  := $(APP_NAME)-$(VERSION)
# ──────────────────────────────
#        DEV PURPOSES
# ──────────────────────────────
REMOTE_USER   := schahid
REMOTE_HOST   := 192.168.0.170
REMOTE_PATH   := /home/schahid/workspace/p2/config
SSH_KEY       := ~/.ssh/id_rsa

# Colors
BOLD=\033[1m
RESET=\033[0m
GREEN=\033[32m
BLUE=\033[34m
YELLOW=\033[33m
CYAN=\033[36m
CHECKMARK=$(GREEN)✔$(RESET)

# ──────────────────────────────
#        DEFAULT TARGET
# ──────────────────────────────
.PHONY: all
all: build

# ──────────────────────────────
#        BUILD TARGET
# ──────────────────────────────
.PHONY: build
build:
	@echo "$(BLUE)🚀 Building Go binaries...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/rbati-d cmd/cni/main.go
	@echo "$(CHECKMARK) Binaries built successfully."

buildDev:
	@echo "$(BLUE)🚀 Building Go binaries for dev env...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/rbati-d cmd/cni/main.go
	@GOOS=linux GOARCH=amd64 go build -o $(BIN_DIR)/rbati-agent cmd/agent/main.go
	@echo "$(CHECKMARK) Binaries built successfully."

# ──────────────────────────────
#     SCP TO REMOTE TARGET
# ──────────────────────────────
.PHONY: scp
scp:
	@echo "$(CYAN)📦 Copying binaries to remote dev machine...$(RESET)"
	@scp -i $(SSH_KEY) -r $(RELEASE_DIR) $(REMOTE_USER)@$(REMOTE_HOST):$(REMOTE_PATH)
	@echo "$(CHECKMARK) Files copied."

# ──────────────────────────────
#       INIT MODULE TARGET
# ──────────────────────────────
.PHONY: init
init:
	@echo "$(YELLOW)🔧 Initializing Go module...$(RESET)"
	@go mod init $(APP_NAME)
	@echo "$(CHECKMARK) Module initialized."

# ──────────────────────────────
#       CREATE RELEASE
# ──────────────────────────────
.PHONY: release
release: build
	@echo "$(BLUE)🎉 Creating release: $(RELEASE_NAME)...$(RESET)"
	@rm -rf $(RELEASE_DIR)
	@mkdir -p $(RELEASE_DIR)/$(RELEASE_NAME)

	@# Copy binaries and deployment scripts
	@cp -r $(BIN_DIR) $(RELEASE_DIR)/$(RELEASE_NAME)/
	@cp -r $(DEPLOY_DIR)/*.sh $(RELEASE_DIR)/$(RELEASE_NAME)/
	@cp $(DEPLOY_DIR)/conf/10-rbati.conf $(RELEASE_DIR)/$(RELEASE_NAME)/

	@# Optional loader animation
	@printf "📦 Packaging release"
	@for i in . .. ...; do sleep 0.3 && printf "$$i\r"; done    
	
	@cd $(RELEASE_DIR) && zip -rq $(RELEASE_NAME).zip $(RELEASE_NAME)
	@cd $(RELEASE_DIR) && tar -zcf $(RELEASE_NAME).tar.gz $(RELEASE_NAME)
	@echo "$(CHECKMARK) Release archives created."

# ──────────────────────────────
#          CLEANUP
# ──────────────────────────────
.PHONY: clean
clean:
	@echo "$(YELLOW)🧹 Cleaning up build and release artifacts...$(RESET)"
	@rm -rf $(BIN_DIR) $(RELEASE_DIR)
	@echo "$(CHECKMARK) Cleanup complete."

# ──────────────────────────────
#           HELP TARGET
# ──────────────────────────────
.PHONY: help
help:
	@echo ""
	@echo "\033[1m📦 $(APP_NAME) Makefile — Available Targets:\033[0m"
	@echo "────────────────────────────────────────────────────────"
	@echo "  🆘  \033[36mhelp\033[0m         Show this help message"
	@echo "  🛠️   \033[36mall\033[0m          Build default binary (rbati-d only)"
	@echo "  🧱  \033[36mbuild\033[0m        Build \033[1mrbati-d\033[0m (minimal default)"
	@echo "  🧪  \033[36mbuildDev\033[0m     Build \033[1mrbati-d\033[0m and \033[1mrbati-agent\033[0m (dev use)"
	@echo "  🧰  \033[36minit\033[0m         Initialize Go module as \033[1m$(APP_NAME)\033[0m"
	@echo "  📦  \033[36mrelease\033[0m      Create release zip with binaries and scripts"
	@echo "  🚀  \033[36mscp\033[0m          Copy release folder to remote dev machine"
	@echo "  🧹  \033[36mclean\033[0m        Clean up build and release artifacts"
	@echo ""
	@echo "\033[1m🔖 Version:\033[0m $(VERSION) from branch '\033[33m$(BRANCH)\033[0m'"
	@echo "\033[1m📁 Output:\033[0m  $(RELEASE_DIR)/$(RELEASE_NAME).zip"
	@echo ""