# README

## Q: When creating a custom provider for Azure Bicep, does the container app need to handle "GET /" before it will call the create action?

**Short answer:**  
Yes, your container app (the endpoint implementing the custom provider) must handle a `GET /` request before Azure will call the `PUT` (create) action for your custom resource. This is because Azure Resource Manager (ARM) performs a "validation" or "discovery" step by calling the root endpoint (`GET /`) of your custom resource provider before it attempts to create or update resources using `PUT`.

**How it works:**

- When you deploy a custom resource (e.g., `PUT /subscriptions/{subscriptionId}/resourcegroups/{resourceGroupName}/providers/Microsoft.CustomProviders/resourceProviders/{resourceProviderName}/RESOURCETYPE/{resourceName}`), ARM first calls `GET /` on your endpoint to ensure it is reachable and returns a valid response.
- If the `GET /` request fails (e.g., returns 404 or times out), ARM will not proceed to the `PUT` operation.
- Your container app should respond to `GET /` with a 200 status and a simple JSON payload (even `{}` is sufficient).

**References:**

- This pattern is documented in the <https://learn.microsoft.com/en-us/azure/azure-resource-manager/custom-providers/resource-provider-endpoints#endpoint-requirements>.
- The `GET /` is used for "discovery" and health checking by ARM.
