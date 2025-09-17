#!/bin/bash

# Deploy CyberArk Custom Provider Infrastructure
# This script deploys the main infrastructure using Bicep

set -e

# Source environment variables
if [ -f ".env" ]; then
    set -a
    source .env
    set +a
    echo "✓ Environment variables loaded from .env"
else
    echo "❌ Error: .env file not found"
    echo "Please run ./env-setup.sh first to create the environment file"
    exit 1
fi

# Validate required environment variables
required_vars=(
    "RESOURCE_GROUP"
    "ENVIRONMENT"
    "ACR_NAME"
    "CONTAINER_IMAGE"
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
az deployment group create \
    --resource-group "$RESOURCE_GROUP" \
    --template-file main.bicep \
    --parameters environment="$ENVIRONMENT" \
    location="$LOCATION" \
    projectName="$PROJECT_NAME" \
    acrName="$ACR_NAME" \
    acrResourceGroup="$RESOURCE_GROUP" \
    containerImage="$CONTAINER_IMAGE" \
    cyberarkIdTenantUrl="$CYBERARK_ID_TENANT_URL" \
    cyberarkPamUser="$CYBERARK_PAM_USER" \
    cyberarkPamPassword="$CYBERARK_PAM_PASSWORD" \
    cyberarkPCloudUrl="$CYBERARK_PCLOUD_URL"

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Infrastructure deployment completed successfully!"
    echo ""
    echo "Next steps:"
    echo "1. Verify the container app is running: az containerapp show --name $PROJECT_NAME-custom-provider-* --resource-group $RESOURCE_GROUP"
    echo "2. Test the health endpoint"
    echo "3. Create your first safe using the Bicep template"
else
    echo ""
    echo "❌ Infrastructure deployment failed!"
    exit 1
fi
