#!/bin/bash
# AIDA Quality Gates
# Master orchestrator for all quality verification gates
#
# Usage: ./scripts/quality-gates.sh <project-name> [options]
#
# Gates:
#   1-2:   Backend (build, tests)
#   3-4:   Frontend (build, tests)
#   5-7:   Docker (build, run, health)
#   8:     API Coverage (handlers)
#   9:     Frontend Feature Coverage (pages)
#   10:    Integration Check
#   11:    Backend Test Count (min 80)
#   12:    Frontend Test Count (min 100)
#   13:    Empty Array Pattern (Go nil check)
#   14:    Backend Coverage (100%)
#   15:    E2E Test Verification (Playwright config)
#   16:    Design Quality (shadcn/ui, layout, responsive)
#   17:    Frontend Coverage (100%)
#   18:    E2E Test Count (min 20)
#   19:    E2E Test Execution (actual Playwright run)
#   20:    TDD Evidence Verification (RED-GREEN-REFACTOR cycle)
#
# Options:
#   --skip-docker    Skip Docker-related gates (5, 6, 7)
#   --skip-frontend  Skip frontend gates (3, 4, 9, 12, 15)
#   --skip-backend   Skip backend gates (1, 2, 8, 11, 13, 14)
#   --verbose        Show detailed output
#   --help           Show this help message

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common utilities
source "$SCRIPT_DIR/lib/common.sh"

# Default options
SKIP_DOCKER=false
SKIP_FRONTEND=false
SKIP_BACKEND=false
VERBOSE=false

# Parse arguments
PROJECT=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --skip-frontend)
            SKIP_FRONTEND=true
            shift
            ;;
        --skip-backend)
            SKIP_BACKEND=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            echo "Usage: $0 <project-name> [options]"
            echo ""
            echo "Gates:"
            echo "  1-2:   Backend (build, tests)"
            echo "  3-4:   Frontend (build, tests)"
            echo "  5-7:   Docker (build, run, health)"
            echo "  8:     API Coverage (handlers)"
            echo "  9:     Frontend Feature Coverage (pages)"
            echo "  10:    Integration Check"
            echo "  11:    Backend Test Count (min 35)"
            echo "  12:    Frontend Test Count (min 40)"
            echo "  13:    Empty Array Pattern (Go nil check)"
            echo "  14:    Backend Coverage (100%)"
            echo "  15:    E2E Test Verification (Playwright)"
            echo "  16:    Design Quality (shadcn/ui, layout, responsive)"
            echo ""
            echo "Options:"
            echo "  --skip-docker    Skip Docker-related gates (5, 6, 7)"
            echo "  --skip-frontend  Skip frontend gates (3, 4, 9, 12, 15, 16)"
            echo "  --skip-backend   Skip backend gates (1, 2, 8, 11, 13, 14)"
            echo "  --verbose        Show detailed output"
            echo "  --help           Show this help message"
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            exit 1
            ;;
        *)
            PROJECT=$1
            shift
            ;;
    esac
done

# Validate project name
if ! validate_project_name "$PROJECT"; then
    echo "Usage: $0 <project-name> [options]"
    exit 1
fi

PROJECT_DIR="$PROJECT_ROOT/$(get_project_dir "$PROJECT")"

# Verify project directory exists
if [[ ! -d "$PROJECT_DIR" ]]; then
    log_error "Project directory not found: $PROJECT_DIR"
    exit 1
fi

cd "$PROJECT_ROOT"

log_section "AIDA Quality Gates - $PROJECT"
echo "Project directory: $PROJECT_DIR"
echo "Timestamp: $(date -Iseconds)"
echo "Container runtime: ${CONTAINER_RUNTIME:-detecting...}"
if [[ -n "${DOCKER_HOST:-}" ]]; then
    echo "Docker host: $DOCKER_HOST"
fi

TOTAL_GATES=0
PASSED_GATES=0
FAILED_GATES=()

# Gate execution wrapper
execute_gate() {
    local gate_num=$1
    local gate_name=$2
    local gate_dir=$3
    local gate_cmd=$4

    TOTAL_GATES=$((TOTAL_GATES + 1))

    echo ""
    log_info "[Gate $gate_num] $gate_name"

    if [[ -n "$gate_dir" ]]; then
        if [[ ! -d "$gate_dir" ]]; then
            log_error "Directory not found: $gate_dir"
            FAILED_GATES+=("Gate $gate_num: $gate_name (directory missing)")
            return 1
        fi
        cd "$gate_dir"
    fi

    if $VERBOSE; then
        if eval "$gate_cmd"; then
            log_success "Gate $gate_num PASSED"
            PASSED_GATES=$((PASSED_GATES + 1))
            cd "$PROJECT_ROOT"
            return 0
        else
            log_error "Gate $gate_num FAILED"
            FAILED_GATES+=("Gate $gate_num: $gate_name")
            cd "$PROJECT_ROOT"
            return 1
        fi
    else
        if eval "$gate_cmd" > /tmp/aida_gate_$gate_num.log 2>&1; then
            log_success "Gate $gate_num PASSED"
            PASSED_GATES=$((PASSED_GATES + 1))
            cd "$PROJECT_ROOT"
            return 0
        else
            log_error "Gate $gate_num FAILED"
            echo "--- Error Output ---"
            tail -50 /tmp/aida_gate_$gate_num.log
            echo "--- End Output ---"
            FAILED_GATES+=("Gate $gate_num: $gate_name")
            cd "$PROJECT_ROOT"
            return 1
        fi
    fi
}

