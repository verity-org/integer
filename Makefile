.PHONY: help build test test-coverage lint lint-vuln lint-workflows lint-yaml lint-shell lint-markdown fmt validate gen build-all quality clean install-tools

# Default target
help:
	@echo "Available targets:"
	@echo "  make build            - Build the integer binary"
	@echo "  make test             - Run unit tests"
	@echo "  make test-coverage    - Run tests with HTML coverage report"
	@echo "  make lint             - Run Go linter (golangci-lint with gofumpt, goimports, gosec)"
	@echo "  make fmt              - Auto-fix Go formatting and imports"
	@echo "  make lint-vuln        - Check for Go vulnerabilities"
	@echo "  make lint-workflows   - Lint GitHub Actions workflows"
	@echo "  make lint-yaml        - Lint YAML files"
	@echo "  make lint-shell       - Lint shell scripts"
	@echo "  make lint-markdown    - Lint markdown files"
	@echo "  make validate         - Validate all image configs"
	@echo "  make gen              - Generate all apko configs (no build)"
	@echo "  make build-all        - Build all images locally with apko (slow)"
	@echo "  make quality          - Run ALL linters, validate, and tests"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make install-tools    - Install development tools via mise"

# Build binary
build:
	go build -o integer .

# Run tests
test:
	go test -race -v ./...

# Run tests with coverage
test-coverage:
	go test -race -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run golangci-lint (includes gofumpt, goimports, gosec, and more)
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run: make install-tools" && exit 1)
	golangci-lint run --timeout=5m

# Auto-fix Go formatting and imports
fmt:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run: make install-tools" && exit 1)
	golangci-lint run --fix --timeout=5m

# Check for Go vulnerabilities
lint-vuln:
	@which govulncheck > /dev/null || (echo "govulncheck not found. Run: make install-tools" && exit 1)
	govulncheck ./...

# Lint GitHub Actions workflows
lint-workflows:
	@which actionlint > /dev/null || (echo "actionlint not found. Run: make install-tools" && exit 1)
	actionlint

# Lint YAML files (image configs, integer.yaml — excludes .github/ and packages/)
lint-yaml:
	@which yamllint > /dev/null || (echo "yamllint not found. Run: make install-tools" && exit 1)
	yamllint -c .yamllint.yml .

# Lint shell scripts
lint-shell:
	@which shellcheck > /dev/null || (echo "shellcheck not found. Run: make install-tools" && exit 1)
	shellcheck .github/scripts/*.sh

# Lint markdown files
lint-markdown:
	@if ! which markdownlint > /dev/null 2>&1; then echo "No markdown files to lint (markdownlint not installed)."; exit 0; fi
	@if ls *.md 2>/dev/null | grep -q .; then markdownlint "*.md" --ignore node_modules; else echo "No markdown files to lint."; fi

# Validate all image configs (schema + file existence)
validate: build
	./integer validate

# Generate all apko configs into ./gen/ (fast — no apko required)
gen: build
	./integer discover --gen-dir ./gen
	@echo "✓ Generated apko configs → gen/"

# Build all image variants locally with apko (amd64 only, no push)
# Runs apko for each of the 173+ variants — takes a long time.
# Narrow scope with: IMAGE=node make build-all
build-all: build
	@which apko > /dev/null || (echo "apko not found. Run: make install-tools" && exit 1)
	./integer discover --gen-dir ./gen | \
	  jq -r '.[] | [.name, .version, .type] | @tsv' | \
	  $(if $(IMAGE),grep "^$(IMAGE)	",cat) | \
	  while IFS=$$'\t' read -r name version type; do \
	    echo "Building $$name:$$version-$$type..."; \
	    apko build --arch amd64 \
	      "gen/$$name/$$version/$$type.apko.yaml" \
	      "$$name:$$version-$$type" \
	      /dev/null || exit 1; \
	  done
	@echo "✓ All images built"

# Run all quality checks
quality: lint lint-vuln lint-workflows lint-yaml lint-shell lint-markdown validate test
	@echo "✓ All quality checks passed!"

# Clean build artifacts
clean:
	rm -f integer
	rm -f coverage.out coverage.html

# Install development tools
install-tools:
	@echo "Installing tools via mise..."
	@which mise > /dev/null || (echo "mise not found. Install from: https://mise.jdx.dev" && exit 1)
	mise install
	@echo ""
	@echo "✓ All tools installed via mise!"
	@echo ""
	@echo "Installed tools:"
	@echo "  - go              (Go toolchain)"
	@echo "  - apko            (OCI image builder)"
	@echo "  - melange         (APK package builder)"
	@echo "  - trivy           (Vulnerability scanner)"
	@echo "  - cosign          (Image signing)"
	@echo "  - golangci-lint   (Go linter)"
	@echo "  - govulncheck     (Go vulnerability checker)"
	@echo "  - actionlint      (GitHub Actions linter)"
	@echo "  - yamllint        (YAML linter)"
	@echo "  - shellcheck      (Shell script linter)"
	@echo "  - markdownlint    (Markdown linter)"
	@echo ""
	@echo "Run 'mise list' to see all installed tools"
