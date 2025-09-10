package pullrequest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"

	"github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/internal/handlers"
)

func GetPullRequests(opts handlers.HandlerOptions) handlers.Handler {
	return &getHandler{baseHandler: newBaseHandler(opts)}
}

//	func PostPullRequest(opts handlers.HandlerOptions) handlers.Handler {
//		return &postHandler{baseHandler: newBaseHandler(opts)}
//	}
func PatchPullRequest(opts handlers.HandlerOptions) handlers.Handler {
	return &patchHandler{baseHandler: newBaseHandler(opts)}
}

// Interface compliance verification
var _ handlers.Handler = &getHandler{}

// var _ handlers.Handler = &postHandler{}
var _ handlers.Handler = &patchHandler{}

// Base handler with common functionality
type baseHandler struct {
	handlers.HandlerOptions
}

// Constructor for the base handler
func newBaseHandler(opts handlers.HandlerOptions) *baseHandler {
	return &baseHandler{HandlerOptions: opts}
}

type getHandler struct {
	*baseHandler
}

//	type postHandler struct {
//		*baseHandler
//	}
type patchHandler struct {
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

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
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

// Helper to get the current state of a pull request by ID
// For example, to determine what fields need to be updated in a PATCH operation
// since Azure DevOps throws an error if you try to set a field to its current value (`TargetRefName`)
func (h *patchHandler) getPullRequestByID(organization, project, repositoryId, pullRequestId, apiVersion, authHeader string) (*GitPullRequest, error) {
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/pullrequests/%s?api-version=%s",
		organization, project, repositoryId, pullRequestId, apiVersion)

	resp, err := h.makeAzuredevopsRequest("GET", url, authHeader, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request to azure devops: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure devops returned non-200 status: %d - %s", resp.StatusCode, string(body))
	}

	var currentPR GitPullRequest
	if err := json.Unmarshal(body, &currentPR); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pull request response: %w", err)
	}

	return &currentPR, nil
}

// GET handler implementation
// @Summary Get pull requests (list)
// @Description Retrieve all pull requests matching a specified criteria.
// @ID get-pullrequests
// @Param organization path string true "Organization name"
// @Param project path string true "Project name or ID"
// @Param repositoryId path string true "Repository ID"
// @Param sourceRefName query string true "Search for pull requests from this branch."
// @Param targetRefName query string true "Search for pull requests into this branch."
// @Param title query string true "Search pull requests that contain the specified text in the title."
// @Param api-version query string true "API version (e.g., 7.2-preview.2)"
// @Param Authorization header string true "Basic Auth header"
// @Produce json
// @Success 200 {array} GitPullRequest "A list of pull requests"
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/{organization}/{project}/git/repositories/{repositoryId}/pullrequests [get]
func (h *getHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	organization := r.PathValue("organization")
	if organization == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "organization parameter is required")
		return
	}

	project := r.PathValue("project")
	if project == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "project parameter is required")
		return
	}

	repositoryId := r.PathValue("repositoryId")
	if repositoryId == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "repositoryId parameter is required")
		return
	}

	apiVersion := r.URL.Query().Get("api-version")
	if apiVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "api-version is required")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeErrorResponse(w, http.StatusUnauthorized, "request rejected due to missing or invalid Basic authentication")
		return
	}

	// Validate and transform search query parameters
	query := r.URL.Query()
	sourceRefName := query.Get("sourceRefName")
	//status := query.Get("status")
	targetRefName := query.Get("targetRefName")
	title := query.Get("title")

	if sourceRefName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "query parameter 'sourceRefName' is required")
		return
	}
	//if status == "" {
	//	h.writeErrorResponse(w, http.StatusBadRequest, "query parameter 'status' is required")
	//	return
	//}
	if targetRefName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "query parameter 'targetRefName' is required")
		return
	}
	if title == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "query parameter 'title' is required")
		return
	}

	//validStatuses := map[string]bool{
	//	"notSet": true, "active": true, "abandoned": true, "completed": true, "all": true,
	//}
	//if !validStatuses[status] {
	//	h.writeErrorResponse(w, http.StatusBadRequest, "invalid status value; must be one of: notSet, active, abandoned, completed, all")
	//	return
	//}

	adoQuery := url.Values{}
	adoQuery.Set("api-version", apiVersion)
	adoQuery.Set("searchCriteria.sourceRefName", sourceRefName)
	//adoQuery.Set("searchCriteria.status", status)
	adoQuery.Set("searchCriteria.targetRefName", targetRefName)
	adoQuery.Set("searchCriteria.title", title)

	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/pullrequests?%s",
		organization, project, repositoryId, adoQuery.Encode())

	resp, err := h.makeAzuredevopsRequest("GET", url, authHeader, nil)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error making request to azure devops: %s", err.Error()))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error reading response body: %s", err.Error()))
		return
	}

	h.writeJSONResponse(w, resp.StatusCode, body)
}

