#!/bin/bash
# AIDA Common Utilities
# Shared functions for quality gates and validation scripts

set -euo pipefail

# ===========================================
# Container Runtime Detection (Docker/Podman)
# ===========================================

detect_container_runtime() {
    # Check if DOCKER_HOST is already set for podman
    if [[ -n "${DOCKER_HOST:-}" ]] && [[ "$DOCKER_HOST" == *"podman"* ]]; then
        CONTAINER_RUNTIME="podman"
        COMPOSE_CMD="docker compose"
        return 0
    fi

    # Check for podman socket
    if [[ -S "/run/user/$(id -u)/podman/podman.sock" ]]; then
        export DOCKER_HOST="unix:///run/user/$(id -u)/podman/podman.sock"
        CONTAINER_RUNTIME="podman"
        # Prefer podman-compose over docker compose for podman
        if command -v podman-compose > /dev/null 2>&1; then
            COMPOSE_CMD="podman-compose"
        else
            COMPOSE_CMD="docker compose"
        fi
        return 0
    fi

    # Check for docker daemon
    if docker info > /dev/null 2>&1; then
        CONTAINER_RUNTIME="docker"
        COMPOSE_CMD="docker compose"
        return 0
    fi

    # Check for podman directly
    if command -v podman > /dev/null 2>&1; then
        if podman info > /dev/null 2>&1; then
            CONTAINER_RUNTIME="podman"
            if command -v podman-compose > /dev/null 2>&1; then
                COMPOSE_CMD="podman-compose"
            else
                COMPOSE_CMD="podman compose"
            fi
            return 0
        fi
    fi

    CONTAINER_RUNTIME="none"
    COMPOSE_CMD=""
    return 1
}

# Initialize container runtime
CONTAINER_RUNTIME="${CONTAINER_RUNTIME:-}"
COMPOSE_CMD="${COMPOSE_CMD:-}"

# Auto-detect if not already set
if [[ -z "$CONTAINER_RUNTIME" ]]; then
    detect_container_runtime || true
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

# Alias for log_warning
log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $*"
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]] || [[ "${VERBOSE:-false}" == "true" ]]; then
        echo -e "${CYAN}[DEBUG]${NC} $*"
    fi
}

log_section() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Gate execution helper
run_gate() {
    local gate_num=$1
    local gate_name=$2
    local gate_cmd=$3

    echo ""
    log_info "[Gate $gate_num] $gate_name"

    if eval "$gate_cmd" > /tmp/gate_output_$gate_num.txt 2>&1; then
        log_success "Gate $gate_num PASSED: $gate_name"
        return 0
    else
        log_error "Gate $gate_num FAILED: $gate_name"
        echo "--- Output ---"
        cat /tmp/gate_output_$gate_num.txt
        echo "--- End Output ---"
        return 1
    fi
}

# Check if file exists and is not empty
check_file_exists() {
    local file_path=$1
    local min_size=${2:-1}

    if [[ ! -f "$file_path" ]]; then
        log_error "File not found: $file_path"
        return 1
    fi

    local size=$(wc -c < "$file_path")
    if [[ $size -lt $min_size ]]; then
        log_error "File too small: $file_path (${size} bytes, minimum ${min_size})"
        return 1
    fi

    log_success "File exists: $file_path (${size} bytes)"
    return 0
}

# Check if directory exists and is not empty
check_dir_exists() {
    local dir_path=$1

    if [[ ! -d "$dir_path" ]]; then
        log_error "Directory not found: $dir_path"
        return 1
    fi

    local count=$(find "$dir_path" -type f | wc -l)
    if [[ $count -eq 0 ]]; then
        log_error "Directory is empty: $dir_path"
        return 1
    fi

    log_success "Directory exists: $dir_path ($count files)"
    return 0
}

# Count files matching pattern
count_files() {
    local dir=$1
    local pattern=$2
    find "$dir" -name "$pattern" -type f 2>/dev/null | wc -l
}

# Check minimum file count
check_min_files() {
    local dir=$1
    local pattern=$2
    local min_count=$3
    local description=$4

    local count=$(count_files "$dir" "$pattern")

    if [[ $count -lt $min_count ]]; then
        log_error "$description: Found $count files, minimum $min_count required"
        return 1
    fi

    log_success "$description: Found $count files (minimum $min_count)"
    return 0
}

# Wait for service to be healthy
wait_for_service() {
    local url=$1
    local max_attempts=${2:-30}
    local delay=${3:-2}

    log_info "Waiting for service at $url..."

    for ((i=1; i<=max_attempts; i++)); do
        if curl -sf "$url" > /dev/null 2>&1; then
            log_success "Service is healthy: $url"
            return 0
        fi
        echo -n "."
        sleep $delay
    done

    echo ""
    log_error "Service did not become healthy: $url (after $max_attempts attempts)"
    return 1
}

