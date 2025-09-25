targetScope = 'resourceGroup'

@description('The location for all resources')
param location string = resourceGroup().location

@description('The name of the custom provider')
param customProviderName string

@description('Environment name (dev, staging, prod)')
param environment string = 'dev'

@description('The name of the project')
param projectName string = 'cyberarkcp'

@description('The custom provider app name')
param customProviderAppName string

@description('The name of the Azure Container Registry')
param acrName string

@description('The resource group containing the ACR')
param acrResourceGroup string = resourceGroup().name

@description('Container image name and tag')
param containerImage string = 'cyberark-custom-provider:latest'

@description('CyberArk Identity Tenant URL')
@secure()
param cyberarkIdTenantUrl string

@description('CyberArk Privilege Cloud URL')
@secure()
param cyberarkPCloudUrl string

@description('CyberArk PAM User')
@secure()
param cyberarkPamUser string

@description('CyberArk PAM Password')
@secure()
param cyberarkPamPassword string

// Generate unique names using resource token
var resourceToken = toLower(take(uniqueString(subscription().id, resourceGroup().id, location), 8))
var tags = {
  'azd-env-name': environment
  project: projectName
  environment: environment
}

// Reference existing ACR
resource acr 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' existing = {
  name: acrName
  scope: resourceGroup(acrResourceGroup)
}

// Create Log Analytics Workspace
resource logAnalytics 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {
  name: '${projectName}-logs-${resourceToken}'
  location: location
  tags: tags
  properties: {
    sku: {
      name: 'PerGB2018'
    }
    retentionInDays: 30
  }
}

// Create user-assigned managed identity
resource managedIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {
  name: '${projectName}-identity-${resourceToken}'
  location: location
  tags: tags
}

// Grant ACR Pull permissions to managed identity
module acrRoleAssignment 'modules/acr-role-assignment.bicep' = {
  name: 'acr-role-assignment'
  scope: resourceGroup(acrResourceGroup)
  params: {
    acrName: acrName
    managedIdentityPrincipalId: managedIdentity.properties.principalId
  }
}

// Create Container Apps Environment
resource containerAppsEnvironment 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: '${projectName}-env-${resourceToken}'
  location: location
  tags: tags
  properties: {
    appLogsConfiguration: {
      destination: 'log-analytics'
      logAnalyticsConfiguration: {
        customerId: logAnalytics.properties.customerId
        sharedKey: logAnalytics.listKeys().primarySharedKey
      }
    }
  }
}

// Create Container App for Custom Provider
resource customProviderApp 'Microsoft.App/containerApps@2023-05-01' = {
  name: customProviderAppName
  location: location
  tags: tags
  identity: {
    type: 'UserAssigned'
    userAssignedIdentities: {
      '${managedIdentity.id}': {}
    }
  }
  properties: {
    managedEnvironmentId: containerAppsEnvironment.id
    configuration: {
      ingress: {
        external: true
        targetPort: 8080
        allowInsecure: false
        traffic: [
          {
            weight: 100
            latestRevision: true
          }
        ]
        corsPolicy: {
          allowedOrigins: ['*']
          allowedMethods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS']
          allowedHeaders: ['*']
          allowCredentials: false
        }
      }
      registries: [
        {
          server: acr.properties.loginServer
          identity: managedIdentity.id
        }
      ]
      secrets: [
        {
          name: 'cyberark-id-tenant-url'
          value: cyberarkIdTenantUrl
        }
        {
          name: 'cyberark-pam-user'
          value: cyberarkPamUser
        }
        {
          name: 'cyberark-pam-password'
          value: cyberarkPamPassword
        }
        {
          name: 'cyberark-pcloud-url'
          value: cyberarkPCloudUrl
        }
      ]
    }
    template: {
      containers: [
        {
          image: '${acr.properties.loginServer}/${containerImage}'
          name: 'cyberark-custom-provider'
          env: [
            {
              name: 'PORT'
              value: '8080'
            }
            {
              name: 'IDTENANTURL'
              secretRef: 'cyberark-id-tenant-url'
            }
            {
              name: 'PAMUSER'
              secretRef: 'cyberark-pam-user'
            }
            {
              name: 'PAMPASS'
              secretRef: 'cyberark-pam-password'
            }
            {
              name: 'PCLOUDURL'
              secretRef: 'cyberark-pcloud-url'
            }
          ]
          resources: {
            cpu: json('0.5')
            memory: '1.0Gi'
          }
          probes: [
            {
              type: 'Liveness'
              httpGet: {
                path: '/health'
                port: 8080
              }
              initialDelaySeconds: 30
              periodSeconds: 30
            }
            {
              type: 'Readiness'
              httpGet: {
                path: '/health'
                port: 8080
              }
              initialDelaySeconds: 5
              periodSeconds: 10
            }
          ]
        }
      ]
      scale: {
        minReplicas: 1
        maxReplicas: 3
        rules: [
          {
            name: 'http-scaling'
            http: {
              metadata: {
                concurrentRequests: '10'
              }
            }
          }
        ]
      }
    }
  }
  dependsOn: [
    acrRoleAssignment
  ]
}

// Create Azure Custom Provider
resource customProvider 'Microsoft.CustomProviders/resourceProviders@2018-09-01-preview' = {
  name: customProviderName
  location: location
  tags: tags
  properties: {
    resourceTypes: [
      {
        name: 'safes'
        routingType: 'Proxy'
        endpoint: 'https://${customProviderApp.properties.configuration.ingress.fqdn}'
      }
    ]
  }
}

// Outputs
output containerAppUrl string = 'https://${customProviderApp.properties.configuration.ingress.fqdn}'
output customProviderName string = customProvider.name
output customProviderResourceId string = customProvider.id
output logAnalyticsWorkspaceId string = logAnalytics.id
output managedIdentityId string = managedIdentity.id
output containerAppsEnvironmentId string = containerAppsEnvironment.id
