#!/bin/bash
# AIDA Fix Plan Generator
# Purpose: Analyze quality gate failures and generate actionable fix plans
# Usage: ./generate-fix-plan.sh <project_name> [iteration]
#
# This script analyzes the last gate run and produces a structured
# fix plan to guide the agent in addressing failures.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common utilities
source "$SCRIPT_DIR/lib/common.sh"

# Use CLAUDE_PROJECT_DIR if available
if [[ -n "${CLAUDE_PROJECT_DIR:-}" ]]; then
    PROJECT_ROOT="$CLAUDE_PROJECT_DIR"
fi

PROJECT="${1:-}"
ITERATION="${2:-1}"

if [[ -z "$PROJECT" ]]; then
    echo "Usage: $0 <project_name> [iteration]" >&2
    exit 1
fi

# ============================================
# Paths
# ============================================
PROJECT_DIR="$PROJECT_ROOT/$PROJECT"
GATE_LOG="$PROJECT_ROOT/.aida/logs/gate-run.log"
SESSION_FILE="$PROJECT_ROOT/.aida/state/session.json"
FIX_PLAN_DIR="$PROJECT_ROOT/.aida/fix-plans"

# Use ensure_dir from common.sh
ensure_dir "$FIX_PLAN_DIR"

# ============================================
# Analyze gate failures
# ============================================
analyze_failures() {
    local failures=()

    # Check backend tests
    if [[ -d "$PROJECT_DIR/backend" ]]; then
        local backend_tests
        backend_tests=$(grep -r "func Test" "$PROJECT_DIR/backend" --include="*_test.go" 2>/dev/null | wc -l)
        if [[ $backend_tests -lt 80 ]]; then
            failures+=("BACKEND_TESTS:${backend_tests}:80:Add $((80 - backend_tests)) more Go test functions")
        fi
    fi

    # Check frontend tests
    if [[ -d "$PROJECT_DIR/frontend/src" ]]; then
        local frontend_tests
        frontend_tests=$(grep -rE "^\s*(it|test)\s*\(" "$PROJECT_DIR/frontend/src" --include="*.test.tsx" --include="*.test.ts" 2>/dev/null | wc -l)
        if [[ $frontend_tests -lt 100 ]]; then
            failures+=("FRONTEND_TESTS:${frontend_tests}:100:Add $((100 - frontend_tests)) more Jest/Vitest test cases")
        fi
    fi

    # Check E2E tests
    if [[ -d "$PROJECT_DIR/frontend/e2e" ]]; then
        local e2e_tests
        e2e_tests=$(grep -rE "^\s*(it|test)\s*\(" "$PROJECT_DIR/frontend/e2e" --include="*.spec.ts" --include="*.test.ts" 2>/dev/null | wc -l)
        if [[ $e2e_tests -lt 20 ]]; then
            failures+=("E2E_TESTS:${e2e_tests}:20:Add $((20 - e2e_tests)) more E2E test cases")
        fi
    fi

    # Check TDD evidence
    local tdd_dir="$PROJECT_ROOT/.aida/tdd-evidence"
    if [[ -d "$tdd_dir" ]]; then
        local tdd_count
        tdd_count=$(find "$tdd_dir" -name "*.json" -type f 2>/dev/null | wc -l)
        if [[ $tdd_count -lt 10 ]]; then
            failures+=("TDD_EVIDENCE:${tdd_count}:10:Record $((10 - tdd_count)) more TDD cycles using tdd-logger.sh")
        fi
    else
        failures+=("TDD_EVIDENCE:0:10:Create .aida/tdd-evidence/ and record 10 TDD cycles")
    fi

    # Check for build errors
    if [[ -d "$PROJECT_DIR/backend" ]]; then
        if ! (cd "$PROJECT_DIR/backend" && go build ./... 2>/dev/null); then
            failures+=("BUILD_BACKEND:0:1:Fix Go build errors in backend/")
        fi
    fi

    if [[ -d "$PROJECT_DIR/frontend" && -f "$PROJECT_DIR/frontend/package.json" ]]; then
        if ! (cd "$PROJECT_DIR/frontend" && npm run build 2>/dev/null); then
            failures+=("BUILD_FRONTEND:0:1:Fix TypeScript/build errors in frontend/")
        fi
    fi

    printf '%s\n' "${failures[@]}"
}

# ============================================
# Generate fix plan
# ============================================
generate_plan() {
    local timestamp
    timestamp=$(date -Iseconds)
    local plan_file="$FIX_PLAN_DIR/fix-plan-${ITERATION}.json"

    echo "Analyzing gate failures for iteration $ITERATION..." >&2

    local failures
    failures=$(analyze_failures)

    if [[ -z "$failures" ]]; then
        echo "No failures detected. All gates should pass." >&2
        cat << EOF
{
  "iteration": $ITERATION,
  "timestamp": "$timestamp",
  "status": "all_gates_pass",
  "failures": [],
  "priority_actions": []
}
EOF
        return 0
    fi

    # Parse failures into JSON array
    local failure_json="["
    local priority_actions="["
    local first=true

    while IFS= read -r failure; do
        [[ -z "$failure" ]] && continue

        IFS=':' read -r gate current required action <<< "$failure"

        if [[ "$first" != true ]]; then
            failure_json+=","
            priority_actions+=","
        fi
        first=false

        failure_json+="{\"gate\":\"$gate\",\"current\":$current,\"required\":$required,\"action\":\"$action\"}"
        priority_actions+="\"$action\""
    done <<< "$failures"

    failure_json+="]"
    priority_actions+="]"

    local plan
    plan=$(cat << EOF
{
  "iteration": $ITERATION,
  "timestamp": "$timestamp",
  "status": "needs_fixes",
  "failures": $failure_json,
  "priority_actions": $priority_actions,
  "next_steps": [
    "Focus on the highest priority failure first",
    "Use TDD: write failing test, implement, verify green",
    "Run quality-gates.sh after each fix to track progress",
    "Continue until all gates pass"
  ]
}
EOF
)

    echo "$plan" | jq . > "$plan_file"
    echo "$plan"
}

# ============================================
# Update session with iteration info
# ============================================
update_session() {
    if [[ -f "$SESSION_FILE" ]]; then
        local updated
        updated=$(jq --argjson iter "$ITERATION" '
            .iteration = $iter |
            .iteration_history += [{
                "iteration": $iter,
                "timestamp": (now | todate)
            }]
        ' "$SESSION_FILE")
        echo "$updated" > "$SESSION_FILE"
    fi
}

# ============================================
# Main
# ============================================
update_session
plan=$(generate_plan)

# Output summary for hook consumption
failure_count=$(echo "$plan" | jq '.failures | length')
if [[ $failure_count -gt 0 ]]; then
    first_action=$(echo "$plan" | jq -r '.priority_actions[0] // "Check quality gates"')
    echo "Iteration $ITERATION: $failure_count failures. Priority: $first_action"
else
    echo "Iteration $ITERATION: All gates should pass"
fi
