targetScope = 'resourceGroup'

@description('Location for the safe metadata')
param location string = resourceGroup().location

@description('The name of the custom provider')
param customProviderName string

@description('The name of the safe to create')
@minLength(3)
@maxLength(28)
param safeName string

@description('Description of the safe')
param safeDescription string = ''

// Reference the existing custom provider
resource customProvider 'Microsoft.CustomProviders/resourceProviders@2018-09-01-preview' existing = {
  name: customProviderName
}

// Create a CyberArk safe using the custom provider
#disable-next-line BCP081
resource cyberarkSafe 'Microsoft.CustomProviders/resourceProviders/safes@2018-09-01-preview' = {
  parent: customProvider
  name: safeName
  location: location
  properties: {
    safeName: safeName
    description: safeDescription
  }
}

// Output safe information
output safeId string = cyberarkSafe.id
output safeName string = cyberarkSafe.properties.safeName
output safeDescription string = cyberarkSafe.properties.description
output provisioningState string = cyberarkSafe.properties.provisioningState