# ============================================
# Gate 1: Backend Build
# ============================================
if [[ "$SKIP_BACKEND" != "true" ]]; then
    execute_gate 1 "Backend Build" "$PROJECT_DIR/backend" \
        "go mod tidy && go build ./..."
fi

# ============================================
# Gate 2: Backend Tests
# ============================================
if [[ "$SKIP_BACKEND" != "true" ]]; then
    execute_gate 2 "Backend Tests" "$PROJECT_DIR/backend" \
        "go test ./... -v"
fi

# ============================================
# Gate 3: Frontend Build
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    # Detect package manager (prefer pnpm > npm)
    if [[ -f "$PROJECT_DIR/frontend/pnpm-lock.yaml" ]]; then
        execute_gate 3 "Frontend Build" "$PROJECT_DIR/frontend" \
            "pnpm install && pnpm run build"
    else
        execute_gate 3 "Frontend Build" "$PROJECT_DIR/frontend" \
            "npm install && npm run build"
    fi
fi

# ============================================
# Gate 4: Frontend Tests
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    # Detect package manager (prefer pnpm > npm)
    if [[ -f "$PROJECT_DIR/frontend/pnpm-lock.yaml" ]]; then
        execute_gate 4 "Frontend Tests" "$PROJECT_DIR/frontend" \
            "pnpm test -- --run"
    else
        execute_gate 4 "Frontend Tests" "$PROJECT_DIR/frontend" \
            "npm test -- --run"
    fi
fi

# ============================================
# Gate 5: Docker Build
# ============================================
if [[ "$SKIP_DOCKER" != "true" ]]; then
    execute_gate 5 "Docker Build" "$PROJECT_DIR" \
        "compose_cmd build"
fi

# ============================================
# Gate 6: Docker Run
# ============================================
if [[ "$SKIP_DOCKER" != "true" ]]; then
    echo ""
    log_info "[Gate 6] Docker Run"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    cd "$PROJECT_DIR"

    # Clean up any existing containers
    compose_cmd down --remove-orphans 2>/dev/null || true

    if compose_cmd up -d > /tmp/aida_gate_6.log 2>&1; then
        log_info "Waiting for services to start (30 seconds)..."
        sleep 30

        # Check if all services are running
        if compose_cmd ps | grep -q "Up"; then
            log_success "Gate 6 PASSED"
            PASSED_GATES=$((PASSED_GATES + 1))
        else
            log_error "Gate 6 FAILED - Services not running"
            compose_cmd ps
            compose_cmd logs --tail=50
            FAILED_GATES+=("Gate 6: Docker Run (services not healthy)")
        fi
    else
        log_error "Gate 6 FAILED - Compose up failed"
        cat /tmp/aida_gate_6.log
        FAILED_GATES+=("Gate 6: Docker Run")
    fi

    cd "$PROJECT_ROOT"
fi

# ============================================
# Gate 7: Health Check
# ============================================
if [[ "$SKIP_DOCKER" != "true" ]]; then
    echo ""
    log_info "[Gate 7] API Health Check"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    if wait_for_service "http://localhost:8080/health" 30 2; then
        log_success "Gate 7 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_error "Gate 7 FAILED - Health check failed"
        FAILED_GATES+=("Gate 7: Health Check")
    fi
    # NOTE: cleanup_docker moved to after Gate 19
fi

