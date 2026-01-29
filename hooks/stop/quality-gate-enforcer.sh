#!/bin/bash
# AIDA Quality Gate Enforcer Hook
# Purpose: Prevent exit until all quality gates pass
# Exit code 0 = allow exit, Exit code 2 = block exit
#
# This hook implements ralph-loop style enforcement:
# - Intercepts Stop events during AIDA implementation phase
# - Runs quality gates to verify test counts and coverage
# - Blocks exit if requirements not met, forcing iteration
# - Only allows exit when ALL gates pass
#
# Anti-infinite-loop protections (Issue #217):
# - Maximum 5 iterations before forced exit
# - Progress detection: if stuck for 3 iterations, allow exit
# - Each iteration generates a targeted fix plan

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source common utilities
source "$PROJECT_ROOT/scripts/lib/common.sh"

# Use CLAUDE_PROJECT_DIR if available, otherwise use detected root
if [[ -n "${CLAUDE_PROJECT_DIR:-}" ]]; then
    PROJECT_ROOT="$CLAUDE_PROJECT_DIR"
fi

# ============================================
# Anti-infinite-loop configuration
# ============================================
MAX_ITERATIONS=5
STUCK_THRESHOLD=3  # Allow exit if no progress for this many iterations

# ============================================
# Check if AIDA session is active
# ============================================
SESSION_FILE="$PROJECT_ROOT/.aida/state/session.json"
if [[ ! -f "$SESSION_FILE" ]]; then
    # No active session, allow exit
    exit 0
fi

# ============================================
# Get project name from session
# ============================================
PROJECT=$(jq -r '.project_name // empty' "$SESSION_FILE" 2>/dev/null)
if [[ -z "$PROJECT" ]]; then
    # No project defined, allow exit
    exit 0
fi

# ============================================
# Check if in implementation phase
# ============================================
CURRENT_PHASE=$(jq -r '.current_phase // empty' "$SESSION_FILE" 2>/dev/null)
if [[ "$CURRENT_PHASE" != "IMPL_PHASE" ]]; then
    # Not in implementation phase, allow exit
    exit 0
fi

# ============================================
# Check if project directory exists
# ============================================
PROJECT_DIR="$PROJECT_ROOT/$PROJECT"
if [[ ! -d "$PROJECT_DIR" ]]; then
    # Project directory not created yet, allow exit
    exit 0
fi

# ============================================
# Iteration tracking (Issue #217)
# ============================================
CURRENT_ITERATION=$(jq -r '.iteration // 1' "$SESSION_FILE" 2>/dev/null)
ITERATION_HISTORY=$(jq -r '.iteration_history // []' "$SESSION_FILE" 2>/dev/null)

echo "=== AIDA Quality Gate Enforcer ===" >&2
echo "Project: $PROJECT" >&2
echo "Phase: $CURRENT_PHASE" >&2
echo "Iteration: $CURRENT_ITERATION / $MAX_ITERATIONS" >&2
echo "" >&2

# ============================================
# Check for max iterations reached
# ============================================
if [[ $CURRENT_ITERATION -ge $MAX_ITERATIONS ]]; then
    echo "=== MAX ITERATIONS REACHED ($MAX_ITERATIONS) ===" >&2
    echo "" >&2
    echo "Allowing exit to prevent infinite loop." >&2
    echo "Quality gates may not be fully satisfied." >&2
    echo "Consider running /aida:fix to address remaining issues." >&2
    echo "" >&2

    # Update session
    jq '.forced_exit = true | .exit_reason = "max_iterations"' "$SESSION_FILE" > "${SESSION_FILE}.tmp" && \
        mv "${SESSION_FILE}.tmp" "$SESSION_FILE" 2>/dev/null || true

    output_allow "Max iterations reached - allowing exit to prevent infinite loop" \
        "Quality gates may not be fully satisfied. Consider running /aida:fix to address remaining issues."
    exit 0
fi

# Check for container runtime availability (podman or docker)
CONTAINER_RUNTIME=""
if command -v podman &>/dev/null; then
    CONTAINER_RUNTIME="podman"
elif command -v docker &>/dev/null; then
    CONTAINER_RUNTIME="docker"
fi

# Run quality gates with Docker/Podman if available
GATE_RESULT=0
if [[ -n "$CONTAINER_RUNTIME" ]]; then
    echo "Container runtime detected: $CONTAINER_RUNTIME" >&2
    echo "Running full quality gates including Docker and E2E..." >&2
    "$PROJECT_ROOT/scripts/quality-gates.sh" "$PROJECT" 2>&1 || GATE_RESULT=$?
else
    echo "No container runtime found, skipping Docker gates" >&2
    "$PROJECT_ROOT/scripts/quality-gates.sh" "$PROJECT" --skip-docker 2>&1 || GATE_RESULT=$?
fi

echo "" >&2

