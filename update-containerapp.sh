#!/usr/bin/env bash

set -a
source .env
DEBUG=""
while [[ $# -gt 0 ]]; do
    case $1 in
    --debug)
        DEBUG="--debug"
        shift
        ;;
    *)
        echo "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
    esac
done
az containerapp update --name $CUSTOM_PROVIDER_APP_NAME $DEBUG \
    --resource-group $RESOURCE_GROUP \
    --image $ACR_LOGIN_SERVER/$CONTAINER_IMAGE_NAME:v$(cat custom-provider/VERSION)