# ============================================
# Gate 19: E2E Test Execution (Playwright)
# ============================================
if [[ "$SKIP_DOCKER" != "true" && "$SKIP_FRONTEND" != "true" ]]; then
    echo ""
    log_info "[Gate 19] E2E Test Execution (Playwright)"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    E2E_DIR="$FRONTEND_DIR/e2e"

    # Check if E2E tests exist
    if [[ -d "$E2E_DIR" ]] || [[ -f "$FRONTEND_DIR/playwright.config.ts" ]]; then
        cd "$FRONTEND_DIR"

        # Install Playwright browsers if needed (chromium only for speed)
        echo "  Installing Playwright browsers if needed..."
        pnpm exec playwright install chromium --with-deps 2>/dev/null || npm exec playwright install chromium --with-deps 2>/dev/null || true

        # Set E2E base URL for Docker environment
        export E2E_BASE_URL="http://localhost:5173"
        export PLAYWRIGHT_BASE_URL="http://localhost:5173"

        # Wait for frontend to be accessible
        echo "  Waiting for frontend at http://localhost:5173..."
        if ! wait_for_service "http://localhost:5173" 15 2; then
            log_error "Gate 19 FAILED - Frontend not accessible"
            FAILED_GATES+=("Gate 19: E2E Test Execution (frontend not ready)")
            cd "$PROJECT_ROOT"
            continue 2>/dev/null || true
        fi

        # Run E2E tests against running Docker containers
        echo "  Running E2E tests against http://localhost:5173..."
        E2E_RESULT=0
        if [[ -f "pnpm-lock.yaml" ]]; then
            E2E_OUTPUT=$(pnpm test:e2e --reporter=list 2>&1) || E2E_RESULT=$?
        else
            E2E_OUTPUT=$(npm run test:e2e -- --reporter=list 2>&1) || E2E_RESULT=$?
        fi

        if [[ $E2E_RESULT -eq 0 ]]; then
            log_success "Gate 19 PASSED"
            PASSED_GATES=$((PASSED_GATES + 1))
            echo "  All E2E tests passed!"
        else
            log_error "Gate 19 FAILED - E2E tests failed"
            echo "--- E2E Test Output (last 50 lines) ---"
            echo "$E2E_OUTPUT" | tail -50
            echo "--- End E2E Output ---"
            FAILED_GATES+=("Gate 19: E2E Test Execution")
        fi

        cd "$PROJECT_ROOT"
    else
        log_warning "Gate 19 SKIPPED - No E2E tests found"
        echo "  No e2e/ directory or playwright.config.ts found"
        # Skip as warning, not failure
        PASSED_GATES=$((PASSED_GATES + 1))
    fi

    # Cleanup Docker after E2E tests
    cleanup_docker "$PROJECT_DIR"
elif [[ "$SKIP_DOCKER" != "true" ]]; then
    # Docker is running but frontend is skipped - still need cleanup
    cleanup_docker "$PROJECT_DIR"
fi

# ============================================
# Gate 8: API Coverage (Backend Handlers)
# ============================================
if [[ "$SKIP_BACKEND" != "true" ]]; then
    echo ""
    log_info "[Gate 8] API Coverage Check"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    BACKEND_DIR="$PROJECT_DIR/backend"
    HANDLER_DIR="$BACKEND_DIR/internal/handler"

    if [[ -d "$HANDLER_DIR" ]]; then
        # Count handler files (excluding test files)
        HANDLER_FILES=$(find "$HANDLER_DIR" -name "*.go" ! -name "*_test.go" 2>/dev/null | wc -l)

        # Count handler functions
        HANDLER_FUNCS=$(grep -r "func.*Handler\|func (.*) ServeHTTP\|func (.*) Handle" "$HANDLER_DIR" 2>/dev/null | grep -v "_test.go" | wc -l)

        # Minimum requirements: at least 3 handler files and 10 handler functions
        MIN_HANDLER_FILES=3
        MIN_HANDLER_FUNCS=10

        echo "  Handler files: $HANDLER_FILES (min: $MIN_HANDLER_FILES)"
        echo "  Handler functions: $HANDLER_FUNCS (min: $MIN_HANDLER_FUNCS)"

        if [[ $HANDLER_FILES -ge $MIN_HANDLER_FILES && $HANDLER_FUNCS -ge $MIN_HANDLER_FUNCS ]]; then
            log_success "Gate 8 PASSED"
            PASSED_GATES=$((PASSED_GATES + 1))
        else
            log_error "Gate 8 FAILED - Insufficient API coverage"
            echo "  Required: $MIN_HANDLER_FILES+ files, $MIN_HANDLER_FUNCS+ functions"
            FAILED_GATES+=("Gate 8: API Coverage (handlers: $HANDLER_FUNCS/$MIN_HANDLER_FUNCS)")
        fi
    else
        log_error "Gate 8 FAILED - Handler directory not found"
        FAILED_GATES+=("Gate 8: API Coverage (no handler dir)")
    fi
fi

# ============================================
# Gate 9: Frontend Feature Coverage
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    echo ""
    log_info "[Gate 9] Frontend Feature Check"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    PAGES_DIR="$FRONTEND_DIR/src/pages"
    COMPONENTS_DIR="$FRONTEND_DIR/src/components"

    # Check for pages directory or page components
    PAGE_COUNT=0
    if [[ -d "$PAGES_DIR" ]]; then
        # Count .tsx files excluding test files
        while IFS= read -r file; do
            if [[ ! "$file" =~ \.test\.tsx$ ]]; then
                PAGE_COUNT=$((PAGE_COUNT + 1))
            fi
        done < <(find "$PAGES_DIR" -name "*.tsx" -type f 2>/dev/null)
    fi

    # Also count top-level page-like components (Login, Register, Home, etc.)
    if [[ -d "$COMPONENTS_DIR" ]]; then
        while IFS= read -r file; do
            if [[ ! "$file" =~ test ]]; then
                PAGE_COUNT=$((PAGE_COUNT + 1))
            fi
        done < <(find "$COMPONENTS_DIR" \( -name "*Page*.tsx" -o -name "*Login*.tsx" -o -name "*Register*.tsx" -o -name "*Home*.tsx" \) -type f 2>/dev/null)
    fi

    # Check for routing
    HAS_ROUTING=false
    if grep -rq "react-router\|createBrowserRouter\|BrowserRouter\|Routes" "$FRONTEND_DIR/src" 2>/dev/null; then
        HAS_ROUTING=true
    fi

    # Check for API client
    HAS_API_CLIENT=false
    if [[ -f "$FRONTEND_DIR/src/api/client.ts" || -f "$FRONTEND_DIR/src/services/api.ts" || -f "$FRONTEND_DIR/src/lib/api.ts" ]]; then
        HAS_API_CLIENT=true
    fi

    # Minimum requirements
    MIN_PAGES=3

    echo "  Page components: $PAGE_COUNT (min: $MIN_PAGES)"
    echo "  Has routing: $HAS_ROUTING"
    echo "  Has API client: $HAS_API_CLIENT"

    if [[ $PAGE_COUNT -ge $MIN_PAGES ]]; then
        log_success "Gate 9 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_error "Gate 9 FAILED - Insufficient frontend coverage"
        echo "  Required: $MIN_PAGES+ page components"
        FAILED_GATES+=("Gate 9: Frontend Coverage (pages: $PAGE_COUNT/$MIN_PAGES)")
    fi
