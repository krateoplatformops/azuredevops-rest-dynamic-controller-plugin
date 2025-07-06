# Krateo Azure DevOps Plugin for `rest-dynamic-controller`

A specialized web service that addresses some inconsistencies in the Azure DevOps REST API
It is designed to work with the [`rest-dynamic-controller`](https://github.com/krateoplatformops/rest-dynamic-controller/) and [`azuredevops-provider-kog`](https://github.com/krateoplatformops/azuredevops-provider-kog-chart).

## Summary

- [Summary](#summary)
- [API Endpoints](#api-endpoints)
  - [PipelinePermission](#pipelinepermission)
    - [Get PipelinePermission](#get-pipeline-permission)
  - [GitRepository](#gitrepository)
    - [Create GitRepository](#create-gitrepository)
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

<details>
<summary><b>Request</b></summary>

**Path parameters**:
- `organization` (string, required): The name of the Azure DevOps organization.
- `project` (string, required): The name of the Azure DevOps project.
- `resourceType` (string, required): The type of resource for which permissions are being requested (e.g., `repository`, `environment`, `queue`).
- `resourceId` (string, required): The ID of the resource for which permissions are being requested.

**Query parameters**:
- `api-version` (string, required): The version of the Azure DevOps REST API to use. For example, `7.2-preview.2`.
</details>

<details>
<summary><b>Response</b></summary>

**Response status codes**:
- `200 OK`: The request was successful.
- `401 Unauthorized`: The request is not authorized. Ensure that the `Authorization` header is set correctly.
- `500 Internal Server Error`: An unexpected error occurred while processing the request.

**Response body example**:
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

### GitRepository

#### Create GitRepository

```http
POST /api/{organization}/{projectId}/git/repositories
```

**Description**:
This endpoint creates a new GitRepository in the specified Azure DevOps project.
It allows you to specify the `initialize` field to indicate whether the repository should be initialized with a first commit. (Note: you cannot initialize a repository with a first commit if you are forking a repository).
It allows you to specify the `defaultBranch` field to set the default branch of the repository.

**Why This Endpoint Exists**:
- The standard Azure DevOps REST API has two different request body schemas for creating (`POST`) and updating (`PATCH`) Git repositories. In particular, the field `defaultBranch` is only available in the `PATCH` request body.
- This endpoint allows you to create a Git repository with the `defaultBranch` field, which is not supported in the standard Azure DevOps REST API for the `POST` request body. Practially performing a `PATCH` operation on the repository immediately after creation.
- Moreover, it allows you to initialize the repository with a first commit by setting the `initialize` field to `true`.
- In addition, it performs additional validations related to branch existence (for forks) and repository initialization. For instance auto-initialization of the repository with a first commit on the `defaultBranch` branch when the `initialize` field is set to `false` or omitted but the `defaultBranch` field is set to a branch name.
- Another additional validation is that it checks if the `sourceRef` branch exists in the parent repository when forking a repository. If it does not exist, it returns a `400 Bad Request` error.

<details>
<summary><b>Request</b></summary>

**Path parameters**:
- `organization` (string, required): The name of the Azure DevOps organization.
- `projectId` (string, required): The ID or name of the Azure DevOps project.

**Query parameters**:
- `api-version` (string, required): The version of the Azure DevOps REST API to use. For example, `7.2-preview.2`.
- `sourceRef` (string, optional): The source reference for the repository. This is typically a branch name (e.g., `refs/heads/main`).

**Request body example**:
```json
{
  "name": "string",
  "defaultBranch": "string",    // Adjusted field
  "initialize": true,           // Adjusted field

  // From here, optional, fork-related fields:
  "parentRepository": {
    "id": "4b8c6f64-5717-4562-b3fc-2c963f66afa6",
    "project": {
      "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    }
  },
  "project": {
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  }
}
```
</details>

<details>
<summary><b>Response</b></summary>

**Response status codes**:
- `201 Created`: The GitRrepository was successfully created.
- `202 Accepted`: The GitRrepository was successfully created but `defaultBranch` specified in the request body does not exist in the repository.
- `400 Bad Request`: The request body is invalid, the `sourceRef` branch does not exist in the parent repository or other validation errors occurred.
- `401 Unauthorized`: The request is not authorized. Ensure that the `Authorization` header is set correctly.
- `500 Internal Server Error`: An unexpected error occurred while processing the request.

**Response body example**:
```json
{
  "_links": {
    "links": {
      "additionalProp1": {},
      "additionalProp2": {},
      "additionalProp3": {}
    }
  },
  "creationDate": "2025-07-06T12:28:03.454Z",
  "defaultBranch": "string",
  "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "isDisabled": true,
  "isFork": true,
  "isInMaintenance": true,
  "name": "string",
  "parentRepository": {
    "collection": {
      "avatarUrl": "string",
      "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "name": "string",
      "url": "string"
    },
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "isFork": true,
    "name": "string",
    "project": {
      "abbreviation": "string",
      "defaultTeamImageUrl": "string",
      "description": "string",
      "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
      "lastUpdateTime": "2025-07-06T12:28:03.454Z",
      "name": "string",
      "revision": 0,
      "state": "deleting",
      "url": "string",
      "visibility": "private"
    },
    "remoteUrl": "string",
    "sshUrl": "string",
    "url": "string"
  },
  "project": {
    "abbreviation": "string",
    "defaultTeamImageUrl": "string",
    "description": "string",
    "id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
    "lastUpdateTime": "2025-07-06T12:28:03.454Z",
    "name": "string",
    "revision": 0,
    "state": "deleting",
    "url": "string",
    "visibility": "private"
  },
  "remoteUrl": "string",
  "size": 0,
  "sshUrl": "string",
  "url": "string",
  "validRemoteUrls": [
    "string"
  ],
  "webUrl": "string"
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
