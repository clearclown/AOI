#!/bin/bash
# AIDA TDD Logger
# Purpose: Record TDD (RED-GREEN-REFACTOR) evidence for Gate 20
# Usage: ./tdd-logger.sh <command> [args]
#
# Commands:
#   start <feature>    - Start new TDD cycle
#   red <test_file>    - Record RED phase (test must fail)
#   green <test_file>  - Record GREEN phase (test must pass)
#   refactor           - Record REFACTOR phase
#   complete           - Complete and save evidence
#   status             - Show current cycle status
#   list               - List all evidence files

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common utilities
source "$SCRIPT_DIR/lib/common.sh"

# Use CLAUDE_PROJECT_DIR if available
if [[ -n "${CLAUDE_PROJECT_DIR:-}" ]]; then
    PROJECT_ROOT="$CLAUDE_PROJECT_DIR"
fi

EVIDENCE_DIR="$PROJECT_ROOT/.aida/tdd-evidence"
CURRENT_FILE="$EVIDENCE_DIR/.current-cycle.json"
COMMAND="${1:-help}"
shift || true

ensure_dir "$EVIDENCE_DIR"

# ============================================
# Helper functions
# ============================================
timestamp() {
    date -Iseconds
}

run_test() {
    local test_file="$1"
    local result=0

    # Detect test type and run
    if [[ "$test_file" == *"_test.go" ]]; then
        # Go test
        go test -v "$test_file" 2>&1 || result=$?
    elif [[ "$test_file" == *.test.ts ]] || [[ "$test_file" == *.test.tsx ]]; then
        # Jest/Vitest test
        local test_dir
        test_dir=$(dirname "$test_file")
        (cd "$test_dir" && npm test -- --testPathPattern="$(basename "$test_file")" 2>&1) || result=$?
    elif [[ "$test_file" == *.spec.ts ]]; then
        # Playwright test
        local test_dir
        test_dir=$(dirname "$test_file")
        (cd "$test_dir" && npx playwright test "$(basename "$test_file")" 2>&1) || result=$?
    elif [[ "$test_file" == *_test.py ]] || [[ "$test_file" == *test_*.py ]]; then
        # Python test
        python -m pytest "$test_file" -v 2>&1 || result=$?
    else
        echo "Unknown test type: $test_file" >&2
        return 1
    fi

    return $result
}

# ============================================
# Start new TDD cycle
# ============================================
start_cycle() {
    local feature="${1:-}"

    if [[ -z "$feature" ]]; then
        echo "Usage: tdd-logger.sh start <feature_name>" >&2
        exit 1
    fi

    # Check if there's an incomplete cycle
    if [[ -f "$CURRENT_FILE" ]]; then
        local status
        status=$(jq -r '.status' "$CURRENT_FILE")
        if [[ "$status" != "complete" ]]; then
            echo "Warning: Incomplete cycle exists. Complete it first or delete $CURRENT_FILE" >&2
            jq . "$CURRENT_FILE"
            exit 1
        fi
    fi

    # Create new cycle
    cat > "$CURRENT_FILE" << EOF
{
  "feature": "$feature",
  "started_at": "$(timestamp)",
  "status": "started",
  "red_phase": null,
  "green_phase": null,
  "refactor_phase": null
}
EOF

    echo "Started TDD cycle for: $feature"
    echo "Next step: ./tdd-logger.sh red <test_file>"
}

