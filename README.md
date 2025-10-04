# CyberArk Custom Provider for Azure

This project implements an Azure Custom Provider that integrates with CyberArk Privileged Access Manager (PAM) to manage safes through Azure Resource Manager templates and Bicep.

## Overview

The Azure `CyberArkProvider`, is a custom provider enables you to:

- Create CyberArk safes using a Bicep template
- Create CyberArk accounts using a Bicep template

## Prerequisites

- Azure CLI installed and configured
- Azure subscription with appropriate permissions
- Access to CyberArk Privileged Cloud environment
  - User creds with permissions to create safes and add accounts
- Go 1.22+ installed
- Docker installed

## Quick Start

Get up and running in minutes with these steps.

### 1. Configure

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

### 3. Setup Azure

```bash
# Setup Azure infrastructure (resource group and ACR)
./setup-azure-infrastructure.sh
```

### 4. Deploy to Azure

```bash
# Build Docker image and push to ACR using the build script
./build-docker.sh --push

# Deploy infrastructure
./deploy-infrastructure.sh
```

### 5. Create Your First Safe

```bash
# Source environment variables (if not already done)
source .env

# Validate your environment first
./validate-environment.sh

# Use the provided Bicep template
NEW_SAFE_NAME="my-safe"
source .env && az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file templates/create-cyberark-safe.bicep \
  --parameters customProviderName="CyberArkProvider" \
               safeName="$NEW_SAFE_NAME"               
```

### 6. Create Your First Account

Copy the parameters file, and add the parameters for your account.

Edit the parameters file set the "account" property.  See [Account Definition Example](#account-definition-example).

```bash
cp templates/create-cyberark-account.parameters.json-example templates/create-cyberark-account.parameters.json

# Edit parameters json file
vi templates/create-cyberark-account.parameters.json

source .env && az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file templates/create-cyberark-account.bicep \
  --parameters @templates/create-cyberark-account.parameters.json
 ```

#### Account Definition Example

Refer to the add account documentation for the full structure of the "account" property.  Look at the example below, the body shall be rendered as JSON object as the `account.value`.

 REF: [CyberArk Privilege Cloud Add Account Doc](https://docs.cyberark.com/privilege-cloud-shared-services/latest/en/content/webservices/add%20account%20v10.htm#Bodyparameters)

Example definition that can be used in `create-cyberark-account.parameters.json` file.

```json
{
    "$schema": "https://schema.management.azure.com/schemas/2019-04-01/deploymentParameters.json#",
    "contentVersion": "1.0.0.0",
    "parameters": {
        "customProviderName": {
            "value": "CyberArkProvider"
        },
        "account": {
            "value": {
                "safeName": "my-example-safe1",
                "name": "my-example-account1",
                "address": "10.0.0.1",
                "userName": "pavel20i-user1",
                "platformId": "UnixSSH",
                "secretType": "key",
                "secret": "my-example-password1",
                "platformAccountProperties": {
                    "key1": "value1",
                    "key2": "value2"
                },
                "secretManagement": {
                    "manualManagementReason": "Reason for disabling automatic secret management.",
                    "automaticManagementEnabled": true
                },
                "remoteMachinesAccess": {
                    "remoteMachines": "List of remote machines, separated by semicolons.",
                    "accessRestrictedToRemoteMachines": true
                }
            }
        }
    }
}
```

## Monitoring and Troubleshooting

### Troubleshooting

- Check Container App logs for detailed error messages, `./fetch-container-logs.sh`
- Use the health endpoint to verify service status
- Monitor Azure Resource Manager deployment operations
- Review CyberArk audit logs for API call details

### Health Checks

```bash
# Source environment variables
source .env

# Store the container app FQDN in a variable
CONTAINER_APP_FQDN=$(az containerapp show \
  --name $CUSTOM_PROVIDER_APP_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "properties.configuration.ingress.fqdn" \
  --output tsv)

echo "Container App FQDN: $CONTAINER_APP_FQDN"

# Test the health endpoint
curl "https://$CONTAINER_APP_FQDN/health"
curl "https://$CONTAINER_APP_FQDN/healthex"

# Check Container App status
az containerapp show \
  --name $CUSTOM_PROVIDER_APP_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "properties.runningStatus"
```

## License

This project is licensed under the Apache License - see the LICENSE file for details.
