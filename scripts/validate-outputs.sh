#!/bin/bash
# AIDA Output Validation
# Validates that required outputs were actually created
#
# Usage: ./scripts/validate-outputs.sh <project-name> <phase>
#
# Phases:
#   spec - Validate specification outputs
#   impl - Validate implementation outputs
#   all  - Validate both spec and impl

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Source common utilities
source "$SCRIPT_DIR/lib/common.sh"

# Parse arguments
PROJECT=${1:-}
PHASE=${2:-all}

if [[ -z "$PROJECT" ]]; then
    echo "Usage: $0 <project-name> [phase]"
    echo ""
    echo "Phases:"
    echo "  spec - Validate specification outputs"
    echo "  impl - Validate implementation outputs"
    echo "  all  - Validate both (default)"
    exit 1
fi

# Validate project name
if ! validate_project_name "$PROJECT"; then
    exit 1
fi

cd "$PROJECT_ROOT"

log_section "AIDA Output Validation - $PROJECT"
echo "Phase: $PHASE"
echo "Timestamp: $(date -Iseconds)"

ERRORS=0

# ============================================
# Spec Phase Validation
# ============================================
validate_spec() {
    log_section "Spec Phase Validation"

    local spec_dir=".aida/specs"
    local min_spec_size=500  # Minimum bytes for spec files

    # Check requirements.md
    if ! check_file_exists "$spec_dir/${PROJECT}-requirements.md" $min_spec_size; then
        ERRORS=$((ERRORS + 1))
    fi

    # Check design.md
    if ! check_file_exists "$spec_dir/${PROJECT}-design.md" $min_spec_size; then
        ERRORS=$((ERRORS + 1))
    fi

    # Check tasks.md
    if ! check_file_exists "$spec_dir/${PROJECT}-tasks.md" 100; then
        ERRORS=$((ERRORS + 1))
    fi

    # Check spec-complete.json
    local result_file=".aida/results/spec-complete.json"
    if [[ -f "$result_file" ]]; then
        log_success "Spec completion report exists: $result_file"

        # Validate JSON structure
        if command -v jq &> /dev/null; then
            if jq -e '.status == "completed"' "$result_file" > /dev/null 2>&1; then
                log_success "Spec status: completed"
            else
                log_error "Spec status is not 'completed'"
                ERRORS=$((ERRORS + 1))
            fi
        fi
    else
        log_warning "Spec completion report not found: $result_file"
    fi
}

