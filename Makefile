# Makefile

BINDIR := ./bin
STATICDIR := ./static
DATADIR := ./data

FUNCTION_DIR := functions
FUNCTION_BINDIR := $(FUNCTION_DIR)/bin

.PHONY: Makefile scripts/common.makefile scripts/docs.makefile Makefile.local 
include scripts/common.makefile
include scripts/docs.makefile
include Makefile.local
export

DOCFILES := $(wildcard *.md)

GO := $(shell command -v go 2> /dev/null)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"


bootstrap: ## set Azure subscription id, create RG, create functionapp
ifndef SUBSCRIPTION_ID
	$(error SUBSCRIPTION_ID is not set)
endif
	az account set --subscription $(SUBSCRIPTION_ID)
	# Resource group
	az group create -n $(RESOURCE_GROUP) -l $(LOCATION)
	# Storage Account
	az storage account create -n $(STORAGE_ACCT) -g $(RESOURCE_GROUP) -l $(LOCATION)
	# Create Function
	az functionapp create --consumption-plan-location $(LOCATION) \
	--assign-identity '[system]' \
	--runtime custom \
	--os-type linux \
	--name $(FUNCTION_NAME) \
	--resource-group $(RESOURCE_GROUP) \
	--storage-account $(STORAGE_ACCT)

build: VERSION $(FUNCTION_BINDIR)/handler    ## build function handler

$(FUNCTION_BINDIR)/handler: cmd/handler/handler.go
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(FUNCTION_BINDIR)/handler $(LDFLAGS) cmd/handler/handler.go

publish: build  ## publish functionapp to Azure
	(cd $(FUNCTION_DIR) && func azure functionapp publish $(FUNCTION_NAME))

clean::
	rm -f $(FUNCTION_BINDIR)/handler

run: build
	(cd functions && func start)

run-debug: build
	(cd functions && dlv debug ../cmd/handler/handler.go --headless --listen=:2345 --accept-multiclient --continue)
	# also add as needed: --accept-multiclient --continue)

