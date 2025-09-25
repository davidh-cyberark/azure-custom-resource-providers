# CyberArk Custom Provider for Azure

This project implements an Azure Custom Provider that integrates with CyberArk Privileged Access Manager (PAM) to manage safes through Azure Resource Manager templates and Bicep.

## Quick Start

Get up and running in minutes with our guided setup:

### 1. Clone and Configure

```bash
# Clone the repository (or navigate to your existing clone)
cd cyberark-custom-provider

# Run the interactive setup script
./env-setup.sh

# Follow the prompts to configure your environment
# The script will create a .env file with your settings
```

### 2. Local Testing

```bash
# Test the application locally
./rebuild-and-run.sh

# Verify it's working
curl http://localhost:8080/health

# Stop local testing
./rebuild-and-run.sh --stop
```

### 3. Quick Deploy to Azure

```bash
# Setup Azure infrastructure (resource group and ACR)
./setup-azure-infrastructure.sh

# Build Docker image and push to ACR using the build script
./build-docker.sh         # Build the image
./build-docker.sh --push  # Push the image to ACR

# Deploy infrastructure
./deploy-infrastructure.sh
```

### 4. Create Your First Safe

```bash
# Source environment variables (if not already done)
source .env

# Validate your environment first
./validate-environment.sh

# Use the provided Bicep template
NEW_SAFE_NAME="my-test-safe"
az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file templates/create-cyberark-safe.bicep \
  --parameters safeName="$NEW_SAFE_NAME" \
               customProviderName="$CUSTOM_PROVIDER_NAME"
```

That's it! Your CyberArk Custom Provider is now running in Azure. 🎉

### Environment Management Scripts

This project includes several helper scripts to streamline your workflow:

- **`./env-setup.sh`** - Interactive environment configuration wizard
- **`./deploy-infrastructure.sh`** - Deploy Azure infrastructure using Bicep
- **`./validate-environment.sh`** - Validates configuration and connectivity  
- **`./rebuild-and-run.sh`** - Local development and testing
- **`./get-latest-image.sh`** - Manage container image versions from ACR
- **`./get-custom-provider-name.sh`** - Discover and update dynamic custom provider names
- **`./update-resources.sh`** - Unified management for both images and provider names

Run `./validate-environment.sh --verbose` for detailed environment diagnostics.

---

## Overview

The Custom Provider enables you to:

- Create CyberArk safes using a Bicep template

## Architecture

- **Go Application**: Custom provider implementation using CyberArk PAM SDK
- **Azure Container Apps**: Hosting platform for the custom provider
- **Azure Container Registry**: Docker image storage
- **Azure Custom Provider**: Resource type registration and routing

## Prerequisites

- Azure CLI installed and configured
- Azure subscription with appropriate permissions
- Access to CyberArk Privileged Cloud environment
  - User creds with permissions to create safes and add accounts
- Go 1.22+ installed
- Docker installed

## Configuration

### Environment Variables

Create a `.env` file with the following configuration:

TODO: Update this section with .env.template vars

```bash
# Azure Configuration
LOCATION="eastus"
RESOURCE_GROUP="your-rg-name"
ENVIRONMENT="dev"

# Custom Provider Configuration
CONTAINER_IMAGE="your-custom-provider:latest"  # Managed by get-latest-image.sh
CUSTOM_PROVIDER_NAME="your-custom-provider"    # Managed by get-custom-provider-name.sh

# Azure Container Registry
ACR_NAME="youracr"
ACR_LOGIN_SERVER="youracr.azurecr.io"

# CyberArk Configuration
CYBERARK_ID_TENANT_URL="https://your-tenant.id.cyberark.cloud"
CYBERARK_PAM_PASSWORD="your-pam-password"
CYBERARK_PAM_USER="your-pam-user@cyberark.cloud.tenant"
CYBERARK_PCLOUD_URL="https://your-tenant.privilegecloud.cyberark.cloud"
```

## Build and Deployment

### Container Image Management

This project includes an automated system for managing container images in Azure Container Registry (ACR). The `get-latest-image.sh` script can automatically retrieve and update container image references.

