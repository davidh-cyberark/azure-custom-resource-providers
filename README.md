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
NEW_SAFE_NAME="my-safe"

az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file templates/create-cyberark-safe.bicep \
  --parameters safeName="$NEW_SAFE_NAME" \
               customProviderName="$CUSTOM_PROVIDER_NAME"
```

### 5. Create Your First Account

Copy the parameters file, and add the parameters for your account.

```bash
source .env
cp templates/create-cyberark-account.parameters.json-example templates/create-cyberark-account.parameters.json

# EDIT the parameters file

az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file templates/create-cyberark-account.bicep \
  --parameters @templates/create-cyberark-account.parameters.json \
  --debug
 ```

---

## Overview

The Custom Provider enables you to:

- Create CyberArk safes using a Bicep template
- Create CyberArk accounts using a Bicep template

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
curl "https://$CONTAINER_APP_FQDN/healthex"

# Check Container App status
az containerapp show \
  --name $CUSTOM_PROVIDER_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "properties.runningStatus"
```

## Security Considerations

- Store CyberArk credentials securely using Azure Container Apps secrets
- Use managed identities for Azure resource access
- Enable HTTPS-only access for the Custom Provider
- Regularly rotate CyberArk credentials
- Monitor access logs for suspicious activity

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

This project is licensed under the Apache License - see the LICENSE file for details.
