#!/bin/bash
# AIDA TDD Verification
# Verifies that TDD protocol was followed
#
# Usage: ./scripts/verify-tdd.sh <project-name> [component]
#
# Components:
#   backend  - Verify Go backend tests
#   frontend - Verify React frontend tests
#   all      - Verify both (default)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common utilities
source "$SCRIPT_DIR/lib/common.sh"

# Minimum requirements
MIN_BACKEND_TESTS=5
MIN_FRONTEND_TESTS=3
MIN_TEST_COVERAGE=50

# Parse arguments
PROJECT=${1:-}
COMPONENT=${2:-all}

if [[ -z "$PROJECT" ]]; then
    echo "Usage: $0 <project-name> [component]"
    echo ""
    echo "Components:"
    echo "  backend  - Verify Go backend tests"
    echo "  frontend - Verify React frontend tests"
    echo "  all      - Verify both (default)"
    exit 1
fi

# Validate project name
if ! validate_project_name "$PROJECT"; then
    exit 1
fi

PROJECT_DIR="$PROJECT_ROOT/$PROJECT"

if [[ ! -d "$PROJECT_DIR" ]]; then
    log_error "Project directory not found: $PROJECT_DIR"
    exit 1
fi

cd "$PROJECT_ROOT"

log_section "AIDA TDD Verification - $PROJECT"
echo "Component: $COMPONENT"
echo "Timestamp: $(date -Iseconds)"

ERRORS=0
WARNINGS=0

# ============================================
# Backend TDD Verification
# ============================================
verify_backend_tdd() {
    log_section "Backend TDD Verification"

    local backend_dir="$PROJECT_DIR/backend"

    if [[ ! -d "$backend_dir" ]]; then
        log_error "Backend directory not found"
        ERRORS=$((ERRORS + 1))
        return
    fi

    cd "$backend_dir"

    # ----------------------------------------
    # Check test file count
    # ----------------------------------------
    log_info "Checking test file count..."

    local test_count=$(find . -name "*_test.go" -type f | wc -l)
    if [[ $test_count -lt $MIN_BACKEND_TESTS ]]; then
        log_error "Insufficient test files: $test_count (minimum $MIN_BACKEND_TESTS)"
        ERRORS=$((ERRORS + 1))
    else
        log_success "Test file count: $test_count (minimum $MIN_BACKEND_TESTS)"
    fi

    # ----------------------------------------
    # Run tests and capture output
    # ----------------------------------------
    log_info "Running Go tests..."

    local test_output="/tmp/aida_backend_tests_$PROJECT.txt"

    if go test ./... -v > "$test_output" 2>&1; then
        log_success "All Go tests passed"

        # Count test functions
        local test_func_count=$(grep -c "=== RUN" "$test_output" || echo "0")
        log_info "Test functions executed: $test_func_count"

        if [[ $test_func_count -lt $MIN_BACKEND_TESTS ]]; then
            log_warning "Few test functions: $test_func_count"
            WARNINGS=$((WARNINGS + 1))
        fi
    else
        log_error "Go tests failed"
        echo "--- Test Output ---"
        tail -30 "$test_output"
        echo "--- End Output ---"
        ERRORS=$((ERRORS + 1))
    fi

    # ----------------------------------------
    # Check test coverage (if available)
    # ----------------------------------------
    log_info "Checking test coverage..."

    local coverage_output="/tmp/aida_backend_coverage_$PROJECT.txt"

    if go test ./... -cover > "$coverage_output" 2>&1; then
        # Extract coverage percentage
        local coverage=$(grep -oP 'coverage: \K[0-9.]+' "$coverage_output" | head -1)

        if [[ -n "$coverage" ]]; then
            local coverage_int=${coverage%.*}
            if [[ $coverage_int -ge $MIN_TEST_COVERAGE ]]; then
                log_success "Test coverage: ${coverage}% (minimum ${MIN_TEST_COVERAGE}%)"
            else
                log_warning "Low test coverage: ${coverage}% (minimum ${MIN_TEST_COVERAGE}%)"
                WARNINGS=$((WARNINGS + 1))
            fi
        else
            log_info "Coverage data not available for all packages"
        fi
    fi

    # ----------------------------------------
    # Check test patterns (TDD indicators)
    # ----------------------------------------
    log_info "Checking TDD patterns..."

    # Check for table-driven tests
    local table_tests=$(grep -r "tests := \[\]struct\|testCases :=" --include="*_test.go" | wc -l)
    if [[ $table_tests -gt 0 ]]; then
        log_success "Table-driven tests found: $table_tests"
    else
        log_info "No table-driven tests detected"
    fi

    # Check for test helpers
    local test_helpers=$(grep -r "func Test\|func Benchmark" --include="*_test.go" | wc -l)
    log_info "Test/Benchmark functions: $test_helpers"

    # Check for assertions
    local assertions=$(grep -r "t\.Error\|t\.Fatal\|t\.Fail\|assert\.\|require\." --include="*_test.go" | wc -l)
    log_info "Assertions found: $assertions"

    cd "$PROJECT_ROOT"
}