#### Quick Start - Image Management

```bash
# Update .env with latest image from ACR
./get-latest-image.sh

# Update to specific version
./get-latest-image.sh --tag v1.2.3

# List all available images
./get-latest-image.sh --list

# Preview changes without updating
./get-latest-image.sh --dry-run
```

### Custom Provider Name Management

Azure dynamically generates custom provider names during deployment, making them difficult to predict. The project includes automated tools to discover and manage these dynamic names.

#### Quick Start - Custom Provider Management

```bash
# Update .env with current custom provider name from Azure
./get-custom-provider-name.sh

# List all custom providers in your resource group
./get-custom-provider-name.sh --list

# Select provider matching a pattern
./get-custom-provider-name.sh --pattern "*prod*"

# Preview changes without updating
./get-custom-provider-name.sh --dry-run
```

### Unified Resource Management

Use the helper script to manage both image and provider names:

```bash
# Update both image and custom provider name
./update-resources.sh both

# Update only the image to latest version
./update-resources.sh image

# Update only the custom provider name
./update-resources.sh provider

# Preview all changes
./update-resources.sh both --dry-run
```

#### Enhanced Build and Deploy Workflow

```bash
# Build, push to ACR, and update .env in one command
./build-docker.sh --push-and-update

# Deploy with latest image from ACR
./deploy-infrastructure.sh --update-image

# Deploy with image update and capture provider name
./deploy-infrastructure.sh --update-image --update-provider-name

# Deploy specific version with provider name capture
./deploy-infrastructure.sh --image-tag v1.2.3 --update-provider-name

# Validate with refreshed provider name
./validate-environment.sh --refresh-provider-name
```

For detailed documentation on dynamic resource management features, see [Dynamic Image Management Guide](docs/Dynamic-Image-Management.md).

### 1. Local Development

#### Build and Run Locally

```bash
# Make the rebuild script executable
chmod +x rebuild-and-run.sh

# Build and run the container locally
./rebuild-and-run.sh

# Check health endpoint
curl http://localhost:8080/health
```

#### View Local Logs

```bash
# View live logs
docker logs -f cyberark-local-test

# Stop local container
./rebuild-and-run.sh --stop
```

### 2. Azure Infrastructure Deployment

#### Deploy Base Infrastructure

```bash
# Source environment variables
source .env

# Deploy Azure resources (Container Apps, ACR, Custom Provider)
./deploy-infrastructure.sh
```

### 3. Application Deployment to Azure

#### Build and Push Container Image

```bash
# Source environment variables
source .env

# Navigate to custom provider directory
cd custom-provider/

# Build the application
make clean && make build

# Build and push Docker image to ACR
cd ..
./build-docker.sh --push-and-update
cd custom-provider/
```

#### Update Container App

```bash
# Source environment variables
source .env

# Update Container App with new image
VERSION=$(cat VERSION)
az containerapp update \
  --name $CUSTOM_PROVIDER_NAME \
  --resource-group $RESOURCE_GROUP \
  --image $ACR_LOGIN_SERVER/cyberark-custom-provider:v$VERSION
```

#### Verify Deployment

```bash
# Check health endpoint
curl https://your-custom-provider-fqdn/health

# Expected response:
{
  "build_date": "2025-09-12 16:23:47",
  "publicIP": "xxx.xxx.xxx.xxx",
  "service": "cyberark-custom-provider",
  "status": "healthy",
  "version": "1.0.0"
}
```

## Usage

### Creating CyberArk Safes with Bicep

#### 1. Using the Bicep Template

```bash
# Source environment variables
source .env

# Deploy safe using Bicep template
az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file templates/create-cyberark-safe.bicep \
  --parameters safeName="my-application-safe" \
               safeDescription="Safe for my application secrets" \
               customProviderName="$CUSTOM_PROVIDER_NAME"
```

#### 2. Custom Bicep Template

You can also create your own Bicep template:

```bicep
// my-safe-template.bicep
@description('Name of the CyberArk safe to create')
param safeName string

@description('Description for the CyberArk safe')
param safeDescription string = 'Safe created via Azure Custom Provider'

@description('Name of the Custom Provider resource')
param customProviderName string

// Reference to existing Custom Provider
resource customProvider 'Microsoft.CustomProviders/resourceProviders@2018-09-01-preview' existing = {
  name: customProviderName
}

// Create CyberArk Safe
resource cyberarkSafe 'Microsoft.CustomProviders/resourceProviders/cyberarkSafes@2018-09-01-preview' = {
  name: '${customProvider.name}/${safeName}'
  properties: {
    safeName: safeName
    description: safeDescription
    location: resourceGroup().location
  }
}

// Output safe information
output safeId string = cyberarkSafe.properties.safeID
output safeName string = cyberarkSafe.properties.safeName
output provisioningState string = cyberarkSafe.properties.provisioningState
```

Deploy your custom template:

```bash
# Source environment variables
source .env

az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file my-safe-template.bicep \
  --parameters safeName="your-safe-name" \
               safeDescription="Your safe description" \
               customProviderName="$CUSTOM_PROVIDER_NAME"
```

### Direct API Usage

You can also call the Custom Provider API directly:

#### Create Safe (PUT Request)

```bash
curl -X PUT \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "safeName": "api-test-safe",
      "description": "Safe created via direct API call",
      "location": "eastus"
    }
  }' \
  "https://your-custom-provider-fqdn/subscriptions/your-subscription-id/resourcegroups/your-rg-name/providers/Microsoft.CustomProviders/resourceProviders/your-custom-provider-name/cyberarkSafes/api-test-safe"
```

#### Create Safe (POST Action)

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "properties": {
      "safeName": "action-test-safe",
      "description": "Safe created via action endpoint"
    }
  }' \
  "https://your-custom-provider-fqdn/subscriptions/your-subscription-id/resourcegroups/your-rg-name/providers/Microsoft.CustomProviders/resourceProviders/your-custom-provider-name/createSafe"
```

## Monitoring and Troubleshooting

### Health Checks

```bash
# Source environment variables
source .env

# Store the container app FQDN in a variable
CONTAINER_APP_FQDN=$(az containerapp show \
  --name $CUSTOM_PROVIDER_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "properties.configuration.ingress.fqdn" \
  --output tsv)

echo "Container App FQDN: $CONTAINER_APP_FQDN"

# Test the health endpoint
curl "https://$CONTAINER_APP_FQDN/health"

# Check Container App status
az containerapp show \
  --name $CUSTOM_PROVIDER_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "properties.runningStatus"
```

### Logs

```bash
# Source environment variables
source .env

# View Container App logs
az containerapp logs show \
  --name $CUSTOM_PROVIDER_NAME \
  --resource-group $RESOURCE_GROUP \
  --tail 50

# Follow logs in real-time
az containerapp logs show \
  --name $CUSTOM_PROVIDER_NAME \
  --resource-group $RESOURCE_GROUP \
  --follow
```

### Version Verification

```bash
# Compare local vs deployed versions
echo "Local version:"
curl -s http://localhost:8080/health | jq '.version, .build_date'

echo "Azure version:"
curl -s https://your-custom-provider-fqdn/health | jq '.version, .build_date'
```

## Development Workflow

### 1. Make Code Changes

- Edit files in `custom-provider/` directory
- Update version in `VERSION` file if needed

### 2. Test Locally

```bash
# Rebuild and test locally
./rebuild-and-run.sh

# Test endpoints
curl http://localhost:8080/health
```

### 3. Deploy to Azure

```bash
# Source environment variables
source .env

# Build and push new version with automatic .env update
cd custom-provider/
echo "1.0.1" > VERSION
cd ..
./build-docker.sh --push-and-update

# Deploy with resource name management
./deploy-infrastructure.sh --update-provider-name
```

### 4. Verify Deployment

```bash
# Check health and version
curl https://your-custom-provider-fqdn/health