# ============================================
# Record RED phase
# ============================================
record_red() {
    local test_file="${1:-}"

    if [[ -z "$test_file" ]]; then
        echo "Usage: tdd-logger.sh red <test_file>" >&2
        exit 1
    fi

    if [[ ! -f "$CURRENT_FILE" ]]; then
        echo "No active TDD cycle. Run: ./tdd-logger.sh start <feature>" >&2
        exit 1
    fi

    echo "=== RED Phase ===" >&2
    echo "Running test (expecting FAILURE): $test_file" >&2
    echo "" >&2

    local exit_code=0
    local output
    output=$(run_test "$test_file" 2>&1) || exit_code=$?

    echo "$output" >&2
    echo "" >&2

    if [[ $exit_code -eq 0 ]]; then
        echo "ERROR: Test passed! RED phase requires a FAILING test." >&2
        echo "Write a test that fails first, then implement the feature." >&2
        exit 1
    fi

    # Record RED phase
    local updated
    updated=$(jq --arg file "$test_file" --argjson code "$exit_code" --arg out "$output" '
        .status = "red" |
        .red_phase = {
            "test_file": $file,
            "exit_code": $code,
            "timestamp": (now | todate),
            "output_snippet": ($out | split("\n") | last | . // "")
        }
    ' "$CURRENT_FILE")

    echo "$updated" > "$CURRENT_FILE"

    echo "RED phase recorded (test failed as expected)" >&2
    echo "Next step: Implement the feature, then run: ./tdd-logger.sh green $test_file" >&2
}

# ============================================
# Record GREEN phase
# ============================================
record_green() {
    local test_file="${1:-}"

    if [[ -z "$test_file" ]]; then
        echo "Usage: tdd-logger.sh green <test_file>" >&2
        exit 1
    fi

    if [[ ! -f "$CURRENT_FILE" ]]; then
        echo "No active TDD cycle. Run: ./tdd-logger.sh start <feature>" >&2
        exit 1
    fi

    local status
    status=$(jq -r '.status' "$CURRENT_FILE")
    if [[ "$status" != "red" ]]; then
        echo "Must complete RED phase first. Current status: $status" >&2
        exit 1
    fi

    echo "=== GREEN Phase ===" >&2
    echo "Running test (expecting SUCCESS): $test_file" >&2
    echo "" >&2

    local exit_code=0
    local output
    output=$(run_test "$test_file" 2>&1) || exit_code=$?

    echo "$output" >&2
    echo "" >&2

    if [[ $exit_code -ne 0 ]]; then
        echo "ERROR: Test still failing. Implement the feature first." >&2
        exit 1
    fi

    # Record GREEN phase
    local updated
    updated=$(jq --arg file "$test_file" --argjson code "$exit_code" '
        .status = "green" |
        .green_phase = {
            "test_file": $file,
            "exit_code": $code,
            "timestamp": (now | todate)
        }
    ' "$CURRENT_FILE")

    echo "$updated" > "$CURRENT_FILE"

    echo "GREEN phase recorded (test passed)" >&2
    echo "Next step: ./tdd-logger.sh refactor (optional) or ./tdd-logger.sh complete" >&2
}

# ============================================
# Record REFACTOR phase
# ============================================
record_refactor() {
    if [[ ! -f "$CURRENT_FILE" ]]; then
        echo "No active TDD cycle." >&2
        exit 1
    fi

    local status
    status=$(jq -r '.status' "$CURRENT_FILE")
    if [[ "$status" != "green" ]]; then
        echo "Must complete GREEN phase first. Current status: $status" >&2
        exit 1
    fi

    # Record REFACTOR phase
    local updated
    updated=$(jq '
        .status = "refactored" |
        .refactor_phase = {
            "timestamp": (now | todate),
            "completed": true
        }
    ' "$CURRENT_FILE")

    echo "$updated" > "$CURRENT_FILE"

    echo "REFACTOR phase recorded" >&2
    echo "Next step: ./tdd-logger.sh complete" >&2
}

# ============================================
# Complete cycle and save evidence
# ============================================
complete_cycle() {
    if [[ ! -f "$CURRENT_FILE" ]]; then
        echo "No active TDD cycle." >&2
        exit 1
    fi

    local status
    status=$(jq -r '.status' "$CURRENT_FILE")
    if [[ "$status" != "green" ]] && [[ "$status" != "refactored" ]]; then
        echo "Must complete at least GREEN phase. Current status: $status" >&2
        exit 1
    fi

    local feature
    feature=$(jq -r '.feature' "$CURRENT_FILE")
    local safe_name
    safe_name=$(echo "$feature" | tr ' ' '-' | tr -cd '[:alnum:]-')
    local evidence_file="$EVIDENCE_DIR/${safe_name}-$(date +%Y%m%d-%H%M%S).json"

    # Finalize
    local final
    final=$(jq '
        .status = "complete" |
        .completed_at = (now | todate)
    ' "$CURRENT_FILE")

    echo "$final" > "$evidence_file"
    rm -f "$CURRENT_FILE"

    echo "=== TDD Cycle Complete ===" >&2
    echo "Feature: $feature" >&2
    echo "Evidence saved: $evidence_file" >&2
    echo "" >&2

    # Count total evidence
    local count
    count=$(find "$EVIDENCE_DIR" -name "*.json" -type f ! -name ".current-cycle.json" | wc -l)
    echo "Total TDD evidence files: $count / 10 required" >&2
}

# ============================================
# Show status
# ============================================
show_status() {
    if [[ -f "$CURRENT_FILE" ]]; then
        echo "=== Current TDD Cycle ===" >&2
        jq . "$CURRENT_FILE"
    else
        echo "No active TDD cycle." >&2
    fi

    echo "" >&2
    echo "=== Evidence Summary ===" >&2
    local count
    count=$(find "$EVIDENCE_DIR" -name "*.json" -type f ! -name ".current-cycle.json" 2>/dev/null | wc -l)
    echo "Total evidence files: $count / 10 required for Gate 20" >&2
}

# ============================================
# List evidence
# ============================================
list_evidence() {
    echo "=== TDD Evidence Files ===" >&2

    if [[ ! -d "$EVIDENCE_DIR" ]]; then
        echo "No evidence directory yet." >&2
        return 0
    fi

    local count=0
    for f in "$EVIDENCE_DIR"/*.json; do
        [[ -f "$f" ]] || continue
        [[ "$(basename "$f")" == ".current-cycle.json" ]] && continue

        count=$((count + 1))
        local feature
        feature=$(jq -r '.feature // "unknown"' "$f")
        local completed
        completed=$(jq -r '.completed_at // "incomplete"' "$f")
        echo "$count. $feature (completed: $completed)"
    done

    if [[ $count -eq 0 ]]; then
        echo "No evidence files yet." >&2
    fi

    echo "" >&2
    echo "Total: $count / 10 required" >&2
}

# ============================================
# Print help
# ============================================
print_help() {
    cat << 'EOF'
AIDA TDD Logger - Record RED-GREEN-REFACTOR evidence

Usage:
  ./tdd-logger.sh <command> [args]

Commands:
  start <feature>    Start new TDD cycle for a feature
  red <test_file>    Record RED phase (test must FAIL)
  green <test_file>  Record GREEN phase (test must PASS)
  refactor           Record REFACTOR phase (optional)
  complete           Complete and save evidence
  status             Show current cycle status
  list               List all evidence files

TDD Workflow:
  1. Write a failing test (RED)
  2. Implement just enough to pass (GREEN)
  3. Clean up the code (REFACTOR)
  4. Save evidence for Gate 20

Example:
  ./tdd-logger.sh start "User authentication"
  ./tdd-logger.sh red backend/internal/auth/auth_test.go
  # ... implement feature ...
  ./tdd-logger.sh green backend/internal/auth/auth_test.go
  ./tdd-logger.sh refactor
  ./tdd-logger.sh complete

Gate 20 requires 10+ evidence files to pass.

EOF
}

# ============================================
# Main
# ============================================
case "$COMMAND" in
    start)
        start_cycle "$@"
        ;;
    red)
        record_red "$@"
        ;;
    green)
        record_green "$@"
        ;;
    refactor)
        record_refactor
        ;;
    complete)
        complete_cycle
        ;;
    status)
        show_status
        ;;
    list)
        list_evidence
        ;;
    help|--help|-h)
        print_help
        ;;
    *)
        echo "Unknown command: $COMMAND" >&2
        print_help
        exit 1
        ;;
esac