# ============================================
# Frontend TDD Verification
# ============================================
verify_frontend_tdd() {
    log_section "Frontend TDD Verification"

    local frontend_dir="$PROJECT_DIR/frontend"

    if [[ ! -d "$frontend_dir" ]]; then
        log_error "Frontend directory not found"
        ERRORS=$((ERRORS + 1))
        return
    fi

    cd "$frontend_dir"

    # Check if node_modules exists
    if [[ ! -d "node_modules" ]]; then
        log_info "Installing dependencies..."
        npm install > /dev/null 2>&1
    fi

    # ----------------------------------------
    # Check test file count
    # ----------------------------------------
    log_info "Checking test file count..."

    local test_count=$(find src -name "*.test.tsx" -o -name "*.test.ts" -type f 2>/dev/null | wc -l)
    if [[ $test_count -lt $MIN_FRONTEND_TESTS ]]; then
        log_error "Insufficient test files: $test_count (minimum $MIN_FRONTEND_TESTS)"
        ERRORS=$((ERRORS + 1))
    else
        log_success "Test file count: $test_count (minimum $MIN_FRONTEND_TESTS)"
    fi

    # ----------------------------------------
    # Run tests and capture output
    # ----------------------------------------
    log_info "Running frontend tests..."

    local test_output="/tmp/aida_frontend_tests_$PROJECT.txt"

    if npm test -- --run > "$test_output" 2>&1; then
        log_success "All frontend tests passed"

        # Count test cases
        local test_case_count=$(grep -c "✓\|PASS" "$test_output" || echo "0")
        log_info "Test cases passed: $test_case_count"
    else
        log_error "Frontend tests failed"
        echo "--- Test Output ---"
        tail -30 "$test_output"
        echo "--- End Output ---"
        ERRORS=$((ERRORS + 1))
    fi

    # ----------------------------------------
    # Check test patterns
    # ----------------------------------------
    log_info "Checking test patterns..."

    # Check for testing-library usage
    local rtl_imports=$(grep -r "@testing-library/react" --include="*.test.tsx" --include="*.test.ts" | wc -l)
    if [[ $rtl_imports -gt 0 ]]; then
        log_success "React Testing Library usage found: $rtl_imports files"
    else
        log_warning "React Testing Library not detected"
        WARNINGS=$((WARNINGS + 1))
    fi

    # Check for render calls
    local render_calls=$(grep -r "render(" --include="*.test.tsx" | wc -l)
    log_info "Component render calls: $render_calls"

    # Check for user event / fireEvent
    local user_events=$(grep -r "userEvent\|fireEvent" --include="*.test.tsx" | wc -l)
    log_info "User interaction tests: $user_events"

    # Check for assertions
    local assertions=$(grep -r "expect(" --include="*.test.tsx" --include="*.test.ts" | wc -l)
    log_info "Assertions found: $assertions"

    cd "$PROJECT_ROOT"
}

# ============================================
# Execute Verification
# ============================================
case $COMPONENT in
    backend)
        verify_backend_tdd
        ;;
    frontend)
        verify_frontend_tdd
        ;;
    all)
        verify_backend_tdd
        verify_frontend_tdd
        ;;
    *)
        log_error "Unknown component: $COMPONENT"
        echo "Valid components: backend, frontend, all"
        exit 1
        ;;
esac

# ============================================
# Summary
# ============================================
echo ""
log_section "TDD Verification Summary"

echo "Errors: $ERRORS"
echo "Warnings: $WARNINGS"

if [[ $ERRORS -eq 0 ]]; then
    if [[ $WARNINGS -eq 0 ]]; then
        log_success "TDD verification passed with no issues"
    else
        log_success "TDD verification passed with $WARNINGS warning(s)"
    fi
    exit 0
else
    log_error "TDD verification failed with $ERRORS error(s)"
    echo ""
    echo "TDD Requirements:"
    echo "  - Minimum $MIN_BACKEND_TESTS backend test files"
    echo "  - Minimum $MIN_FRONTEND_TESTS frontend test files"
    echo "  - All tests must pass"
    echo "  - Recommended ${MIN_TEST_COVERAGE}%+ coverage"
    exit 1
fi
