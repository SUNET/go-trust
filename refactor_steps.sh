#!/bin/bash
# Refactoring script to split steps.go into smaller files
# This script will be executed carefully to ensure tests pass after each step

set -e  # Exit on error

cd "$(dirname "$0")"

echo "Starting refactoring of steps.go..."

# Backup the original file
cp pkg/pipeline/steps.go pkg/pipeline/steps.go.backup

# Run this refactoring in controlled phases with git commits after each

echo "Phase 1: Creating step_registry.go..."
echo "This will be done manually to ensure proper imports and structure"

echo ""
echo "Manual steps required:"
echo "1. Extract lines 1-73 (registry) to step_registry.go"  
echo "2. Extract lines 74-540 (generate) to step_generate.go"
echo "3. Extract lines 541-733 (load) to step_load.go"
echo "4. Extract lines 734-852 (fetch options) to step_fetch_options.go"
echo "5. Extract lines 853-1106 (select) to step_select.go"
echo "6. Extract lines 1107-1148 (echo+log) to step_log.go"
echo "7. Extract lines 1149-1527 (publish) to step_publish.go"
echo "8. Create steps_init.go with init() function (lines 1528-1539)"
echo "9. Remove extracted content from steps.go"
echo "10. Run tests after each extraction"
