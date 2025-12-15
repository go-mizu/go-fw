# Makefile

BOOK := BOOK.md

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show help
	@echo ""
	@echo "go-fw"
	@echo "-----"
	@echo ""
	@grep -E '^[a-zA-Z0-9_\-]+:.*?## ' $(MAKEFILE_LIST) | \
	  sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
	@echo ""

.PHONY: book
book: ## Build BOOK.md from README.md files
	@bash scripts/book.sh

.PHONY: check
check: ## Generate code from README.md, sync workspace, build all, smoke test
	@go run ./cmd/extract
	@bash scripts/work.sh
	@bash scripts/build.sh
	@bash scripts/smoke.sh
	@echo ""
	@echo "OK: all examples compile and start"
	@echo ""

.PHONY: clean
clean: ## Remove generated artifacts
	@rm -f $(BOOK)

# Advanced targets (not listed in help)
.PHONY: _work _build _smoke _extract
_extract:
	@go run ./cmd/extract
_work:
	@bash scripts/work.sh
_build:
	@bash scripts/build.sh
_smoke:
	@bash scripts/smoke.sh