# Get project directory
get_project_dir() {
    local project=$1
    echo "$project"
}

# Validate project name
validate_project_name() {
    local project=$1

    if [[ -z "$project" ]]; then
        log_error "Project name is required"
        return 1
    fi

    if [[ ! "$project" =~ ^[a-zA-Z][a-zA-Z0-9_-]*$ ]]; then
        log_error "Invalid project name: $project (must start with letter, contain only alphanumeric, dash, underscore)"
        return 1
    fi

    return 0
}

# Print summary (for quality gates)
print_gate_summary() {
    local total="${1:-0}"
    local passed="${2:-0}"
    local failed=$((total - passed))

    echo ""
    log_section "Summary"
    echo "Total gates: $total"
    echo -e "Passed: ${GREEN}$passed${NC}"
    echo -e "Failed: ${RED}$failed${NC}"
    echo ""

    if [[ $failed -eq 0 ]]; then
        log_success "ALL GATES PASSED"
        return 0
    else
        log_error "$failed GATE(S) FAILED"
        return 1
    fi
}

# Compose command wrapper (uses detected runtime)
compose_cmd() {
    if [[ -z "$COMPOSE_CMD" ]]; then
        detect_container_runtime || {
            log_error "No container runtime available (Docker or Podman)"
            return 1
        }
    fi
    $COMPOSE_CMD "$@"
}

# Cleanup function for containers
cleanup_docker() {
    local project_dir=$1
    log_info "Cleaning up Docker resources..."
    cd "$project_dir"
    compose_cmd down --remove-orphans 2>/dev/null || true
}

# Get container runtime info
get_runtime_info() {
    echo "Container Runtime: ${CONTAINER_RUNTIME:-none}"
    echo "Compose Command: ${COMPOSE_CMD:-not available}"
    if [[ -n "${DOCKER_HOST:-}" ]]; then
        echo "Docker Host: $DOCKER_HOST"
    fi
}

# ===========================================
# AIDA Session Management
# ===========================================

# Get AIDA project root
get_aida_root() {
    local script_dir="${1:-$(pwd)}"
    local project_root="$script_dir"

    # Use CLAUDE_PROJECT_DIR if available
    if [[ -n "${CLAUDE_PROJECT_DIR:-}" ]]; then
        project_root="$CLAUDE_PROJECT_DIR"
    fi

    echo "$project_root"
}

# Get session file path
get_session_file() {
    local root
    root=$(get_aida_root)
    echo "$root/.aida/state/session.json"
}

# Check if AIDA session is active
is_session_active() {
    local session_file
    session_file=$(get_session_file)
    [[ -f "$session_file" ]]
}

# Get session value
get_session_value() {
    local key="$1"
    local default="${2:-}"
    local session_file
    session_file=$(get_session_file)

    if [[ -f "$session_file" ]]; then
        local value
        value=$(jq -r ".$key // empty" "$session_file" 2>/dev/null)
        if [[ -n "$value" ]]; then
            echo "$value"
            return 0
        fi
    fi

    echo "$default"
}

# Set session value
set_session_value() {
    local key="$1"
    local value="$2"
    local session_file
    session_file=$(get_session_file)

    if [[ -f "$session_file" ]]; then
        local updated
        updated=$(jq --arg key "$key" --arg val "$value" '.[$key] = $val' "$session_file")
        echo "$updated" > "$session_file"
    fi
}

# Initialize AIDA directories
init_aida_dirs() {
    local root
    root=$(get_aida_root)

    mkdir -p "$root/.aida/state"
    mkdir -p "$root/.aida/artifacts"
    mkdir -p "$root/.aida/logs"
    mkdir -p "$root/.aida/tdd-evidence"
    mkdir -p "$root/.aida/fix-plans"
    mkdir -p "$root/.aida/queue"
    mkdir -p "$root/.aida/worktrees"
    mkdir -p "$root/.aida/results"
}

# ===========================================
# JSON Response Helpers (Official Claude Code format)
# ===========================================

# Output allow decision (Official Claude Code format: null = allow)
output_allow() {
    local reason="${1:-Task completed successfully}"
    local system_msg="${2:-}"
    if [[ -n "$system_msg" ]]; then
        cat << EOF
{
  "decision": null,
  "reason": "$reason",
  "systemMessage": "$system_msg"
}
EOF
    else
        cat << EOF
{
  "decision": null,
  "reason": "$reason"
}
EOF
    fi
}

# Alias for backward compatibility
output_approve() {
    output_allow "$@"
}

