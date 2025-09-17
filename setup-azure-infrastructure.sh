#!/bin/bash

# Azure Infrastructure Setup Script for CyberArk Custom Provider
# This script creates the necessary Azure resources for the custom provider

set -e # Exit on any error

source color-lib.sh

# Load configuration from .env file
if [[ -f ".env" ]]; then
    source .env
    info "Loaded configuration from .env file"
else
    error ".env file not found. Please create one with the required configuration."
    exit 1
fi

# Validate required environment variables
if [[ -z "$LOCATION" || -z "$RESOURCE_GROUP" || -z "$ACR_NAME" ]]; then
    error "Missing required environment variables. Please check your .env file."
    error "Required: LOCATION, RESOURCE_GROUP, ACR_NAME"
    exit 1
fi

# Function to check Azure CLI authentication
check_azure_auth() {
    info "Checking Azure CLI authentication..."
    if ! az account show >/dev/null 2>&1; then
        error "Azure CLI is not authenticated. Please run 'az login' first."
        exit 1
    fi

    local subscription_name=$(az account show --query "name" -o tsv)
    local subscription_id=$(az account show --query "id" -o tsv)
    success "Authenticated to Azure subscription: ${subscription_name} (${subscription_id})"
}

# Function to create resource group
create_resource_group() {
    info "Creating resource group: ${RESOURCE_GROUP}"

    if az group show --name "${RESOURCE_GROUP}" >/dev/null 2>&1; then
        warning "Resource group ${RESOURCE_GROUP} already exists"
    else
        az group create \
            --name "${RESOURCE_GROUP}" \
            --location "${LOCATION}" \
            --output none
        success "Resource group ${RESOURCE_GROUP} created successfully"
    fi
}

# Function to create Azure Container Registry
create_acr() {
    info "Creating Azure Container Registry: ${ACR_NAME}"

    if az acr show --name "${ACR_NAME}" --resource-group "${RESOURCE_GROUP}" >/dev/null 2>&1; then
        warning "ACR ${ACR_NAME} already exists"
    else
        az acr create \
            --name "${ACR_NAME}" \
            --resource-group "${RESOURCE_GROUP}" \
            --location "${LOCATION}" \
            --sku Standard \
            --admin-enabled true \
            --output none
        success "ACR ${ACR_NAME} created successfully"
    fi

    # Get ACR login server
    local login_server=$(az acr show --name "${ACR_NAME}" --resource-group "${RESOURCE_GROUP}" --query "loginServer" -o tsv)
    info "ACR Login Server: ${login_server}"
}

# Function to configure Docker authentication to ACR
configure_acr_auth() {
    info "Configuring Docker authentication to ACR..."

    # Login to ACR
    az acr login --name "${ACR_NAME}"
    success "Docker authenticated to ACR ${ACR_NAME}"

    # Get ACR credentials
    local acr_username=$(az acr credential show --name "${ACR_NAME}" --query "username" -o tsv)
    local acr_password=$(az acr credential show --name "${ACR_NAME}" --query "passwords[0].value" -o tsv)

    info "ACR Credentials:"
    info "  Username: ${acr_username}"
    info "  Password: [REDACTED] (stored in Azure)"

    # Update config.yml with credentials (optional)
    info "ACR setup completed. You can now push images to ${ACR_NAME}.azurecr.io"
}

# Function to show summary
show_summary() {
    success "Azure Infrastructure Setup Complete!"
    echo
    info "Created resources:"
    info "  Resource Group: ${RESOURCE_GROUP} (${LOCATION})"
    info "  Container Registry: ${ACR_NAME}.azurecr.io"
    echo
    info "Next steps:"
    info "  1. Push your Docker image: docker push ${ACR_NAME}.azurecr.io/cyberark-custom-provider:latest"
    info "  2. Deploy the custom provider infrastructure with Bicep"
    info "  3. Configure the custom provider endpoints"
}

# Main execution
main() {
    info "Starting Azure Infrastructure Setup for CyberArk Custom Provider"
    echo

    check_azure_auth
    create_resource_group
    create_acr
    configure_acr_auth
    show_summary
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
    --location)
        LOCATION="$2"
        shift 2
        ;;
    --resource-group)
        RESOURCE_GROUP="$2"
        shift 2
        ;;
    --acr-name)
        ACR_NAME="$2"
        shift 2
        ;;
    -h | --help)
        echo "Usage: $0 [OPTIONS]"
        echo "Set up Azure infrastructure for CyberArk Custom Provider"
        echo ""
        echo "Options:"
        echo "  --location LOCATION         Azure region (default: eastus)"
        echo "  --resource-group RG_NAME    Resource group name (default: pavel5-rg)"
        echo "  --acr-name ACR_NAME         ACR name (default: pavel5acr)"
        echo "  -h, --help                  Show this help message"
        exit 0
        ;;
    *)
        error "Unknown option: $1"
        echo "Use -h or --help for usage information"
        exit 1
        ;;
    esac
done

# Run main function
main
