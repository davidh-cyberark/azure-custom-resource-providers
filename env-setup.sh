#!/bin/bash

# =============================================================================
# CyberArk Custom Provider Setup Script
# =============================================================================
# This script helps you configure the environment for the CyberArk Custom Provider
# by copying the template and prompting for required values.
#
# Usage: ./env-setup.sh [--non-interactive]
# =============================================================================

source color-lib.sh
set -e # Exit on any error

# Configuration
ENV_FILE=".env"
TEMPLATE_FILE=".env.template"
NON_INTERACTIVE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
    --non-interactive)
        NON_INTERACTIVE=true
        shift
        ;;
    -h | --help)
        echo "Usage: $0 [--non-interactive]"
        echo ""
        echo "Options:"
        echo "  --non-interactive   Run in non-interactive mode (use template defaults)"
        echo "  -h, --help          Show this help message"
        exit 0
        ;;
    *)
        error "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
    esac
done

# Function to prompt for input with default value
prompt_with_default() {
    local prompt="$1"
    local default="$2"
    local var_name="$3"
    local sensitive="${4:-false}"

    if [ "$NON_INTERACTIVE" = true ]; then
        eval "$var_name=\"$default\""
        return
    fi

    if [ "$sensitive" = true ]; then
        echo -ne "${CYAN}$prompt${NC}"
        if [ -n "$default" ]; then
            echo -ne " [${default}]: "
        else
            echo -ne ": "
        fi
        read -s user_input
        echo # New line after hidden input
    else
        echo -ne "${CYAN}$prompt${NC}"
        if [ -n "$default" ]; then
            echo -ne " [${default}]: "
        else
            echo -ne ": "
        fi
        read user_input
    fi

    if [ -z "$user_input" ]; then
        eval "$var_name=\"$default\""
    else
        eval "$var_name=\"$user_input\""
    fi
}

# Function to validate required fields
validate_required() {
    local value="$1"
    local field_name="$2"

    if [ -z "$value" ]; then
        error "Required field '$field_name' cannot be empty"
        return 1
    fi
    return 0
}

# Function to validate Azure resource names
validate_azure_name() {
    local name="$1"
    local type="$2"

    case $type in
    "acr")
        if [[ ! "$name" =~ ^[a-zA-Z0-9]{5,50}$ ]]; then
            error "ACR name must be 5-50 alphanumeric characters"
            return 1
        fi
        ;;
    "resource-group")
        if [[ ! "$name" =~ ^[a-zA-Z0-9._-]{1,90}$ ]]; then
            error "Resource group name must be 1-90 characters (alphanumeric, periods, underscores, hyphens)"
            return 1
        fi
        ;;
    esac
    return 0
}

# Function to check if Azure CLI is installed and user is logged in
check_azure_cli() {
    if ! command -v az &>/dev/null; then
        error "Azure CLI is not installed. Please install it first:"
        echo "  https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
        return 1
    fi

    # Check if user is logged in
    if ! az account show &>/dev/null; then
        warning "You are not logged in to Azure CLI"
        info "Please run: az login"
        return 1
    fi

    return 0
}

# Function to get current Azure subscription
get_azure_subscription() {
    if check_azure_cli; then
        az account show --query id -o tsv 2>/dev/null || echo ""
    else
        echo ""
    fi
}