# Output block decision
output_block() {
    local reason="${1:-Quality gates not passed}"
    local system_msg="${2:-}"
    if [[ -n "$system_msg" ]]; then
        cat << EOF
{
  "decision": "block",
  "reason": "$reason",
  "systemMessage": "$system_msg"
}
EOF
    else
        cat << EOF
{
  "decision": "block",
  "reason": "$reason"
}
EOF
    fi
}

# ===========================================
# Test Counting Utilities
# ===========================================

# Count Go tests
count_go_tests() {
    local dir="$1"
    grep -r "func Test" "$dir" --include="*_test.go" 2>/dev/null | wc -l
}

# Count JavaScript/TypeScript tests
count_js_tests() {
    local dir="$1"
    grep -rE "^\s*(it|test)\s*\(" "$dir" --include="*.test.ts" --include="*.test.tsx" --include="*.test.js" --include="*.test.jsx" 2>/dev/null | wc -l
}

# Count E2E tests
count_e2e_tests() {
    local dir="$1"
    grep -rE "^\s*(it|test)\s*\(" "$dir" --include="*.spec.ts" --include="*.spec.tsx" 2>/dev/null | wc -l
}

# Count Python tests
count_python_tests() {
    local dir="$1"
    grep -rE "^\s*(def test_|async def test_)" "$dir" --include="*_test.py" --include="test_*.py" 2>/dev/null | wc -l
}

# ===========================================
# Timestamp Utilities
# ===========================================

# Get ISO timestamp
iso_timestamp() {
    date -Iseconds
}

# Get Unix timestamp
unix_timestamp() {
    date +%s
}

# ===========================================
# Command & Dependency Utilities
# ===========================================

# Check if a command is available
require_command() {
    local cmd="$1"
    local install_hint="${2:-}"

    if ! command -v "$cmd" &>/dev/null; then
        log_error "Required command not found: $cmd"
        if [[ -n "$install_hint" ]]; then
            echo "  Install: $install_hint" >&2
        fi
        return 1
    fi
    log_debug "Found command: $cmd"
    return 0
}

# Check multiple commands
require_commands() {
    local missing=0
    for cmd in "$@"; do
        if ! command -v "$cmd" &>/dev/null; then
            log_error "Missing command: $cmd"
            ((missing++))
        fi
    done
    return $missing
}

# ===========================================
# JSON File Utilities
# ===========================================

# Ensure JSON file exists with default content
ensure_json_file() {
    local file="$1"
    # Use explicit quoting to avoid bash brace expansion issues
    local default_json='{}'
    local default="${2:-$default_json}"

    if [[ ! -f "$file" ]]; then
        mkdir -p "$(dirname "$file")"
        echo "$default" > "$file"
        log_debug "Created JSON file: $file"
    fi
}

# Safe JSON update (atomic write)
safe_jq_update() {
    local file="$1"
    local jq_filter="$2"

    if [[ ! -f "$file" ]]; then
        log_error "File not found: $file"
        return 1
    fi

    local tmp_file="${file}.tmp.$$"
    if jq "$jq_filter" "$file" > "$tmp_file" 2>/dev/null; then
        mv "$tmp_file" "$file"
        return 0
    else
        rm -f "$tmp_file"
        log_error "JSON update failed: $file"
        return 1
    fi
}

# ===========================================
# File System Utilities
# ===========================================

# Create directory if not exists
ensure_dir() {
    local dir="$1"
    if [[ ! -d "$dir" ]]; then
        mkdir -p "$dir"
        log_debug "Created directory: $dir"
    fi
}

# Safe file backup
backup_file() {
    local file="$1"
    local backup_dir="${2:-.aida/backups}"

    if [[ -f "$file" ]]; then
        ensure_dir "$backup_dir"
        local backup_name="$(basename "$file").$(date +%Y%m%d_%H%M%S)"
        cp "$file" "$backup_dir/$backup_name"
        log_debug "Backed up: $file -> $backup_dir/$backup_name"
    fi
}

# Export functions for use in other scripts
export -f log_info log_success log_warning log_warn log_error log_debug log_section
export -f run_gate check_file_exists check_dir_exists
export -f count_files check_min_files wait_for_service
export -f get_project_dir validate_project_name print_gate_summary
export -f detect_container_runtime compose_cmd cleanup_docker get_runtime_info
export -f get_aida_root get_session_file is_session_active
export -f get_session_value set_session_value init_aida_dirs
export -f output_allow output_approve output_block
export -f count_go_tests count_js_tests count_e2e_tests count_python_tests
export -f iso_timestamp unix_timestamp
export -f require_command require_commands
export -f ensure_json_file safe_jq_update ensure_dir backup_file
export CONTAINER_RUNTIME COMPOSE_CMD
export RED GREEN YELLOW BLUE CYAN NC
