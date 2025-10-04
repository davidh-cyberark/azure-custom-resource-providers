targetScope = 'resourceGroup'

@description('Location for the safe metadata')
param location string = resourceGroup().location

@description('The name of the custom provider')
param customProviderName string = 'CyberArkProvider'

@description('Account object with all CyberArk properties')
param account object

// Reference the existing custom provider
resource customProvider 'Microsoft.CustomProviders/resourceProviders@2018-09-01-preview' existing = {
  name: customProviderName
}

// Create a CyberArk account using the custom provider

// Account object is defined in the docs as the Body parameters.
// https://docs.cyberark.com/privilege-cloud-shared-services/latest/en/content/webservices/add+account+v10.htm#Bodyparameters

#disable-next-line BCP081
resource cyberarkAccount 'Microsoft.CustomProviders/resourceProviders/accounts@2018-09-01-preview' = {
  parent: customProvider
  // '.' is a character that cannot be used in a safename; 
  // note that '.' IS VALID in an account name
  name: '${account.safeName}.${account.name}'

  location: location
  properties: account
}

// Output safe information
output account object = cyberarkAccount
output provisioningState string = cyberarkAccount.properties.provisioningState
