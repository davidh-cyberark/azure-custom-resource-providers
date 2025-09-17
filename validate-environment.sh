#!/bin/bash

# =============================================================================
# Environment Validation Script for CyberArk Custom Provider
# =============================================================================
# This script validates your .env configuration and checks connectivity
# to required services.
#
# Usage: ./validate-environment.sh [--verbose]
# =============================================================================

source color-lib.sh
set -e

# Configuration
ENV_FILE=".env"
VERBOSE=false
ERRORS=0
WARNINGS=0

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
    --verbose)
        VERBOSE=true
        shift
        ;;
    -h | --help)
        echo "Usage: $0 [--verbose]"
        echo ""
        echo "This script validates your environment configuration and connectivity."
        echo ""
        echo "Options:"
        echo "  --verbose           Show detailed output"
        echo "  -h, --help         Show this help message"
        exit 0
        ;;
    *)
        error "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
    esac
done

# Function to increment error counter
add_error() {
    ((ERRORS++))
    error "$1"
}

# Function to increment warning counter
add_warning() {
    ((WARNINGS++))
    warning "$1"
}

# Function to log verbose information
verbose() {
    if [ "$VERBOSE" = true ]; then
        info "$1"
    fi
}

# Function to check if required command exists
check_command() {
    local cmd="$1"
    local description="$2"

    if command -v "$cmd" &>/dev/null; then
        success "$description is installed"
        if [ "$VERBOSE" = true ]; then
            local version
            case $cmd in
            "az")
                version=$(az version --query '"azure-cli"' -o tsv 2>/dev/null)
                ;;
            "docker")
                version=$(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',')
                ;;
            "curl")
                version=$(curl --version 2>/dev/null | head -n1 | cut -d' ' -f2)
                ;;
            "jq")
                version=$(jq --version 2>/dev/null)
                ;;
            *)
                version="unknown"
                ;;
            esac
            verbose "  Version: $version"
        fi
        return 0
    else
        add_error "$description is not installed"
        return 1
    fi
}

# Function to validate environment variable
validate_env_var() {
    local var_name="$1"
    local description="$2"
    local required="${3:-true}"
    local pattern="$4"

    local value="${!var_name}"

    if [ -z "$value" ]; then
        if [ "$required" = true ]; then
            add_error "$description ($var_name) is required but not set"
            return 1
        else
            add_warning "$description ($var_name) is not set (optional)"
            return 0
        fi
    fi

    # Check pattern if provided
    if [ -n "$pattern" ] && [[ ! "$value" =~ $pattern ]]; then
        add_error "$description ($var_name) format is invalid: $value"
        return 1
    fi

    success "$description is configured"
    verbose "  $var_name=$value"
    return 0
}

# Function to test Azure CLI connectivity
test_azure_cli() {
    verbose "Testing Azure CLI connectivity..."

    if ! az account show &>/dev/null; then
        add_error "Not logged in to Azure CLI (run: az login)"
        return 1
    fi

    local subscription_id
    subscription_id=$(az account show --query id -o tsv 2>/dev/null)

    if [ -n "$AZURE_SUBSCRIPTION_ID" ] && [ "$subscription_id" != "$AZURE_SUBSCRIPTION_ID" ]; then
        add_warning "Current Azure subscription ($subscription_id) differs from configured AZURE_SUBSCRIPTION_ID ($AZURE_SUBSCRIPTION_ID)"
    fi

    success "Azure CLI connectivity verified"
    verbose "  Current subscription: $subscription_id"

    # Test resource group access
    if [ -n "$RESOURCE_GROUP" ]; then
        verbose "Testing resource group access..."
        if az group show --name "$RESOURCE_GROUP" &>/dev/null; then
            success "Resource group '$RESOURCE_GROUP' exists and is accessible"
        else
            add_warning "Resource group '$RESOURCE_GROUP' does not exist or is not accessible"
        fi
    fi

    # Test ACR access
    if [ -n "$ACR_NAME" ]; then
        verbose "Testing ACR access..."
        if az acr show --name "$ACR_NAME" &>/dev/null; then
            success "Azure Container Registry '$ACR_NAME' exists and is accessible"
        else
            add_warning "Azure Container Registry '$ACR_NAME' does not exist or is not accessible"
        fi
    fi

    return 0
}