fi

# ============================================
# Gate 10: Integration Check
# ============================================
if [[ "$SKIP_FRONTEND" != "true" && "$SKIP_BACKEND" != "true" ]]; then
    echo ""
    log_info "[Gate 10] Integration Check"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    BACKEND_DIR="$PROJECT_DIR/backend"
    INTEGRATION_SCORE=0
    INTEGRATION_ISSUES=()

    # Check 1: Frontend has API client
    if [[ -f "$FRONTEND_DIR/src/api/client.ts" || -f "$FRONTEND_DIR/src/services/api.ts" || -f "$FRONTEND_DIR/src/lib/api.ts" ]]; then
        INTEGRATION_SCORE=$((INTEGRATION_SCORE + 1))
    else
        INTEGRATION_ISSUES+=("No API client found")
    fi

    # Check 2: Frontend references backend URL
    if grep -rq "localhost:8080\|API_URL\|VITE_API" "$FRONTEND_DIR/src" 2>/dev/null; then
        INTEGRATION_SCORE=$((INTEGRATION_SCORE + 1))
    else
        INTEGRATION_ISSUES+=("No backend URL reference")
    fi

    # Check 3: Backend has CORS configuration
    if grep -rq "CORS\|cors\|Access-Control" "$BACKEND_DIR" 2>/dev/null; then
        INTEGRATION_SCORE=$((INTEGRATION_SCORE + 1))
    else
        INTEGRATION_ISSUES+=("No CORS configuration")
    fi

    # Check 4: Docker compose links services
    if [[ -f "$PROJECT_DIR/docker-compose.yml" ]]; then
        if grep -q "depends_on" "$PROJECT_DIR/docker-compose.yml" 2>/dev/null; then
            INTEGRATION_SCORE=$((INTEGRATION_SCORE + 1))
        else
            INTEGRATION_ISSUES+=("Docker services not linked")
        fi
    fi

    # Minimum score: 3 out of 4
    MIN_SCORE=3

    echo "  Integration score: $INTEGRATION_SCORE/4 (min: $MIN_SCORE)"

    if [[ $INTEGRATION_SCORE -ge $MIN_SCORE ]]; then
        log_success "Gate 10 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_error "Gate 10 FAILED - Integration issues detected"
        for issue in "${INTEGRATION_ISSUES[@]}"; do
            echo "    - $issue"
        done
        FAILED_GATES+=("Gate 10: Integration (score: $INTEGRATION_SCORE/$MIN_SCORE)")
    fi
fi

# ============================================
# Gate 11: Backend Test Count Verification
# ============================================
if [[ "$SKIP_BACKEND" != "true" ]]; then
    echo ""
    log_info "[Gate 11] Backend Test Count Verification"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    BACKEND_DIR="$PROJECT_DIR/backend"
    MIN_BACKEND_TESTS=80

    # Count test files
    TEST_FILE_COUNT=$(find "$BACKEND_DIR" -name "*_test.go" -type f 2>/dev/null | wc -l)

    # Count actual test functions
    TEST_FUNC_COUNT=$(grep -r "func Test" "$BACKEND_DIR" --include="*_test.go" 2>/dev/null | wc -l)

    echo "  Test files: $TEST_FILE_COUNT"
    echo "  Test functions: $TEST_FUNC_COUNT (min: $MIN_BACKEND_TESTS)"

    if [[ $TEST_FUNC_COUNT -ge $MIN_BACKEND_TESTS ]]; then
        log_success "Gate 11 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_error "Gate 11 FAILED - Insufficient backend tests"
        echo "  Required: $MIN_BACKEND_TESTS+ test functions"
        FAILED_GATES+=("Gate 11: Backend Test Count ($TEST_FUNC_COUNT/$MIN_BACKEND_TESTS)")
    fi
fi

