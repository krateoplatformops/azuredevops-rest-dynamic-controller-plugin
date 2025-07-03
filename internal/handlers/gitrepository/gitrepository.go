package gitrepository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/internal/handlers"
)

// Handler constructors
func PostGitRepository(opts handlers.HandlerOptions) handlers.Handler {
	return &postHandler{baseHandler: newBaseHandler(opts)}
}

// Interface compliance verification
var _ handlers.Handler = &postHandler{}

// Base handler with common functionality
type baseHandler struct {
	handlers.HandlerOptions
}

// Constructor for the base handler
func newBaseHandler(opts handlers.HandlerOptions) *baseHandler {
	return &baseHandler{HandlerOptions: opts}
}

// Handler types embedding the base handler
type postHandler struct {
	*baseHandler
}

// Common methods, defined once on baseHandler
func (h *baseHandler) makeAzuredevopsRequest(method, url string, authHeader string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Basic Auth header (Azure DevOps requires Basic Auth for API access)
	if authHeader != "" {
		h.Log.Print("Using provided Authorization header for Basic authentication")
		req.Header.Set("Authorization", authHeader)
	} else {
		h.Log.Print("No Authorization header provided, Basic authentication required")
		return nil, fmt.Errorf("no Authorization header provided, Basic authentication required")
	}

	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

func (h *baseHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	h.Log.Print(message)
	w.WriteHeader(statusCode)
	w.Write([]byte(message))
}

func (h *baseHandler) writeJSONResponse(w http.ResponseWriter, statusCode int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(body)
}

// POST handler implementation
// @Summary Create a new GitRepository on Azure DevOps
// @Description Create a new GitRepository on Azure DevOps using the provided organization, project, and repository details.
// @ID post-gitrepository
// @Param organization path string true "Organization name"
// @Param projectId path string true "Project ID or name"
// @Param sourceRef path string false "Specify the source refs to use while creating a fork repo"
// @Param api-version query string true "API version (e.g., 7.2-preview.2)"
// @Param Authorization header string true "Basic Auth header (Basic <base64-encoded-username:password>)"
// @Param gitrepositoryCreate body CreateRepositoryRequest true "GitRepository creation request body (with additional fields handled by the plugin)"
// @Accept json
// @Produce json
// @Success 201 {object} CreateRepositoryResponse "GitRepository details"
// @Router /api/{organization}/{projectId}/git/repositories [post]
func (h *postHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	organization := r.PathValue("organization")
	projectId := r.PathValue("projectId")
	apiVersion := r.URL.Query().Get("api-version")
	sourceRef := r.URL.Query().Get("sourceRef") // Optional sourceRef parameter
	authHeader := r.Header.Get("Authorization")

	// Validate required parameters
	if organization == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Organization parameter is required")
		return
	}
	if projectId == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Project ID parameter is required")
		return
	}
	if apiVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "API version parameter is required")
		return
	}

	// This single check handles missing headers, incorrect formats, and empty credentials.
	username, password, ok := r.BasicAuth()
	if !ok || username == "" || password == "" {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Request rejected due to missing or invalid Basic authentication")
		return
	}

	// Read and parse the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var createRequest CreateRepositoryRequest
	if err := json.Unmarshal(body, &createRequest); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate required fields
	if createRequest.Name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Repository name is required")
		return
	}

	requestedDefaultBranch := createRequest.DefaultBranch
	branchToInit := "refs/heads/main" // Default branch to initialize with
	if requestedDefaultBranch != "" {
		branchToInit = requestedDefaultBranch
	}

	// Automatically enable initialization if defaultBranch was set and Initialize is not set
	if requestedDefaultBranch != "" && !createRequest.Initialize {
		createRequest.Initialize = true
		h.Log.Printf("Default branch specified without initialization — auto-enabling repository initialization")
	}

	// Decide if a defaultBranch update is needed
	needsDefaultBranchUpdate := requestedDefaultBranch != "" && requestedDefaultBranch != "refs/heads/main"

	// Create the repository request body for Azure DevOps (without defaultBranch)
	azureDevOpsRequest := GitRepositoryCreateOptionsMinimal{
		Name:             createRequest.Name,
		ParentRepository: createRequest.ParentRepository,
		Project:          createRequest.Project,
		// Note: defaultBranch is intentionally omitted for the initial POST request as POST does not support it
	}

	// Perform POST request to create a new GitRepository on Azure DevOps
	createdRepo, err := h.createGitRepository(organization, projectId, apiVersion, authHeader, sourceRef, azureDevOpsRequest)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create repository: %v", err))
		return
	}

	if createRequest.ParentRepository != nil {
		h.Log.Printf("Created repository '%s' as a fork of parent repository '%s'", createdRepo.Name, createRequest.ParentRepository.ID)
	}

	// forking and initializing are mutually exclusive operations.
	if createRequest.Initialize && createRequest.ParentRepository == nil {
		h.Log.Printf("Repository '%s' is not a fork, proceeding with initialization", createdRepo.Name)
		h.Log.Printf("Repository '%s' will be initialized with an initial commit on branch '%s'", createdRepo.Name, branchToInit)

		if err := h.initializeRepository(organization, projectId, createdRepo.ID, apiVersion, authHeader, branchToInit); err != nil {
			h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to initialize repository '%s': %v", createdRepo.Name, err))
			return
		}
		h.Log.Printf("Successfully initialized repository '%s' with an initial commit", createdRepo.Name)
	}

	// If defaultBranch is specified and is different from 'refs/heads/main',
	if needsDefaultBranchUpdate {
		h.Log.Printf("Setting default branch to '%s' for repository '%s'", requestedDefaultBranch, createdRepo.Name)

		// Check if the desired branch exists in the new repo.
		exists, err := h.branchExists(organization, projectId, createdRepo.ID, requestedDefaultBranch, apiVersion, authHeader)
		if err != nil {
			h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to check if branch '%s' exists: %v", requestedDefaultBranch, err))
			return
		}

		// If the branch does NOT exist, create it.
		if !exists {
			h.Log.Printf("Branch '%s' does not exist. Creating it now.", requestedDefaultBranch)

			// This `createBranch` function needs to be smart. It should branch from the repository's *current* default branch.
			if err := h.createBranch(organization, projectId, createdRepo.ID, requestedDefaultBranch, apiVersion, authHeader); err != nil {
				// handle error
				return
			}
		}

		h.Log.Printf("Setting default branch to '%s'", requestedDefaultBranch)
		updatedRepo, err := h.updateRepositoryDefaultBranch(organization, projectId, createdRepo.ID, requestedDefaultBranch, apiVersion, authHeader)
		if err != nil {
			h.Log.Printf("Warning: Failed to update default branch: %v", err)
			// Continue with the original created repository response
		} else {
			// Use the updated repository information
			createdRepo = updatedRepo
		}
	}

	// Convert the response to the expected format
	response := CreateRepositoryResponse{
		Name:             createdRepo.Name,
		ParentRepository: createdRepo.ParentRepository,
		Project:          createdRepo.Project,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}

	h.writeJSONResponse(w, http.StatusCreated, responseBody)
	h.Log.Printf("Successfully created GitRepository '%s' in organization '%s', project '%s'", createdRepo.Name, organization, projectId)
}