# Validate environment with refreshed values
./validate-environment.sh --refresh-provider-name
```

### Alternative Workflow with Unified Helper

```bash
# Update all resources and deploy
./update-resources.sh both
./deploy-infrastructure.sh --update-provider-name

# Or use a complete workflow
./update-resources.sh both --dry-run  # Preview changes
./update-resources.sh both           # Apply changes
./deploy-infrastructure.sh --update-provider-name
./validate-environment.sh --refresh-provider-name
```

## Security Considerations

- Store CyberArk credentials securely using Azure Container Apps secrets
- Use managed identities for Azure resource access
- Enable HTTPS-only access for the Custom Provider
- Regularly rotate CyberArk credentials
- Monitor access logs for suspicious activity

## Troubleshooting

### Quick Diagnostics

When experiencing deployment issues, follow this diagnostic sequence:

```bash
# 1. Verify container app is running
az containerapp list --resource-group $RESOURCE_GROUP --query "[].{Name:name, Status:properties.runningStatus, FQDN:properties.configuration.ingress.fqdn}" -o table

# 2. Test the health endpoint (not root path)
CONTAINER_APP_FQDN=$(az containerapp show --name $CUSTOM_PROVIDER_NAME --resource-group $RESOURCE_GROUP --query "properties.configuration.ingress.fqdn" -o tsv)
curl -v "https://$CONTAINER_APP_FQDN/health"

# 3. Check custom provider configuration
az resource show --resource-group $RESOURCE_GROUP --resource-type "Microsoft.CustomProviders/resourceProviders" --name $CUSTOM_PROVIDER_NAME --query "properties" -o json

# 4. Update custom provider name if mismatched
./get-custom-provider-name.sh
source .env

# 5. Test direct API call to custom provider (corrected URL)
TEST_SAFE_NAME="test-safe"
CONTAINER_APP_FQDN=$(az containerapp show --name $CUSTOM_PROVIDER_NAME --resource-group $RESOURCE_GROUP --query "properties.configuration.ingress.fqdn" -o tsv)
curl -v -X PUT \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $(az account get-access-token --query accessToken -o tsv)" \
  -d '{
    "properties": {
      "safeName": "'$TEST_SAFE_NAME'",
      "description": "Test safe for troubleshooting"
    }
  }' \
  "https://$CONTAINER_APP_FQDN/subscriptions/$(az account show --query id -o tsv)/resourcegroups/$RESOURCE_GROUP/providers/Microsoft.CustomProviders/resourceProviders/$CUSTOM_PROVIDER_NAME/cyberarkSafes/$TEST_SAFE_NAME"
```

**Important Note**: The root path `/` will return `{"error":{"code":"EndpointNotFound","message":"Endpoint / not found"}}`. This is **expected behavior** - the custom provider only responds to specific API endpoints like `/health` and the custom resource paths.

### Common Issues

1. **Container App not starting**
   - Check Container App logs for startup errors
   - Verify CyberArk credentials are correctly configured
   - Ensure image is properly pushed to ACR

2. **Safe creation fails with "EndpointNotFound"**
   - **Root cause**: Testing wrong endpoint (like `/`) instead of `/health`
   - **Solution**: Use `/health` endpoint to verify app is running
   - **Expected**: Root path `/` should return "Endpoint / not found" error

3. **Custom Provider name mismatch**
   - **Root cause**: `$CUSTOM_PROVIDER_NAME` doesn't match actual deployed resource
   - **Solution**: Run `./get-custom-provider-name.sh` to sync the name
   - **Verify**: Check that bicep parameter matches Azure resource name

4. **Safe creation fails with CyberArk errors**
   - Verify CyberArk credentials have appropriate permissions
   - Check network connectivity to CyberArk endpoints
   - Review Container App logs for detailed error messages

5. **Custom Provider not responding to requests**
   - Check Container App health status
   - Verify ingress configuration
   - Test health endpoint directly

### Getting Help

- Check Container App logs for detailed error messages
- Use the health endpoint to verify service status
- Monitor Azure Resource Manager deployment operations
- Review CyberArk audit logs for API call details

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test locally using `./rebuild-and-run.sh`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