// POST handler implementation
// @Summary Create a pull request
// @Description Create a new pull request.
// @ID create-pullrequest
// @Param organization path string true "Organization name"
// @Param project path string true "Project name or ID"
// @Param repositoryId path string true "Repository ID"
// @Param supportsIterations query bool true "If true, subsequent pushes to the pull request will be individually reviewable. Set this to false for large pull requests for performance reasons if this functionality is not needed."
// @Param api-version query string true "API version (e.g., 7.2-preview.2)"
// @Param Authorization header string true "Basic Auth header"
// @Param pullRequest body CreatePullRequestReq true "Pull request to create"
// @Accept json
// @Produce json
// @Success 201 {object} GitPullRequest "Created pull request"
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 409 {string} string "Conflict"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/{organization}/{project}/git/repositories/{repositoryId}/pullrequests [post]
//func (h *postHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	organization := r.PathValue("organization")
//	if organization == "" {
//		h.writeErrorResponse(w, http.StatusBadRequest, "organization parameter is required")
//		return
//	}
//
//	project := r.PathValue("project")
//	if project == "" {
//		h.writeErrorResponse(w, http.StatusBadRequest, "project parameter is required")
//		return
//	}
//
//	repositoryId := r.PathValue("repositoryId")
//	if repositoryId == "" {
//		h.writeErrorResponse(w, http.StatusBadRequest, "repositoryId parameter is required")
//		return
//	}
//
//	supportsIterations := r.URL.Query().Get("supportsIterations")
//
//	apiVersion := r.URL.Query().Get("api-version")
//	if apiVersion == "" {
//		h.writeErrorResponse(w, http.StatusBadRequest, "api-version is required")
//		return
//	}
//
//	authHeader := r.Header.Get("Authorization")
//	if authHeader == "" {
//		h.writeErrorResponse(w, http.StatusUnauthorized, "request rejected due to missing or invalid Basic authentication")
//		return
//	}
//
//	requestBody, err := io.ReadAll(r.Body)
//	if err != nil {
//		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error reading request body: %s", err.Error()))
//		return
//	}
//
//	var pr CreateUpdatePullRequestReq
//	if err := json.Unmarshal(requestBody, &pr); err != nil {
//		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error unmarshaling request body: %s", err.Error()))
//		return
//	}
//
//	createReq := CreatePullRequestReq{
//		CompletionOptions: pr.CompletionOptions,
//		Description:       pr.Description,
//		IsDraft:           pr.IsDraft,
//		Reviewers:         pr.Reviewers,
//		SourceRefName:     pr.SourceRefName,
//		Status:            pr.Status, // Typically "active" ?
//		TargetRefName:     pr.TargetRefName,
//		Title:             pr.Title,
//	}
//
//	createBody, err := json.Marshal(createReq)
//	if err != nil {
//		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error marshaling create request: %s", err.Error()))
//		return
//	}
//
//	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/pullrequests?api-version=%s", organization, project, repositoryId, apiVersion)
//
//	if supportsIterations != "" {
//		url += fmt.Sprintf("&supportsIterations=%s", supportsIterations)
//	}
//
//	resp, err := h.makeAzuredevopsRequest("POST", url, authHeader, createBody)
//	if err != nil {
//		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error making request to azure devops: %s", err.Error()))
//		return
//	}
//	defer resp.Body.Close()
//
//	body, err := io.ReadAll(resp.Body)
//	if err != nil {
//		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error reading response body: %s", err.Error()))
//		return
//	}
//
//	h.writeJSONResponse(w, resp.StatusCode, body)
//}