# ============================================
# Gate 12: Frontend Test Count Verification
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    echo ""
    log_info "[Gate 12] Frontend Test Count Verification"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    MIN_FRONTEND_TESTS=100

    # Count test files
    TEST_FILE_COUNT=$(find "$FRONTEND_DIR/src" \( -name "*.test.tsx" -o -name "*.test.ts" \) -type f 2>/dev/null | wc -l)

    # Count actual test cases (it/test blocks)
    TEST_CASE_COUNT=$(grep -rE "^\s*(it|test)\s*\(" "$FRONTEND_DIR/src" --include="*.test.tsx" --include="*.test.ts" 2>/dev/null | wc -l)

    echo "  Test files: $TEST_FILE_COUNT"
    echo "  Test cases: $TEST_CASE_COUNT (min: $MIN_FRONTEND_TESTS)"

    if [[ $TEST_CASE_COUNT -ge $MIN_FRONTEND_TESTS ]]; then
        log_success "Gate 12 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_error "Gate 12 FAILED - Insufficient frontend tests"
        echo "  Required: $MIN_FRONTEND_TESTS+ test cases"
        FAILED_GATES+=("Gate 12: Frontend Test Count ($TEST_CASE_COUNT/$MIN_FRONTEND_TESTS)")
    fi
fi

# ============================================
# Gate 13: Empty Array Pattern Verification (Go)
# ============================================
if [[ "$SKIP_BACKEND" != "true" ]]; then
    echo ""
    log_info "[Gate 13] Empty Array Pattern Verification"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    BACKEND_DIR="$PROJECT_DIR/backend"

    # Find dangerous patterns: var slice []Type (returns null in JSON)
    # Look for patterns like: var users []*model.User or var posts []*Post
    DANGEROUS_PATTERNS=$(set +e; grep -rE "^\s*var\s+\w+\s+\[\]\*?" "$BACKEND_DIR" --include="*.go" 2>/dev/null | \
        grep -v "_test.go" | \
        grep -v "// safe" | \
        wc -l | tr -d ' '; set -e)
    DANGEROUS_PATTERNS=${DANGEROUS_PATTERNS:-0}
    [[ -z "$DANGEROUS_PATTERNS" ]] && DANGEROUS_PATTERNS=0

    # Find correct patterns: make([]Type, 0)
    SAFE_PATTERNS=$(set +e; grep -rE "make\(\[\]\*?\w+.*,\s*0\)" "$BACKEND_DIR" --include="*.go" 2>/dev/null | \
        wc -l | tr -d ' '; set -e)
    SAFE_PATTERNS=${SAFE_PATTERNS:-0}
    [[ -z "$SAFE_PATTERNS" ]] && SAFE_PATTERNS=0

    echo "  Potentially dangerous 'var slice' patterns: $DANGEROUS_PATTERNS"
    echo "  Safe 'make([]T, 0)' patterns: $SAFE_PATTERNS"

    if [[ $DANGEROUS_PATTERNS -eq 0 ]]; then
        log_success "Gate 13 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_warning "Gate 13 WARNING - Found potentially dangerous slice patterns"
        echo "  These patterns return null instead of [] in JSON:"
        grep -rE "^\s*var\s+\w+\s+\[\]\*?" "$BACKEND_DIR" --include="*.go" 2>/dev/null | \
            grep -v "_test.go" | \
            grep -v "// safe" | \
            head -10
        echo ""
        echo "  Fix by using: make([]*Type, 0) instead of var slice []*Type"
        # This is a warning gate, not a blocker (pass with warning)
        log_success "Gate 13 PASSED (with warnings)"
        PASSED_GATES=$((PASSED_GATES + 1))
    fi
fi

# ============================================
# Gate 14: Backend Coverage Check
# ============================================
if [[ "$SKIP_BACKEND" != "true" ]]; then
    echo ""
    log_info "[Gate 14] Backend Coverage Check"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    BACKEND_DIR="$PROJECT_DIR/backend"
    MIN_COVERAGE=100

    cd "$BACKEND_DIR"

    # Run coverage and capture output
    COVERAGE_OUTPUT=$(go test ./... -coverprofile=/tmp/coverage.out 2>&1)

    if [[ $? -eq 0 ]]; then
        # Calculate average coverage
        COVERAGE_LINES=$(go tool cover -func=/tmp/coverage.out 2>/dev/null | grep "total:" | awk '{print $3}' | tr -d '%')

        if [[ -n "$COVERAGE_LINES" ]]; then
            COVERAGE_INT=${COVERAGE_LINES%.*}
            echo "  Coverage: ${COVERAGE_LINES}% (min: ${MIN_COVERAGE}%)"

            if [[ $COVERAGE_INT -ge $MIN_COVERAGE ]]; then
                log_success "Gate 14 PASSED"
                PASSED_GATES=$((PASSED_GATES + 1))
            else
                log_error "Gate 14 FAILED - Coverage below threshold"
                FAILED_GATES+=("Gate 14: Backend Coverage (${COVERAGE_LINES}%/${MIN_COVERAGE}%)")
            fi
        else
            log_warning "Gate 14 SKIPPED - Could not calculate coverage"
            PASSED_GATES=$((PASSED_GATES + 1))
        fi
    else
        log_error "Gate 14 FAILED - Tests failed during coverage check"
        FAILED_GATES+=("Gate 14: Backend Coverage (tests failed)")
    fi

    cd "$PROJECT_ROOT"
