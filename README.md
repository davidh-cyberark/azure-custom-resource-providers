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
# Source environment variables
source .env

# Setup Azure infrastructure (resource group and ACR)
./setup-azure-infrastructure.sh

# Build and push the application BEFORE infrastructure deployment
cd custom-provider/
make clean && make build

# Build Docker image and push to ACR using the build script
cd ..
./build-docker.sh --push

# Deploy infrastructure (after image is pushed)
./deploy-infrastructure.sh
```

### 4. Create Your First Safe

```bash
# Source environment variables (if not already done)
source .env

# Validate your environment first
./validate-environment.sh

# Use the provided Bicep template
az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file templates/create-cyberark-safe.bicep \
  --parameters safeName="my-first-safe" \
               customProviderName="$CUSTOM_PROVIDER_NAME"
```

That's it! Your CyberArk Custom Provider is now running in Azure. 🎉

### Environment Management Scripts

This project includes several helper scripts to streamline your workflow:

- **`./env-setup.sh`** - Interactive environment configuration wizard
- **`./deploy-infrastructure.sh`** - Deploy Azure infrastructure using Bicep
- **`./validate-environment.sh`** - Validates configuration and connectivity  
- **`./rebuild-and-run.sh`** - Local development and testing

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

```bash
# Azure Configuration
LOCATION="eastus"
RESOURCE_GROUP="your-rg-name"
ENVIRONMENT="dev"

# Custom Provider Configuration
CONTAINER_IMAGE="your-custom-provider:latest"
CUSTOM_PROVIDER_NAME="your-custom-provider"

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

# Update version (optional)
echo "1.0.0" > VERSION

# Build the application
make clean && make build

# Build and push Docker image to ACR
cd ..
./build-docker.sh --push
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

# Check application health
curl https://your-custom-provider-fqdn/health

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

# Build and push new version
cd custom-provider/
echo "1.0.1" > VERSION
cd ..
./build-docker.sh --push

# Update Container App with new image
VERSION=$(cat custom-provider/VERSION)
az containerapp update \
  --name $CUSTOM_PROVIDER_NAME \
  --resource-group $RESOURCE_GROUP \
  --image $ACR_LOGIN_SERVER/cyberark-custom-provider:v$VERSION
```

### 4. Verify Deployment

```bash
# Check health and version
curl https://your-custom-provider-fqdn/health
```

## Security Considerations

- Store CyberArk credentials securely using Azure Container Apps secrets
- Use managed identities for Azure resource access
- Enable HTTPS-only access for the Custom Provider
- Regularly rotate CyberArk credentials
- Monitor access logs for suspicious activity

## Troubleshooting

### Common Issues

1. **Container App not starting**
   - Check Container App logs for startup errors
   - Verify CyberArk credentials are correctly configured
   - Ensure image is properly pushed to ACR

2. **Safe creation fails**
   - Verify CyberArk credentials have appropriate permissions
   - Check network connectivity to CyberArk endpoints
   - Review Container App logs for detailed error messages

3. **Custom Provider not responding**
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