// PATCH handler implementation
// @Summary Update a pull request
// @Description Update an existing pull request.
// @ID update-pullrequest
// @Param organization path string true "Organization name"
// @Param project path string true "Project name or ID"
// @Param repositoryId path string true "Repository ID"
// @Param pullRequestId path int true "Pull Request ID"
// @Param api-version query string true "API version (e.g., 7.2-preview.2)"
// @Param Authorization header string true "Basic Auth header"
// @Param pullRequest body UpdatePullRequestReq true "Pull request updates"
// @Accept json
// @Produce json
// @Success 200 {object} GitPullRequest "Updated pull request"
// @Failure 400 {string} string "Bad Request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Not Found"
// @Failure 409 {string} string "Conflict"
// @Failure 500 {string} string "Internal Server Error"
// @Router /api/{organization}/{project}/git/repositories/{repositoryId}/pullrequests/{pullRequestId} [patch]
func (h *patchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	organization := r.PathValue("organization")
	if organization == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "organization parameter is required")
		return
	}

	project := r.PathValue("project")
	if project == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "project parameter is required")
		return
	}

	repositoryId := r.PathValue("repositoryId")
	if repositoryId == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "repositoryId parameter is required")
		return
	}

	pullRequestId := r.PathValue("pullRequestId")
	if pullRequestId == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "pullRequestId parameter is required")
		return
	}

	apiVersion := r.URL.Query().Get("api-version")
	if apiVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "api-version is required")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeErrorResponse(w, http.StatusUnauthorized, "request rejected due to missing or invalid Basic authentication")
		return
	}

	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error reading request body: %s", err.Error()))
		return
	}

	var desiredPR UpdatePullRequestReq
	if err := json.Unmarshal(requestBody, &desiredPR); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("error unmarshaling request body: %s", err.Error()))
		return
	}

	// Get the current state of the PR
	currentPR, err := h.getPullRequestByID(organization, project, repositoryId, pullRequestId, apiVersion, authHeader)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error getting current pull request state: %s", err.Error()))
		return
	}

	// Build a patch payload with only the changed fields
	patchPayload := make(map[string]interface{})

	if desiredPR.Title != "" && desiredPR.Title != currentPR.Title {
		patchPayload["title"] = desiredPR.Title
	}
	if desiredPR.Description != "" && desiredPR.Description != currentPR.Description {
		patchPayload["description"] = desiredPR.Description
	}
	if desiredPR.Status != "" && desiredPR.Status != currentPR.Status {
		patchPayload["status"] = desiredPR.Status
	}
	if desiredPR.CompletionOptions != nil && !reflect.DeepEqual(desiredPR.CompletionOptions, currentPR.CompletionOptions) {
		patchPayload["completionOptions"] = desiredPR.CompletionOptions
	}

	// If nothing changed, return the current PR and do nothing.
	if len(patchPayload) == 0 {
		h.Log.Print("No changes detected for pull request update, returning current state.")
		currentPRBody, _ := json.Marshal(currentPR)
		h.writeJSONResponse(w, http.StatusOK, currentPRBody)
		return
	}

	patchBody, err := json.Marshal(patchPayload)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error marshaling patch payload: %s", err.Error()))
		return
	}

	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/git/repositories/%s/pullrequests/%s?api-version=%s", organization, project, repositoryId, pullRequestId, apiVersion)

	resp, err := h.makeAzuredevopsRequest("PATCH", url, authHeader, patchBody)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error making request to azure devops: %s", err.Error()))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("error reading response body: %s", err.Error()))
		return
	}

	h.writeJSONResponse(w, resp.StatusCode, body)
}
