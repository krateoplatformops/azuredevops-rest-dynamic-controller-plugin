# Krateo Azure DevOps Plugin for `rest-dynamic-controller`

A specialized web service that addresses some inconsistencies in the Azure DevOps REST API
It is designed to work with the [`rest-dynamic-controller`](https://github.com/krateoplatformops/rest-dynamic-controller/) and [`azuredevops-provider-kog`](https://github.com/krateoplatformops/azuredevops-provider-kog-chart).

## Summary

- [Summary](#summary)
- [API Endpoints](#api-endpoints)
  - [PipelinePermission](#pipelinepermission)
    - [Get PipelinePermission](#get-pipeline-permission)
- [Swagger Documentation](#swagger-documentation)
- [Azure DevOps API Reference](#azuredevops-api-reference)
- [Authentication](#authentication)

## API Endpoints

### PipelinePermission

#### Get PipelinePermission

```http
GET /api/{organization}/{project}/pipelines/pipelinepermissions/{resourceType}/{resourceId}
```

**Description**: 
Given a `ResourceType` and `ResourceId`, it returns authorized definitions for that resource.
More precisely, it returns the list of `pipelines` that have permissions to access the specified resource and the fact whether `allPipelines` have access to it.

**Why This Endpoint Exists**:
- The standard Azure DevOps REST API does not return the `allPipelines` property when said property is set to `authorized: false` on Azure DevOps (default behavior).
- This endpoint checks if the response from the Azure DevOps REST API contains the `allPipelines` property and, if not, it adds it with a value of `authorized: false`.

**Path parameters**:
- `organization` (string, required): The name of the Azure DevOps organization.
- `project` (string, required): The name of the Azure DevOps project.
- `resourceType` (string, required): The type of resource for which permissions are being requested (e.g., `repository`, `environment`, `queue`).
- `resourceId` (string, required): The ID of the resource for which permissions are being requested.

<details>
<summary><b>Response example</b></summary>

```json
{
  "resource": {
    "type":"environment",
    "id":"7"
  },
  "allPipelines":{
    "authorized":false // Adjusted field
  },
  "pipelines": [
    {
      "id":14,
      "authorized":true,
      "authorizedBy": {
        "displayName":"<REDACTED>",
        "id":"<REDACTED>",
        "uniqueName":"<REDACTED>",
        "descriptor":"<REDACTED>"
      },
      "authorizedOn":"2025-06-30T14:33:02.06Z"
    },
    {
      "id":15,
      "authorized":true,
      "authorizedBy": {
        "displayName":"<REDACTED>",
        "id":"<REDACTED>",
        "uniqueName":"<REDACTED>",
        "descriptor":"<REDACTED>"
      },
      "authorizedOn":"2025-06-30T14:33:02.06Z"
    }
  ]
}
```
</details>

## Swagger Documentation

For more detailed information about the API endpoints, please refer to the Swagger documentation available at `/swagger/index.html` endpoint of the service.

## Azure DevOps API Reference

For complete Azure DevOps REST API documentation, visit: [Azure DevOps REST API docs](https://learn.microsoft.com/en-us/rest/api/azure/devops/)

## Authentication

The plugin will forward the `Authorization` header passed in the request to this plugin to the Azure DevOps REST API.
In particular, it supports the Basic Authentication scheme, which is the default for Azure DevOps REST API.
How it works:
- You can generate a Personal Access Token (PAT) in Azure DevOps.
- Use the PAT as the password in the Basic Authentication header.
- The username can be any string (e.g., `user`), as Azure DevOps does not require a specific username for PAT authentication.

You can get more information in the README of the [`azuredevops-provider-kog`](https://github.com/krateoplatformops/azuredevops-provider-kog-chart#authentication).