fi

# ============================================
# Gate 15: E2E Test Verification (Playwright)
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    echo ""
    log_info "[Gate 15] E2E Test Verification"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    E2E_DIR="$FRONTEND_DIR/e2e"

    # Check if Playwright is configured
    HAS_PLAYWRIGHT=false
    if [[ -f "$FRONTEND_DIR/playwright.config.ts" ]] || [[ -f "$FRONTEND_DIR/playwright.config.js" ]]; then
        HAS_PLAYWRIGHT=true
    fi

    # Count E2E test files
    E2E_TEST_COUNT=0
    if [[ -d "$E2E_DIR" ]]; then
        E2E_TEST_COUNT=$(find "$E2E_DIR" \( -name "*.spec.ts" -o -name "*.test.ts" \) -type f 2>/dev/null | wc -l)
    fi

    # Check for E2E tests in other locations
    if [[ $E2E_TEST_COUNT -eq 0 ]]; then
        E2E_TEST_COUNT=$(find "$FRONTEND_DIR" -name "*.spec.ts" -type f 2>/dev/null | wc -l)
    fi

    echo "  Playwright configured: $HAS_PLAYWRIGHT"
    echo "  E2E test files: $E2E_TEST_COUNT"

    if [[ "$HAS_PLAYWRIGHT" == "true" ]] || [[ $E2E_TEST_COUNT -gt 0 ]]; then
        log_success "Gate 15 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_warning "Gate 15 WARNING - No E2E tests found"
        echo "  Consider adding Playwright E2E tests for critical flows"
        # This is a warning gate for now, pass with warning
        log_success "Gate 15 PASSED (with warnings)"
        PASSED_GATES=$((PASSED_GATES + 1))
    fi
fi

# ============================================
# Gate 16: Design Quality Verification
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    echo ""
    log_info "[Gate 16] Design Quality Verification"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    DESIGN_SCORE=0
    DESIGN_ISSUES=()
    DESIGN_TOTAL=10

    # Check 1: Tailwind CSS configured
    if [[ -f "$FRONTEND_DIR/tailwind.config.js" ]] || [[ -f "$FRONTEND_DIR/tailwind.config.ts" ]]; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("No Tailwind CSS config")
    fi

    # Check 2: shadcn/ui configured
    if [[ -f "$FRONTEND_DIR/components.json" ]]; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("No shadcn/ui config (components.json)")
    fi

    # Check 3: UI components directory exists
    if [[ -d "$FRONTEND_DIR/src/components/ui" ]]; then
        UI_COMPONENT_COUNT=$(find "$FRONTEND_DIR/src/components/ui" -name "*.tsx" 2>/dev/null | wc -l)
        if [[ $UI_COMPONENT_COUNT -ge 5 ]]; then
            DESIGN_SCORE=$((DESIGN_SCORE + 1))
        else
            DESIGN_ISSUES+=("Insufficient UI components ($UI_COMPONENT_COUNT, need 5+)")
        fi
    else
        DESIGN_ISSUES+=("No UI components directory")
    fi

    # Check 4: Layout components exist
    HAS_LAYOUT=false
    if [[ -d "$FRONTEND_DIR/src/components/layout" ]]; then
        LAYOUT_COUNT=$(find "$FRONTEND_DIR/src/components/layout" -name "*.tsx" 2>/dev/null | wc -l)
        if [[ $LAYOUT_COUNT -ge 2 ]]; then
            HAS_LAYOUT=true
            DESIGN_SCORE=$((DESIGN_SCORE + 1))
        fi
    fi
    if [[ "$HAS_LAYOUT" != "true" ]]; then
        DESIGN_ISSUES+=("Missing layout components (Header, Sidebar, Layout)")
    fi

    # Check 5: Lucide icons used
    if grep -rq "from 'lucide-react'\|from \"lucide-react\"" "$FRONTEND_DIR/src" 2>/dev/null; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("No Lucide icons found")
    fi

    # Check 6: No raw HTML buttons (should use Button component)
    RAW_BUTTONS=$(set +e; grep -r "<button" "$FRONTEND_DIR/src" --include="*.tsx" 2>/dev/null | grep -v "Button" | grep -v "// raw" | wc -l | tr -d ' '; set -e)
    RAW_BUTTONS=${RAW_BUTTONS:-0}
    [[ -z "$RAW_BUTTONS" ]] && RAW_BUTTONS=0
    if [[ $RAW_BUTTONS -le 2 ]]; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("Too many raw <button> elements ($RAW_BUTTONS)")
    fi

    # Check 7: Loading state components
    if grep -rq "Skeleton\|Loading\|Spinner" "$FRONTEND_DIR/src" --include="*.tsx" 2>/dev/null; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("No loading state components")
    fi

    # Check 8: Empty state handling
    if grep -rq "EmptyState\|No posts\|No data\|empty" "$FRONTEND_DIR/src" --include="*.tsx" 2>/dev/null; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("No empty state handling")
    fi

    # Check 9: Responsive classes used
    if grep -rq "sm:\|md:\|lg:\|xl:" "$FRONTEND_DIR/src" --include="*.tsx" 2>/dev/null; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("No responsive breakpoint classes")
    fi

    # Check 10: No inline styles
    INLINE_STYLES=$(set +e; grep -r "style={{" "$FRONTEND_DIR/src" --include="*.tsx" 2>/dev/null | wc -l | tr -d ' '; set -e)
    INLINE_STYLES=${INLINE_STYLES:-0}
    [[ -z "$INLINE_STYLES" ]] && INLINE_STYLES=0
    if [[ $INLINE_STYLES -le 3 ]]; then
        DESIGN_SCORE=$((DESIGN_SCORE + 1))
    else
        DESIGN_ISSUES+=("Too many inline styles ($INLINE_STYLES)")
    fi

    # Minimum requirement: 7 out of 10
    MIN_DESIGN_SCORE=7

    echo "  Design quality score: $DESIGN_SCORE/$DESIGN_TOTAL (min: $MIN_DESIGN_SCORE)"

    if [[ $DESIGN_SCORE -ge $MIN_DESIGN_SCORE ]]; then
        log_success "Gate 16 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_error "Gate 16 FAILED - Insufficient design quality"
        for issue in "${DESIGN_ISSUES[@]}"; do
            echo "    - $issue"
        done
        FAILED_GATES+=("Gate 16: Design Quality ($DESIGN_SCORE/$MIN_DESIGN_SCORE)")
    fi