# ============================================
# Implementation Phase Validation
# ============================================
validate_impl() {
    log_section "Implementation Phase Validation"

    local project_dir="$PROJECT"

    # Check project directory exists
    if ! check_dir_exists "$project_dir"; then
        ERRORS=$((ERRORS + 1))
        return
    fi

    # ----------------------------------------
    # Backend Validation
    # ----------------------------------------
    log_info "Checking backend..."

    local backend_dir="$project_dir/backend"
    if ! check_dir_exists "$backend_dir"; then
        ERRORS=$((ERRORS + 1))
    else
        # Check essential backend files
        local backend_files=(
            "go.mod"
            "cmd/server/main.go"
        )

        for file in "${backend_files[@]}"; do
            if ! check_file_exists "$backend_dir/$file" 50; then
                ERRORS=$((ERRORS + 1))
            fi
        done

        # Check for test files
        if ! check_min_files "$backend_dir" "*_test.go" 3 "Backend test files"; then
            ERRORS=$((ERRORS + 1))
        fi

        # Check internal structure
        local internal_dirs=(
            "internal/models"
            "internal/handler"
            "internal/service"
            "internal/repository"
        )

        for dir in "${internal_dirs[@]}"; do
            if [[ -d "$backend_dir/$dir" ]]; then
                log_success "Backend structure: $dir exists"
            else
                log_warning "Backend structure: $dir missing"
            fi
        done
    fi

    # ----------------------------------------
    # Frontend Validation
    # ----------------------------------------
    log_info "Checking frontend..."

    local frontend_dir="$project_dir/frontend"
    if ! check_dir_exists "$frontend_dir"; then
        log_error "CRITICAL: Frontend directory is empty or missing!"
        ERRORS=$((ERRORS + 1))
    else
        # Check essential frontend files
        local frontend_files=(
            "package.json"
            "vite.config.ts"
            "src/App.tsx"
            "src/main.tsx"
        )

        for file in "${frontend_files[@]}"; do
            if ! check_file_exists "$frontend_dir/$file" 20; then
                ERRORS=$((ERRORS + 1))
            fi
        done

        # Check for test files
        if ! check_min_files "$frontend_dir/src" "*.test.tsx" 3 "Frontend test files"; then
            log_warning "Frontend test files below minimum"
        fi

        # Check for components
        local component_count=$(count_files "$frontend_dir/src/components" "*.tsx")
        if [[ $component_count -lt 2 ]]; then
            log_warning "Few component files: $component_count (expected 2+)"
        else
            log_success "Component files: $component_count"
        fi
    fi

    # ----------------------------------------
    # Docker Validation
    # ----------------------------------------
    log_info "Checking Docker configuration..."

    # Check docker-compose.yml
    if ! check_file_exists "$project_dir/docker-compose.yml" 200; then
        ERRORS=$((ERRORS + 1))
    else
        # Validate docker-compose structure
        if command -v docker &> /dev/null; then
            cd "$project_dir"
            if docker compose config > /dev/null 2>&1; then
                log_success "docker-compose.yml is valid"
            else
                log_error "docker-compose.yml has syntax errors"
                ERRORS=$((ERRORS + 1))
            fi
            cd "$PROJECT_ROOT"
        fi
    fi

    # Check Dockerfiles
    if ! check_file_exists "$project_dir/backend/Dockerfile" 100; then
        ERRORS=$((ERRORS + 1))
    fi

    if ! check_file_exists "$project_dir/frontend/Dockerfile" 50; then
        ERRORS=$((ERRORS + 1))
    fi

    # Check migrations
    local migration_count=$(count_files "$project_dir/backend/migrations" "*.sql")
    if [[ $migration_count -lt 1 ]]; then
        log_warning "No migration files found"
    else
        log_success "Migration files: $migration_count"
    fi

    # ----------------------------------------
    # Implementation Result Validation
    # ----------------------------------------
    local result_file=".aida/results/impl-complete.json"
    if [[ -f "$result_file" ]]; then
        log_success "Implementation completion report exists"

        if command -v jq &> /dev/null; then
            # Check quality gates in report
            if jq -e '.quality_gates.all_passed == true' "$result_file" > /dev/null 2>&1; then
                log_success "Quality gates: all passed (per report)"
            else
                log_warning "Quality gates: not all passed (per report)"
            fi

            # Check verification data
            if jq -e '.verification.backend.test_output' "$result_file" > /dev/null 2>&1; then
                log_success "Backend test output included in report"
            else
                log_warning "Backend test output missing from report"
            fi

            if jq -e '.verification.frontend.test_output' "$result_file" > /dev/null 2>&1; then
                log_success "Frontend test output included in report"
            else
                log_warning "Frontend test output missing from report"
            fi
        fi
    else
        log_warning "Implementation completion report not found"
    fi
}

# ============================================
# Execute Validation
# ============================================
case $PHASE in
    spec)
        validate_spec
        ;;
    impl)
        validate_impl
        ;;
    all)
        validate_spec
        validate_impl
        ;;
    *)
        log_error "Unknown phase: $PHASE"
        echo "Valid phases: spec, impl, all"
        exit 1
        ;;
esac

# ============================================
# Summary
# ============================================
echo ""
log_section "Validation Summary"

if [[ $ERRORS -eq 0 ]]; then
    log_success "All validations passed"
    exit 0
else
    log_error "$ERRORS validation error(s) found"
    exit 1
fi
