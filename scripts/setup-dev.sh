#!/bin/bash
#
# Developer environment setup script for go-trust
#
# This script sets up the development environment including:
# - Installing Git hooks
# - Installing development tools
# - Verifying Go version
# - Running initial checks
#

set -e

echo "🚀 Setting up go-trust development environment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: go.mod not found. Run this from the project root.${NC}"
    exit 1
fi

echo -e "${BLUE}Step 1: Checking Go version...${NC}"
GO_VERSION=$(grep -E '^go [0-9]+\.[0-9]+' go.mod | sed 's/go //g' | tr -d ' ')
CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')

if ! go version | grep -q "go${GO_VERSION}"; then
    echo -e "${YELLOW}⚠️  Warning: Go version mismatch${NC}"
    echo -e "   Required: ${GO_VERSION}"
    echo -e "   Current:  ${CURRENT_GO}"
    echo -e "   This may cause issues. Continue? [y/N] "
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo -e "${GREEN}✓ Go version ${GO_VERSION} detected${NC}"
fi

echo -e "${BLUE}Step 2: Installing development tools...${NC}"
make tools
echo -e "${GREEN}✓ Development tools installed${NC}"

echo -e "${BLUE}Step 3: Setting up Git hooks...${NC}"
if [ -d ".git" ]; then
    # Install pre-commit hook
    if [ -f "scripts/pre-commit.sh" ]; then
        ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
        chmod +x scripts/pre-commit.sh
        chmod +x .git/hooks/pre-commit
        echo -e "${GREEN}✓ Pre-commit hook installed${NC}"
    else
        echo -e "${YELLOW}⚠️  Pre-commit script not found${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Not a git repository, skipping hook installation${NC}"
fi

echo -e "${BLUE}Step 4: Downloading dependencies...${NC}"
go mod download
echo -e "${GREEN}✓ Dependencies downloaded${NC}"

echo -e "${BLUE}Step 5: Running initial checks...${NC}"

# Format check
echo -e "  ${YELLOW}→ Checking code formatting...${NC}"
make fmt
echo -e "  ${GREEN}✓ Code formatted${NC}"

# Vet check
echo -e "  ${YELLOW}→ Running go vet...${NC}"
make vet
echo -e "  ${GREEN}✓ Vet passed${NC}"

# Run tests
echo -e "  ${YELLOW}→ Running tests...${NC}"
if make test > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓ Tests passed${NC}"
else
    echo -e "  ${YELLOW}⚠️  Some tests failed (this might be expected)${NC}"
fi

echo ""
echo -e "${GREEN}✅ Development environment setup complete!${NC}"
echo ""
echo -e "${BLUE}Useful commands:${NC}"
echo -e "  ${YELLOW}make help${NC}        - Show all available make targets"
echo -e "  ${YELLOW}make test${NC}        - Run all tests"
echo -e "  ${YELLOW}make lint${NC}        - Run linters"
echo -e "  ${YELLOW}make build${NC}       - Build the binary"
echo -e "  ${YELLOW}make coverage${NC}    - Generate coverage report"
echo -e "  ${YELLOW}make bench${NC}       - Run benchmarks"
echo ""
echo -e "${BLUE}VS Code users:${NC}"
echo -e "  - Install recommended extensions when prompted"
echo -e "  - Settings are pre-configured in .vscode/settings.json"
echo -e "  - Debug configurations available in .vscode/launch.json"
echo ""
echo -e "${GREEN}Happy coding! 🎉${NC}"