# Function to test CyberArk connectivity
test_cyberark_connectivity() {
    verbose "Testing CyberArk connectivity..."

    if [ -z "$CYBERARK_ID_TENANT_URL" ] || [ -z "$CYBERARK_PCLOUD_URL" ]; then
        add_warning "CyberArk URLs not configured, skipping connectivity test"
        return 0
    fi

    # Test Identity tenant URL
    verbose "Testing CyberArk Identity connectivity..."
    if curl -s --max-time 10 "$CYBERARK_ID_TENANT_URL/.well-known/openid_configuration" >/dev/null 2>&1; then
        success "CyberArk Identity tenant is reachable"
    else
        add_error "Cannot reach CyberArk Identity tenant at $CYBERARK_ID_TENANT_URL"
    fi

    # Test Privileged Cloud URL (basic connectivity)
    verbose "Testing CyberArk Privileged Cloud connectivity..."
    if curl -s --max-time 10 "$CYBERARK_PCLOUD_URL" >/dev/null 2>&1; then
        success "CyberArk Privileged Cloud is reachable"
    else
        add_error "Cannot reach CyberArk Privileged Cloud at $CYBERARK_PCLOUD_URL"
    fi

    return 0
}

# Function to test Docker connectivity
test_docker() {
    verbose "Testing Docker..."

    if ! docker info &>/dev/null; then
        add_error "Docker is not running or not accessible"
        return 1
    fi

    success "Docker is running and accessible"

    # Test Docker login to ACR if credentials are available
    if [ -n "$ACR_NAME" ] && command -v az &>/dev/null && az account show &>/dev/null; then
        verbose "Testing ACR Docker login..."
        if az acr login --name "$ACR_NAME" &>/dev/null; then
            success "ACR Docker login successful"
        else
            add_warning "ACR Docker login failed (may need manual login)"
        fi
    fi

    return 0
}

# Function to validate project structure
validate_project_structure() {
    verbose "Validating project structure..."

    local required_files=(
        "custom-provider/main.go"
        "custom-provider/Dockerfile"
        "custom-provider/Makefile"
        "custom-provider/go.mod"
        "rebuild-and-run.sh"
        "color-lib.sh"
    )

    for file in "${required_files[@]}"; do
        if [ -f "$file" ]; then
            success "Found $file"
        else
            add_error "Missing required file: $file"
        fi
    done

    # Check if scripts are executable
    local executable_files=(
        "rebuild-and-run.sh"
        "env-setup.sh"
        "validate-environment.sh"
    )

    for file in "${executable_files[@]}"; do
        if [ -f "$file" ]; then
            if [ -x "$file" ]; then
                success "$file is executable"
            else
                add_warning "$file is not executable (run: chmod +x $file)"
            fi
        fi
    done

    return 0
}

# Function to show configuration summary
show_config_summary() {
    header "📋 Configuration Summary"

    echo "Azure Configuration:"
    echo "  Region: ${LOCATION:-not set}"
    echo "  Resource Group: ${RESOURCE_GROUP:-not set}"
    echo "  Environment: ${ENVIRONMENT:-not set}"
    echo "  ACR Name: ${ACR_NAME:-not set}"
    echo

    echo "CyberArk Configuration:"
    echo "  Identity URL: ${CYBERARK_ID_TENANT_URL:-not set}"
    echo "  Privileged Cloud URL: ${CYBERARK_PCLOUD_URL:-not set}"
    echo "  PAM User: ${CYBERARK_PAM_USER:-not set}"
    echo "  PAM Password: ${CYBERARK_PAM_PASSWORD:+***configured***}"
    echo

    echo "Application Configuration:"
    echo "  Port: ${APP_PORT:-8080}"
    echo "  Log Level: ${LOG_LEVEL:-INFO}"
    echo "  Container Image: ${CONTAINER_IMAGE:-not set}"
    echo
}

