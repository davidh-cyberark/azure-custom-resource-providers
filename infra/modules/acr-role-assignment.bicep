targetScope = 'resourceGroup'

@description('The name of the Azure Container Registry')
param acrName string

@description('The principal ID of the managed identity')
param managedIdentityPrincipalId string

// Reference existing ACR
resource acr 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' existing = {
  name: acrName
}

// Grant ACR Pull permissions to managed identity
resource acrPullRoleAssignment 'Microsoft.Authorization/roleAssignments@2022-04-01' = {
  scope: acr
  name: guid(acr.id, managedIdentityPrincipalId, 'acrpull')
  properties: {
    roleDefinitionId: subscriptionResourceId(
      'Microsoft.Authorization/roleDefinitions',
      '7f951dda-4ed3-4680-a7ca-43fe172d538d'
    ) // AcrPull
    principalId: managedIdentityPrincipalId
    principalType: 'ServicePrincipal'
  }
}

output roleAssignmentId string = acrPullRoleAssignment.id