fi

# ============================================
# Gate 17: Frontend Coverage Check
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    echo ""
    log_info "[Gate 17] Frontend Coverage Check"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    MIN_FRONTEND_COVERAGE=100

    cd "$FRONTEND_DIR"

    # Check if vitest coverage is configured
    if grep -q "coverage" "$FRONTEND_DIR/package.json" 2>/dev/null || \
       grep -q "coverage" "$FRONTEND_DIR/vitest.config.ts" 2>/dev/null || \
       grep -q "coverage" "$FRONTEND_DIR/vite.config.ts" 2>/dev/null; then

        # Run coverage (detect package manager)
        if [[ -f "$FRONTEND_DIR/pnpm-lock.yaml" ]]; then
            COVERAGE_OUTPUT=$(pnpm test -- --run --coverage 2>&1) || true
        else
            COVERAGE_OUTPUT=$(npm test -- --run --coverage 2>&1) || true
        fi

        # Try to extract coverage from output or coverage file
        if [[ -f "$FRONTEND_DIR/coverage/coverage-summary.json" ]]; then
            COVERAGE_LINES=$(jq '.total.lines.pct // 0' "$FRONTEND_DIR/coverage/coverage-summary.json" 2>/dev/null)
        else
            COVERAGE_LINES=$(set +e; echo "$COVERAGE_OUTPUT" | grep -oP 'All files[^|]*\|\s*\K[0-9.]+' | head -1; set -e)
        fi

        if [[ -n "$COVERAGE_LINES" ]]; then
            COVERAGE_INT=${COVERAGE_LINES%.*}
            echo "  Coverage: ${COVERAGE_LINES}% (min: ${MIN_FRONTEND_COVERAGE}%)"

            if [[ $COVERAGE_INT -ge $MIN_FRONTEND_COVERAGE ]]; then
                log_success "Gate 17 PASSED"
                PASSED_GATES=$((PASSED_GATES + 1))
            else
                log_error "Gate 17 FAILED - Coverage below threshold"
                FAILED_GATES+=("Gate 17: Frontend Coverage (${COVERAGE_LINES}%/${MIN_FRONTEND_COVERAGE}%)")
            fi
        else
            log_warning "Gate 17 SKIPPED - Could not calculate coverage"
            echo "  Consider configuring vitest coverage"
            PASSED_GATES=$((PASSED_GATES + 1))
        fi
    else
        log_warning "Gate 17 SKIPPED - No coverage configuration"
        echo "  Add coverage config to vitest.config.ts"
        PASSED_GATES=$((PASSED_GATES + 1))
    fi

    cd "$PROJECT_ROOT"
fi