# Main validation function
main() {
    header "🔍 CyberArk Custom Provider Environment Validation"
    echo

    # Check if .env file exists
    if [ ! -f "$ENV_FILE" ]; then
        error ".env file not found!"
        echo "Run ./env-setup.sh to create your configuration file."
        exit 1
    fi

    # Load environment variables
    verbose "Loading environment from $ENV_FILE..."
    set -a # automatically export all variables
    source "$ENV_FILE"
    set +a # stop automatically exporting

    success "Environment file loaded"

    if [ "$VERBOSE" = true ]; then
        echo
        show_config_summary
    fi

    echo
    header "🛠️ Checking Prerequisites"

    # Check required commands
    check_command "az" "Azure CLI"
    check_command "docker" "Docker"
    check_command "curl" "curl"
    check_command "jq" "jq (JSON processor)" || add_warning "jq is recommended for JSON processing"

    echo
    header "📁 Validating Project Structure"
    validate_project_structure

    echo
    header "⚙️ Validating Environment Variables"

    # Required Azure variables
    validate_env_var "LOCATION" "Azure region"
    validate_env_var "RESOURCE_GROUP" "Azure resource group"
    validate_env_var "ENVIRONMENT" "Environment name"
    validate_env_var "ACR_NAME" "Azure Container Registry name" true "^[a-zA-Z0-9]{5,50}$"
    validate_env_var "CUSTOM_PROVIDER_NAME" "Custom Provider name"

    # Required CyberArk variables
    validate_env_var "CYBERARK_ID_TENANT_URL" "CyberArk Identity tenant URL" true "^https://.*\\.id\\.cyberark\\.cloud$"
    validate_env_var "CYBERARK_PCLOUD_URL" "CyberArk Privileged Cloud URL" true "^https://.*\\.privilegecloud\\.cyberark\\.cloud$"
    validate_env_var "CYBERARK_PAM_USER" "CyberArk PAM user"
    validate_env_var "CYBERARK_PAM_PASSWORD" "CyberArk PAM password"

    # Optional variables
    validate_env_var "AZURE_SUBSCRIPTION_ID" "Azure subscription ID" false
    validate_env_var "APP_PORT" "Application port" false "^[0-9]+$"
    validate_env_var "LOG_LEVEL" "Log level" false "^(DEBUG|INFO|WARN|ERROR)$"

    echo
    header "🌐 Testing Connectivity"

    # Test Azure CLI
    if command -v az &>/dev/null; then
        test_azure_cli
    else
        add_warning "Azure CLI not available, skipping Azure connectivity test"
    fi

    # Test Docker
    if command -v docker &>/dev/null; then
        test_docker
    else
        add_warning "Docker not available, skipping Docker test"
    fi

    # Test CyberArk connectivity
    if command -v curl &>/dev/null; then
        test_cyberark_connectivity
    else
        add_warning "curl not available, skipping CyberArk connectivity test"
    fi

    echo
    header "📊 Validation Summary"

    if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
        success "All validation checks passed! ✨"
        echo "Your environment is ready for deployment."
    elif [ $ERRORS -eq 0 ]; then
        warning "Validation completed with $WARNINGS warning(s)"
        echo "Your environment should work, but consider addressing the warnings."
    else
        error "Validation failed with $ERRORS error(s) and $WARNINGS warning(s)"
        echo "Please fix the errors before proceeding with deployment."
        exit 1
    fi

    echo
    info "Next steps:"
    echo "1. Test locally: ./rebuild-and-run.sh"
    echo "2. Deploy infrastructure: cd infra && az deployment group create ..."
    echo "3. Build and deploy: cd custom-provider && make build"
    echo
    echo "For detailed instructions, see README.md"
}

# Run main function
main "$@"
