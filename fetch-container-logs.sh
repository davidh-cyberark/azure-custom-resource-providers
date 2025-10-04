#!/usr/bin/env bash

set -a
source .env
az containerapp logs show --name $CUSTOM_PROVIDER_APP_NAME \
    --resource-group $RESOURCE_GROUP --tail 300 $@