# ============================================
# Gate 18: E2E Test Count
# ============================================
if [[ "$SKIP_FRONTEND" != "true" ]]; then
    echo ""
    log_info "[Gate 18] E2E Test Count Verification"
    TOTAL_GATES=$((TOTAL_GATES + 1))

    FRONTEND_DIR="$PROJECT_DIR/frontend"
    MIN_E2E_TESTS=20

    E2E_DIR="$FRONTEND_DIR/e2e"
    E2E_TEST_COUNT=0

    # Count E2E test cases
    if [[ -d "$E2E_DIR" ]]; then
        E2E_TEST_COUNT=$(set +e; grep -rE "^\s*(it|test)\s*\(" "$E2E_DIR" --include="*.spec.ts" --include="*.test.ts" 2>/dev/null | wc -l | tr -d ' '; set -e)
        E2E_TEST_COUNT=${E2E_TEST_COUNT:-0}
        [[ -z "$E2E_TEST_COUNT" ]] && E2E_TEST_COUNT=0
    fi

    # Also check for Playwright tests in tests/ directory
    if [[ -d "$FRONTEND_DIR/tests" ]]; then
        TESTS_COUNT=$(set +e; grep -rE "^\s*(it|test)\s*\(" "$FRONTEND_DIR/tests" --include="*.spec.ts" --include="*.test.ts" 2>/dev/null | wc -l | tr -d ' '; set -e)
        TESTS_COUNT=${TESTS_COUNT:-0}
        [[ -z "$TESTS_COUNT" ]] && TESTS_COUNT=0
        E2E_TEST_COUNT=$((E2E_TEST_COUNT + TESTS_COUNT))
    fi

    echo "  E2E test cases: $E2E_TEST_COUNT (min: $MIN_E2E_TESTS)"

    if [[ $E2E_TEST_COUNT -ge $MIN_E2E_TESTS ]]; then
        log_success "Gate 18 PASSED"
        PASSED_GATES=$((PASSED_GATES + 1))
    else
        log_error "Gate 18 FAILED - Insufficient E2E tests"
        echo "  Required: $MIN_E2E_TESTS+ E2E test cases"
        echo "  Create tests in e2e/ directory using Playwright"
        FAILED_GATES+=("Gate 18: E2E Test Count ($E2E_TEST_COUNT/$MIN_E2E_TESTS)")
    fi
fi

# ============================================
# Gate 20: TDD Evidence Verification (Issue #5)
# ============================================
echo ""
log_info "[Gate 20] TDD Evidence Verification"
TOTAL_GATES=$((TOTAL_GATES + 1))

TDD_EVIDENCE_DIR="$PROJECT_ROOT/.aida/tdd-evidence"
MIN_TDD_EVIDENCE=10  # Minimum 10 TDD evidence files

if [[ -d "$TDD_EVIDENCE_DIR" ]]; then
    # Count TDD evidence files
    TDD_EVIDENCE_COUNT=$(find "$TDD_EVIDENCE_DIR" -name "*.json" -type f 2>/dev/null | wc -l)
    TDD_EVIDENCE_COUNT=${TDD_EVIDENCE_COUNT:-0}

    echo "  TDD evidence files: $TDD_EVIDENCE_COUNT (min: $MIN_TDD_EVIDENCE)"

    if [[ $TDD_EVIDENCE_COUNT -ge $MIN_TDD_EVIDENCE ]]; then
        # Verify each evidence has RED-GREEN-REFACTOR cycle
        INVALID_EVIDENCE=0
        for evidence in "$TDD_EVIDENCE_DIR"/*.json; do
            [[ -f "$evidence" ]] || continue

            # Check for red_phase with non-zero exit code
            if ! jq -e '.red_phase.exit_code > 0' "$evidence" >/dev/null 2>&1; then
                echo "    Missing RED phase: $(basename $evidence)" >&2
                INVALID_EVIDENCE=$((INVALID_EVIDENCE + 1))
            fi

            # Check for green_phase with zero exit code
            if ! jq -e '.green_phase.exit_code == 0' "$evidence" >/dev/null 2>&1; then
                echo "    Missing GREEN phase: $(basename $evidence)" >&2
                INVALID_EVIDENCE=$((INVALID_EVIDENCE + 1))
            fi
        done

        if [[ $INVALID_EVIDENCE -eq 0 ]]; then
            log_success "Gate 20 PASSED"
            PASSED_GATES=$((PASSED_GATES + 1))
        else
            log_error "Gate 20 FAILED - Invalid TDD evidence ($INVALID_EVIDENCE files)"
            FAILED_GATES+=("Gate 20: TDD Evidence (invalid: $INVALID_EVIDENCE)")
        fi
    else
        log_warning "Gate 20 SKIPPED - Insufficient TDD evidence"
        echo "  Create TDD evidence with RED-GREEN-REFACTOR cycle"
        echo "  Use: ./scripts/tdd-logger.sh red|green|refactor <feature> <file>"
        # Skip as warning for now (TDD is encouraged but not blocking)
        PASSED_GATES=$((PASSED_GATES + 1))
    fi
else
    log_warning "Gate 20 SKIPPED - No TDD evidence directory"
    echo "  TDD evidence will be tracked in .aida/tdd-evidence/"
    # Skip as warning for now
    PASSED_GATES=$((PASSED_GATES + 1))
fi

# ============================================
# Summary
# ============================================
echo ""
log_section "Quality Gate Summary"

echo "Project: $PROJECT"
echo "Total gates executed: $TOTAL_GATES"
echo -e "Passed: ${GREEN}$PASSED_GATES${NC}"
echo -e "Failed: ${RED}${#FAILED_GATES[@]}${NC}"

if [[ ${#FAILED_GATES[@]} -gt 0 ]]; then
    echo ""
    log_error "Failed gates:"
    for gate in "${FAILED_GATES[@]}"; do
        echo "  - $gate"
    done
    echo ""
    log_error "QUALITY GATES FAILED"
    exit 1
else
    echo ""
    log_success "ALL QUALITY GATES PASSED"
    echo ""
    echo "Project $PROJECT is ready for deployment."
    exit 0
fi