if [[ $GATE_RESULT -ne 0 ]]; then
    # Quality gates failed - check for stuck state and generate fix plan

    # ============================================
    # Check for stuck state (Issue #217)
    # ============================================
    # Count how many recent iterations had the same failure count
    STUCK_COUNT=$(jq -r --argjson result "$GATE_RESULT" '
        [.iteration_history[-3:]? // [] | .[].gate_result] |
        map(select(. == $result)) | length
    ' "$SESSION_FILE" 2>/dev/null || echo "0")

    if [[ $STUCK_COUNT -ge $STUCK_THRESHOLD ]]; then
        echo "=== STUCK DETECTED ($STUCK_COUNT iterations with no progress) ===" >&2
        echo "" >&2
        echo "Allowing exit to prevent infinite loop." >&2
        echo "The same issues have persisted for $STUCK_COUNT iterations." >&2
        echo "" >&2
        echo "Suggestions:" >&2
        echo "  1. Review the approach - maybe a different strategy is needed" >&2
        echo "  2. Run /aida:analyze to understand the current state" >&2
        echo "  3. Ask the user for guidance on priorities" >&2
        echo "" >&2

        # Update session
        jq '.forced_exit = true | .exit_reason = "stuck_detected"' "$SESSION_FILE" > "${SESSION_FILE}.tmp" && \
            mv "${SESSION_FILE}.tmp" "$SESSION_FILE" 2>/dev/null || true

        output_allow "Stuck detected - same issues persisted for multiple iterations" \
            "Consider a different approach: 1) Run /aida:analyze 2) Ask user for guidance 3) Review strategy"
        exit 0
    fi

    # ============================================
    # Update iteration counter and history
    # ============================================
    NEXT_ITERATION=$((CURRENT_ITERATION + 1))
    jq --argjson iter "$NEXT_ITERATION" --argjson result "$GATE_RESULT" '
        .iteration = $iter |
        .iteration_history = ((.iteration_history // []) + [{
            "iteration": ($iter - 1),
            "timestamp": (now | todate),
            "gate_result": $result
        }])
    ' "$SESSION_FILE" > "${SESSION_FILE}.tmp" && \
        mv "${SESSION_FILE}.tmp" "$SESSION_FILE" 2>/dev/null || true

    # ============================================
    # Generate targeted fix plan
    # ============================================
    FIX_PLAN=""
    if [[ -x "$PROJECT_ROOT/scripts/generate-fix-plan.sh" ]]; then
        FIX_PLAN=$("$PROJECT_ROOT/scripts/generate-fix-plan.sh" "$PROJECT" "$NEXT_ITERATION" 2>/dev/null || echo "")
    fi

    echo "=== QUALITY GATES NOT PASSED (Iteration $CURRENT_ITERATION/$MAX_ITERATIONS) ===" >&2
    echo "" >&2
    echo "You must fix the following issues before completing:" >&2
    echo "  - Ensure Backend has 80+ tests" >&2
    echo "  - Ensure Frontend has 100+ tests" >&2
    echo "  - Ensure Backend coverage is 75%+" >&2
    echo "  - Ensure Frontend coverage is 70%+" >&2
    echo "  - Ensure E2E has 20+ tests" >&2
    echo "" >&2

    if [[ -n "$FIX_PLAN" ]]; then
        echo "Priority action: $FIX_PLAN" >&2
        echo "" >&2
    fi

    echo "Remaining iterations: $((MAX_ITERATIONS - CURRENT_ITERATION))" >&2
    echo "Continue implementation and try again." >&2
    echo "" >&2

    # Output JSON response to block exit (Official Claude Code format)
    REASON="Iteration $CURRENT_ITERATION/$MAX_ITERATIONS: Quality gates not passed."
    SYSTEM_MSG="Requirements: Backend 80+ tests, Frontend 100+ tests, Coverage 75%/70%, E2E 20+ tests."
    if [[ -n "$FIX_PLAN" ]]; then
        SYSTEM_MSG="$SYSTEM_MSG Priority: $FIX_PLAN"
    fi
    SYSTEM_MSG="$SYSTEM_MSG Remaining iterations: $((MAX_ITERATIONS - CURRENT_ITERATION))"

    output_block "$REASON" "$SYSTEM_MSG"
    exit 0  # JSON is only processed with exit 0
fi

# ============================================
# All gates passed - allow exit
# ============================================
echo "=== ALL QUALITY GATES PASSED ===" >&2
echo "" >&2
echo "DONE - Implementation complete!" >&2

# Update session to mark completion
if command -v jq &>/dev/null; then
    jq '.quality_gates_passed = true' "$SESSION_FILE" > "${SESSION_FILE}.tmp" && \
        mv "${SESSION_FILE}.tmp" "$SESSION_FILE" 2>/dev/null || true
fi

# Output JSON response to allow exit (Official Claude Code format)
output_allow "All quality gates passed successfully"
exit 0