// createGitRepository performs the actual repository creation via Azure DevOps API
func (h *postHandler) createGitRepository(organization, projectId, apiVersion, authHeader, sourceRef string, request GitRepositoryCreateOptionsMinimal) (*GitRepository, error) {
	// Construct the URL for the Azure DevOps API
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories?api-version=%s", organization, projectId, apiVersion)

	// check if sourceRef is provided and append it to the URL
	if sourceRef != "" {
		url += fmt.Sprintf("&sourceRef=%s", sourceRef)
	}

	// Marshal the request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
	}

	h.Log.Printf("Creating repository with request body: %s", string(requestBody))

	// Make the POST request to Azure DevOps API
	resp, err := h.makeAzuredevopsRequest("POST", url, authHeader, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to make create repository request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read create repository response: %w", err)
	}

	// Check for non-201 status codes
	if resp.StatusCode != http.StatusCreated {
		h.Log.Printf("Azure DevOps API returned non-201 status for repository creation: %d. Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("azure DevOps API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var createdRepo GitRepository
	if err := json.Unmarshal(body, &createdRepo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create repository response: %w", err)
	}

	return &createdRepo, nil
}

// updateRepositoryDefaultBranch updates the default branch of an existing repository
func (h *postHandler) updateRepositoryDefaultBranch(organization, projectId, repositoryId, defaultBranch, apiVersion, authHeader string) (*GitRepository, error) {
	// Construct the URL for the Azure DevOps API
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s?api-version=%s", organization, projectId, repositoryId, apiVersion)

	// Create the update request
	updateRequest := GitRepositoryUpdateOptions{
		DefaultBranch: defaultBranch,
	}

	// Marshal the request body
	requestBody, err := json.Marshal(updateRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
	}

	h.Log.Printf("Updating repository default branch with request body: %s", string(requestBody))

	// Make the PATCH request to Azure DevOps API
	resp, err := h.makeAzuredevopsRequest("PATCH", url, authHeader, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to make update repository request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read update repository response: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		h.Log.Printf("Azure DevOps API returned non-200 status for repository update: %d. Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("azure DevOps API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var updatedRepo GitRepository
	if err := json.Unmarshal(body, &updatedRepo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal update repository response: %w", err)
	}

	return &updatedRepo, nil
}

func (h *postHandler) initializeRepository(organization, projectId, repositoryId, apiVersion, authHeader, branchToInit string) error {
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/pushes?api-version=%s",
		organization, projectId, repositoryId, apiVersion)

	requestBody := map[string]interface{}{
		"refUpdates": []map[string]string{
			{
				"name":        branchToInit, // Use the provided branch name
				"oldObjectId": "0000000000000000000000000000000000000000",
			},
		},
		"commits": []map[string]interface{}{
			{
				"comment": "Initial commit",
				"changes": []map[string]interface{}{
					{
						"changeType": "add",
						"item": map[string]string{
							"path": "/README.md",
						},
						"newContent": map[string]string{
							"content":     "# New Repository\n\nThis repository was initialized automatically.",
							"contentType": "rawtext",
						},
					},
				},
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal initial commit request: %w", err)
	}

	h.Log.Printf("Initializing repository with request body: %s", string(body))

	resp, err := h.makeAzuredevopsRequest("POST", url, authHeader, body)
	if err != nil {
		return fmt.Errorf("failed to initialize repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to initialize repository, status: %d, body: %s", resp.StatusCode, string(responseBody))
	}

	h.Log.Printf("Successfully initialized repository '%s'", repositoryId)
	return nil
}

// dynamically discovers the repository's default branch and uses it as the source for creating the new branch.
func (h *postHandler) createBranch(organization, projectId, repositoryId, branchName, apiVersion, authHeader string) error {

	// Step 1: Get the repository's details to find its default branch name.
	repoInfoURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s?api-version=%s", organization, projectId, repositoryId, apiVersion)

	resp, err := h.makeAzuredevopsRequest("GET", repoInfoURL, authHeader, nil)
	if err != nil {
		return fmt.Errorf("failed to get repository info to determine default branch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get repository info, status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Unmarshal only the field we need: defaultBranch
	var repoDetails struct {
		DefaultBranch string `json:"defaultBranch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoDetails); err != nil {
		return fmt.Errorf("failed to unmarshal repository details to find default branch: %w", err)
	}

	if repoDetails.DefaultBranch == "" {
		return fmt.Errorf("could not determine the default branch for the repository")
	}

	h.Log.Printf("Determined repository's default branch is '%s'. Using it as source.", repoDetails.DefaultBranch)

	// Step 2: Get the commit SHA (objectId) of that default branch.
	// The API gives us "refs/heads/main", so we trim the prefix for the filter.
	sourceBranchNameForFilter := strings.TrimPrefix(repoDetails.DefaultBranch, "refs/heads/")
	sourceBranchInfoURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/refs?filter=heads/%s&api-version=%s", organization, projectId, repositoryId, sourceBranchNameForFilter, apiVersion)

	resp, err = h.makeAzuredevopsRequest("GET", sourceBranchInfoURL, authHeader, nil)
	if err != nil {
		return fmt.Errorf("failed to get info for source branch '%s': %w", sourceBranchNameForFilter, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get info for source branch '%s', status: %d", sourceBranchNameForFilter, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read source branch response: %w", err)
	}

	var refsResponse struct {
		Value []struct {
			ObjectId string `json:"objectId"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &refsResponse); err != nil {
		return fmt.Errorf("failed to unmarshal source branch refs response: %w", err)
	}

	if len(refsResponse.Value) == 0 {
		return fmt.Errorf("source branch '%s' not found in repository, cannot create new branch", sourceBranchNameForFilter)
	}
	sourceBranchObjectId := refsResponse.Value[0].ObjectId

	// Step 3: Create the new branch pointing to the source branch's commit SHA.
	createBranchUrl := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/refs?api-version=%s", organization, projectId, repositoryId, apiVersion)

	createBranchRequest := []map[string]interface{}{
		{
			"name":        branchName,                                 // The new branch to create
			"oldObjectId": "0000000000000000000000000000000000000000", // Required for creating a new ref
			"newObjectId": sourceBranchObjectId,                       // Point the new branch to the source commit
		},
	}

	requestBody, err := json.Marshal(createBranchRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal create branch request: %w", err)
	}

	h.Log.Printf("Creating branch '%s' from source commit '%s'", branchName, sourceBranchObjectId)
	resp, err = h.makeAzuredevopsRequest("POST", createBranchUrl, authHeader, requestBody)
	if err != nil {
		return fmt.Errorf("failed to execute create branch request: %w", err)
	}
	defer resp.Body.Close()

	// The API returns 201 Created on success.
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create branch, status: %d, body: %s", resp.StatusCode, string(body))
	}

	h.Log.Printf("Successfully created branch '%s'", branchName)
	return nil
}

func (h *postHandler) branchExists(organization, projectId, repositoryId, branchName, apiVersion, authHeader string) (bool, error) {
	// Remove the 'refs/heads/' prefix if present for the API call
	branchNameForAPI := strings.TrimPrefix(branchName, "refs/heads/")

	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/refs?filter=heads/%s&api-version=%s",
		organization, projectId, repositoryId, branchNameForAPI, apiVersion)

	resp, err := h.makeAzuredevopsRequest("GET", url, authHeader, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check branch existence: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to check branch existence, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read branch check response: %w", err)
	}

	var refsResponse struct {
		Value []interface{} `json:"value"`
	}

	if err := json.Unmarshal(body, &refsResponse); err != nil {
		return false, fmt.Errorf("failed to unmarshal refs response: %w", err)
	}

	// log value for debugging
	h.Log.Printf("Branch existence check response: %s", string(body))

	return len(refsResponse.Value) > 0, nil
}
