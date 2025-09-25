#!/bin/bash

# Deploy CyberArk Custom Provider Infrastructure
# This script deploys the main infrastructure using Bicep
#
# Options:
#   --update-image          Update CONTAINER_IMAGE from ACR before deployment
#   --image-tag TAG         Use specific image tag (requires --update-image)
#   --help                  Show this help message

set -e

set -a
source color-lib.sh
source .env

MAIN_BICEP="./main.bicep"
DEBUG=""
set +a

while [[ $# -gt 0 ]]; do
    case $1 in
    --debug)
        DEBUG="--debug"
        shift
        ;;
    --help | -h)
        echo "Usage: $0 [OPTIONS]"
        echo "Deploy CyberArk Custom Provider Infrastructure"
        echo ""
        echo "Options:"
        echo "  --help | -h             Show this help message"
        echo "  --debug                 Run with debug"
        echo ""
        echo "Examples:"
        echo "  $0                      # Deploy with latest CONTAINER_IMAGE"
        exit 0
        ;;
    *)
        echo "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
    esac
done

# Validate required environment variables
required_vars=(
    "RESOURCE_GROUP"
    "ENVIRONMENT"
    "ACR_NAME"
    "CUSTOM_PROVIDER_NAME"
    "CONTAINER_IMAGE_NAME"
    "CYBERARK_ID_TENANT_URL"
    "CYBERARK_PAM_USER"
    "CYBERARK_PAM_PASSWORD"
    "CYBERARK_PCLOUD_URL"
)

echo "Validating environment variables..."
for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "❌ Error: $var is not set"
        exit 1
    fi
done
echo "✓ All required environment variables are set"

# Set default project name if not provided
PROJECT_NAME=${PROJECT_NAME:-"cyberarkcp"}
LOCATION=${LOCATION:-"eastus"}

IMAGE_NAME="${CONTAINER_IMAGE_NAME:-cyberark-custom-provider}"
IMAGE_TAG=$(az acr repository show-tags --name "$ACR_NAME" --repository "$CONTAINER_IMAGE_NAME" --orderby time_desc --top 1 --output tsv)
if [ $? -ne 0 ]; then
    error "could not find latest container tag"
    exit 1
fi
if [ -z "$IMAGE_TAG" ]; then
    error "latest container tag not found"
    exit 1
fi
CONTAINER_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"

echo ""
echo "🚀 Starting infrastructure deployment..."
echo "   Resource Group: $RESOURCE_GROUP"
echo "   Location: $LOCATION"
echo "   Project Name: $PROJECT_NAME"
echo "   ACR Name: $ACR_NAME"
echo "   Container Image: $CONTAINER_IMAGE"
echo ""

# Navigate to infra directory
cd infra

# Deploy infrastructure
echo "Deploying Bicep template..."
echo "Variables for az deployment command:"
echo "  RESOURCE_GROUP: $RESOURCE_GROUP"
echo "  MAIN_BICEP: $MAIN_BICEP"
echo "  ENVIRONMENT: $ENVIRONMENT"
echo "  LOCATION: $LOCATION"
echo "  PROJECT_NAME: $PROJECT_NAME"
echo "  ACR_NAME: $ACR_NAME"
echo "  CONTAINER_IMAGE: $CONTAINER_IMAGE"
echo "  CUSTOM_PROVIDER_NAME: $CUSTOM_PROVIDER_NAME  ## (also used for customProviderAppName)"
echo "  CYBERARK_ID_TENANT_URL: $CYBERARK_ID_TENANT_URL"
echo "  CYBERARK_PAM_USER: $CYBERARK_PAM_USER"
echo "  CYBERARK_PAM_PASSWORD: $CYBERARK_PAM_PASSWORD"
echo "  CYBERARK_PCLOUD_URL: $CYBERARK_PCLOUD_URL"
echo ""

echo ""
read -p "Do you want to continue? (y/N): " -n 1 -r
echo ""
if ! [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "...user does not want to continue, exiting."
    exit 0
fi

az deployment group create $DEBUG \
    --resource-group "$RESOURCE_GROUP" \
    --template-file $MAIN_BICEP \
    --parameters environment="$ENVIRONMENT" \
    location="$LOCATION" \
    projectName="$PROJECT_NAME" \
    acrName="$ACR_NAME" \
    acrResourceGroup="$RESOURCE_GROUP" \
    containerImage="$CONTAINER_IMAGE" \
    customProviderName="$CUSTOM_PROVIDER_NAME" \
    customProviderAppName="$CUSTOM_PROVIDER_NAME" \
    cyberarkIdTenantUrl="$CYBERARK_ID_TENANT_URL" \
    cyberarkPamUser="$CYBERARK_PAM_USER" \
    cyberarkPamPassword="$CYBERARK_PAM_PASSWORD" \
    cyberarkPCloudUrl="$CYBERARK_PCLOUD_URL"

if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Infrastructure deployment failed!"
    exit 1
fi

echo ""
echo "✅ Infrastructure deployment completed successfully!"
echo ""
echo "Next steps:"
echo "1. Verify the container app is running: "
echo "   # Determine the name of the containerapp"
echo "   az containerapp list --resource-group $RESOURCE_GROUP --query \"[].name\" --output tsv"
echo ""
echo "   # Show the container app."
echo "   # Substitute the name from the previous command in place of NAME"
echo "   az containerapp show --name NAME --resource-group $RESOURCE_GROUP"
echo "2. Test the health endpoint"
echo "3. Create your first safe using the Bicep template"