# Main setup function
main() {
    header "🚀 CyberArk Custom Provider Setup"
    echo

    # Check if template exists
    if [ ! -f "$TEMPLATE_FILE" ]; then
        error "Template file '$TEMPLATE_FILE' not found!"
        echo "Please ensure you're running this script from the project root directory."
        exit 1
    fi

    # Check if .env already exists
    if [ -f "$ENV_FILE" ]; then
        warning ".env file already exists!"
        if [ "$NON_INTERACTIVE" = false ]; then
            echo -ne "${YELLOW}Do you want to overwrite it? (y/N): ${NC}"
            read -r overwrite
            if [[ ! "$overwrite" =~ ^[Yy]$ ]]; then
                info "Setup cancelled. Existing .env file preserved."
                exit 0
            fi
        else
            info "Non-interactive mode: backing up existing .env to .env.backup"
            cp "$ENV_FILE" "$ENV_FILE.backup"
        fi
    fi

    # Check Azure CLI
    if ! check_azure_cli; then
        warning "Azure CLI not available. Some validations will be skipped."
    fi

    info "This script will help you configure your environment."
    info "Press Enter to use default values shown in brackets."
    echo

    # Get current Azure subscription for default
    current_subscription=$(get_azure_subscription)

    # Collect Azure configuration
    header "📋 Azure Configuration"
    prompt_with_default "Azure region" "eastus" LOCATION
    prompt_with_default "Resource group name" "cyberark-custom-provider-rg" RESOURCE_GROUP
    validate_azure_name "$RESOURCE_GROUP" "resource-group" || exit 1

    prompt_with_default "Environment (dev/test/staging/prod)" "dev" ENVIRONMENT
    prompt_with_default "Azure subscription ID" "$current_subscription" AZURE_SUBSCRIPTION_ID

    echo
    header "🐳 Container Registry Configuration"
    prompt_with_default "ACR name (must be globally unique)" "youracr" ACR_NAME
    validate_azure_name "$ACR_NAME" "acr" || exit 1

    ACR_LOGIN_SERVER="${ACR_NAME}.azurecr.io"

    echo
    header "⚙️ Custom Provider Configuration"
    prompt_with_default "Custom Provider name" "cyberark-custom-provider" CUSTOM_PROVIDER_NAME
    prompt_with_default "Container image name" "cyberark-custom-provider:latest" CONTAINER_IMAGE

    echo
    header "🔐 CyberArk Configuration"
    info "You can find these URLs in your CyberArk admin consoles"

    prompt_with_default "CyberArk Identity tenant URL" "https://your-tenant.id.cyberark.cloud" CYBERARK_ID_TENANT_URL
    validate_required "$CYBERARK_ID_TENANT_URL" "CyberArk Identity tenant URL" || exit 1

    prompt_with_default "CyberArk Privileged Cloud URL" "https://your-tenant.privilegecloud.cyberark.cloud" CYBERARK_PCLOUD_URL
    validate_required "$CYBERARK_PCLOUD_URL" "CyberArk Privileged Cloud URL" || exit 1

    prompt_with_default "CyberArk PAM user" "your-pam-user@cyberark.cloud.tenant" CYBERARK_PAM_USER
    validate_required "$CYBERARK_PAM_USER" "CyberArk PAM user" || exit 1

    prompt_with_default "CyberArk PAM password" "" CYBERARK_PAM_PASSWORD true
    validate_required "$CYBERARK_PAM_PASSWORD" "CyberArk PAM password" || exit 1

    echo
    header "🎯 Application Configuration"
    prompt_with_default "Application port" "8080" APP_PORT
    prompt_with_default "Log level (DEBUG/INFO/WARN/ERROR)" "INFO" LOG_LEVEL

    echo
    header "📝 Creating .env file..."

    # Create the .env file
    cat >"$ENV_FILE" <<EOF
# =============================================================================
# CyberArk Custom Provider Configuration
# Generated by setup script on $(date)
# =============================================================================

# Azure Configuration
LOCATION="$LOCATION"
RESOURCE_GROUP="$RESOURCE_GROUP"
ENVIRONMENT="$ENVIRONMENT"
AZURE_SUBSCRIPTION_ID="$AZURE_SUBSCRIPTION_ID"

# Azure Container Registry
ACR_NAME="$ACR_NAME"
ACR_LOGIN_SERVER="$ACR_LOGIN_SERVER"

# Custom Provider Configuration
CUSTOM_PROVIDER_NAME="$CUSTOM_PROVIDER_NAME"
CONTAINER_IMAGE="$CONTAINER_IMAGE"
CONTAINER_APPS_ENVIRONMENT="\${ENVIRONMENT}-env"

# CyberArk Configuration
CYBERARK_ID_TENANT_URL="$CYBERARK_ID_TENANT_URL"
CYBERARK_PCLOUD_URL="$CYBERARK_PCLOUD_URL"
CYBERARK_PAM_USER="$CYBERARK_PAM_USER"
CYBERARK_PAM_PASSWORD="$CYBERARK_PAM_PASSWORD"

# Application Configuration
APP_PORT="$APP_PORT"
LOG_LEVEL="$LOG_LEVEL"

# Development Configuration
LOCAL_CONTAINER_NAME="cyberark-local-test"
LOCAL_PORT="8080"
DEBUG_MODE="false"

# Deployment Configuration
MAX_REPLICAS="10"
MIN_REPLICAS="1"
CPU_ALLOCATION="0.25"
MEMORY_ALLOCATION="0.5Gi"
INGRESS_EXTERNAL="true"
INGRESS_TARGET_PORT="8080"

# Example Testing Configuration
SAFE_NAME="test-safe"
SAFE_DESCRIPTION="Test safe created via Azure Custom Provider"
EOF

    success ".env file created successfully!"

    echo
    header "🔍 Next Steps"
    echo "1. Review your .env file: cat .env"
    echo "2. Test local build: ./rebuild-and-run.sh"
    echo "3. Deploy infrastructure: cd infra && az deployment group create ..."
    echo "4. Build and deploy application: cd custom-provider && make build"
    echo
    info "For detailed instructions, see README.md"

    echo
    header "⚠️ Security Reminders"
    warning "Your .env file contains sensitive information!"
    echo "• Never commit .env files to version control"
    echo "• Add .env to your .gitignore file"
    echo "• For production, consider using Azure Key Vault"
    echo "• Regularly rotate your CyberArk credentials"

    # Create .gitignore if it doesn't exist
    if [ ! -f ".gitignore" ]; then
        echo ".env" >.gitignore
        success "Created .gitignore with .env entry"
    elif ! grep -q "^\.env$" .gitignore; then
        echo ".env" >>.gitignore
        success "Added .env to existing .gitignore"
    fi

    echo
    success "Setup completed successfully! 🎉"
}

# Run main function
main "$@"
