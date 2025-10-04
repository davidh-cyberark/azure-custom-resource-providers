#!/usr/bin/env bash

source .env

# Store the container app FQDN in a variable
CONTAINER_APP_FQDN=$(az containerapp show \
    --name $CUSTOM_PROVIDER_APP_NAME \
    --resource-group $RESOURCE_GROUP \
    --query "properties.configuration.ingress.fqdn" \
    --output tsv)

echo "Container App FQDN: $CONTAINER_APP_FQDN"

# Test the health endpoint
curl "https://$CONTAINER_APP_FQDN/healthex"

# Check Container App status
az containerapp show \
    --name $CUSTOM_PROVIDER_APP_NAME \
    --resource-group $RESOURCE_GROUP \
    --query "properties.runningStatus"
