package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/krateoplatformops/azuredevops-rest-dynamic-controller-plugin/internal/handlers"
)

// Handler constructors
func GetPipeline(opts handlers.HandlerOptions) handlers.Handler {
	return &getHandler{baseHandler: newBaseHandler(opts)}
}

func PostPipeline(opts handlers.HandlerOptions) handlers.Handler {
	return &postHandler{baseHandler: newBaseHandler(opts)}
}

func DeletePipeline(opts handlers.HandlerOptions) handlers.Handler {
	return &deleteHandler{baseHandler: newBaseHandler(opts)}
}

// Interface compliance verification
var _ handlers.Handler = &getHandler{}
var _ handlers.Handler = &postHandler{}
var _ handlers.Handler = &deleteHandler{}

// Base handler with common functionality
type baseHandler struct {
	handlers.HandlerOptions
}

// Constructor for the base handler
func newBaseHandler(opts handlers.HandlerOptions) *baseHandler {
	return &baseHandler{HandlerOptions: opts}
}

// Handler types embedding the base handler
type getHandler struct {
	*baseHandler
}

type postHandler struct {
	*baseHandler
}

type deleteHandler struct {
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

func (h *baseHandler) validateBasicParams(w http.ResponseWriter, organization, project, apiVersion string) bool {
	if organization == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Organization parameter is required")
		return false
	}
	if project == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Project parameter is required")
		return false
	}
	if apiVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "API version is required")
		return false
	}
	return true
}

func (h *baseHandler) processPipelineResponse(body []byte) ([]byte, error) {

	var finalBody []byte = body

	// Check if the field 'folder' exists in the response body
	if value, err := ReadFieldFromBody(body, "folder"); err != nil {
		// we will not modify the body if the field is not found or if there is an error reading it
		h.Log.Printf("Error reading 'folder' field from response body: %v", err)
	} else {
		h.Log.Printf("Field 'folder' found in response body")

		// Ensure the value is a string
		if folder, ok := value.(string); ok {
			if strings.HasPrefix(folder, "\\") {
				h.Log.Printf("Field 'folder' starts with '\\' (escaped backslash), modifying it")

				// Remove the leading escaped backslash
				modifiedFolder := folder[1:]

				// Update the field in the body
				finalBody, err = AddFieldToBody(finalBody, "folder", modifiedFolder)
				if err != nil {
					return nil, fmt.Errorf("failed to add modified 'folder' field to response body: %w", err)
				}
				h.Log.Printf("Modified 'folder' field added to response body")
			} else {
				h.Log.Printf("Field 'folder' does not start with '\\' (escaped backslash), no modification needed")
			}
		} else {
			h.Log.Printf("Field 'folder' is not a string, skipping modification")
		}
	}

	return finalBody, nil

}

// GET handler implementation
// @Summary Get a pipeline
// @Description Get
// @ID get-pipeline
// @Param organization path string true "Organization name"
// @Param project path string true "Project name or ID"
// @Param id path string true "Pipeline ID"
// @Param api-version query string true "API version (e.g., 7.2-preview.1)"
// @Param Authorization header string true "Basic Auth header (Basic <base64-encoded-username:password>)"
// @Produce json
// @Success 200 {object} GetPipelineResponse "Pipeline details"
// @Failure 400
// @Failure 401 "Unauthorized"
// @Failure 500 "Internal Server Error"
// @Router /api/{organization}/{project}/pipelines/{id} [get]
func (h *getHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	organization := r.PathValue("organization")
	project := r.PathValue("project")
	id := r.PathValue("id")
	apiVersion := r.URL.Query().Get("api-version")
	authHeader := r.Header.Get("Authorization")

	// Validate required parameters
	if !h.validateBasicParams(w, organization, project, apiVersion) {
		return
	}

	// This single check handles missing headers, incorrect formats, and empty credentials.
	username, password, ok := r.BasicAuth()
	if !ok || username == "" || password == "" {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Request rejected due to missing or invalid Basic authentication")
		return
	}

	h.Log.Printf("Getting pipeline with ID %s for organization %s and project %s", id, organization, project)

	// Get Pipeline and respond
	// This method handles the actual API call and response processing
	// It returns an error if something goes wrong
	err := h.getPipelineAndRespond(w, organization, project, id, apiVersion, authHeader)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error getting pipeline: %v", err))
	}
}

func (h *getHandler) getPipelineAndRespond(w http.ResponseWriter, organization, project, id, apiVersion, authHeader string) error {
	// Construct the URL for the Azure DevOps API
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/pipelines/%s?api-version=%s", organization, project, id, apiVersion)

	// Make the request to Azure DevOps API
	resp, err := h.makeAzuredevopsRequest("GET", url, authHeader, nil)
	if err != nil {
		return fmt.Errorf("failed to get pipeline: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body first, as it's useful for both success and error cases
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-200 status codes after reading the body
	if resp.StatusCode != http.StatusOK {
		h.Log.Printf("Azure DevOps API returned a non-200 status: %d. Body: %s", resp.StatusCode, string(body))
		h.writeJSONResponse(w, resp.StatusCode, body)
		return nil
	}

	// Process the response body through the transformation pipeline
	processedBody, err := h.processPipelineResponse(body)
	if err != nil {
		h.Log.Printf("Failed to process response, returning original raw response from Azure DevOps API: %v", err)
		// Fallback to sending the original, valid response
		h.writeJSONResponse(w, http.StatusOK, body)
		return nil
	}

	var pipeline Pipeline
	if err := json.Unmarshal(processedBody, &pipeline); err != nil {
		h.Log.Printf("Failed to unmarshal processed pipeline response: %v", err)
		// Fallback to sending the original, valid response
		h.writeJSONResponse(w, http.StatusOK, body)
		return nil
	}

	h.writeJSONResponse(w, http.StatusOK, processedBody)
	h.Log.Printf("Successfully retrieved pipeline with ID %s", id)
	return nil
}

// Implemented but not currently used by the RestDefinition
// POST handler implementation
// @Summary Create a new Pipeline
// @Description Create a new Pipeline
// @ID post-pipeline
// @Param organization path string true "Organization name"
// @Param projectId path string true "Project ID or name"
// @Param api-version query string true "API version (e.g., 7.2-preview.1)"
// @Param Authorization header string true "Basic Auth header (Basic <base64-encoded-username:password>)"
// @Param pipelineCreate body CreatePipelineRequest true "Pipeline creation request body"
// @Accept json
// @Produce json
// @Success 201 {object} CreatePipelineResponse "Pipeline details"
// @Failure 400 "Bad Request"
// @Failure 401 "Unauthorized"
// @Failure 500 "Internal Server Error"
// @Router /api/{organization}/{project}/pipelines/ [post]
func (h *postHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	organization := r.PathValue("organization")
	project := r.PathValue("project")
	apiVersion := r.URL.Query().Get("api-version")
	authHeader := r.Header.Get("Authorization")

	// Validate required parameters
	if !h.validateBasicParams(w, organization, project, apiVersion) {
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

	var createRequest CreatePipelineRequest
	if err := json.Unmarshal(body, &createRequest); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON in request body")
		return
	}

	// Validate required fields
	if createRequest.Name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Pipeline name is required")
		return
	}

	// other validations can be added here as needed (with a specific function)

	createdPipeline, err := h.createPipeline(organization, project, apiVersion, authHeader, createRequest)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create pipeline: %v", err))
		return
	}

	response := CreatePipelineResponse(*createdPipeline)

	responseBody, err := json.Marshal(response)
	if err != nil {
		h.Log.Printf("Failed to marshal response: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}

	h.writeJSONResponse(w, http.StatusCreated, responseBody)
	h.Log.Printf("Successfully created Pipeline with name %s", createRequest.Name)
}

// createPipeline performs the actual pipeline creation via Azure DevOps API
func (h *postHandler) createPipeline(organization, project, apiVersion, authHeader string, request CreatePipelineRequest) (*Pipeline, error) {
	// Construct the URL for the Azure DevOps API
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/pipelines?api-version=%s", organization, project, apiVersion)

	// Marshal the request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
	}

	h.Log.Printf("Creating pipeline with request body: %s", string(requestBody))

	// Make the POST request to Azure DevOps API
	resp, err := h.makeAzuredevopsRequest("POST", url, authHeader, requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to make create pipeline request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body from the Azure DevOps API
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read create pipeline response: %w", err)
	}

	// Check for non-201 status codes
	if resp.StatusCode != http.StatusCreated {
		h.Log.Printf("Azure DevOps API returned non-201 status for pipeline creation: %d. Body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("azure devops API returned status %d: %s", resp.StatusCode, string(body))
	}

	var finalBody []byte
	finalBody, err = h.processPipelineResponse(body)
	if err != nil {
		h.Log.Printf("Failed to process response, falling back to raw body: %v", err)
		finalBody = body
	}

	// Parse the response
	var createdPipeline Pipeline
	if err := json.Unmarshal(finalBody, &createdPipeline); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create pipeline response: %w", err)
	}

	return &createdPipeline, nil
}

// DELETE handler implementation
// @Summary Delete a pipeline
// @Description Delete a pipeline using build definitions endpoint
// @ID delete-pipeline
// @Param organization path string true "Organization name"
// @Param project path string true "Project name or ID"
// @Param id path string true "Pipeline ID"
// @Param Authorization header string true "Basic Auth header (Basic <base64-encoded-username:password>)"
// @Success 204 "No Content - Pipeline deleted successfully"
// @Failure 400 "Bad Request"
// @Failure 401 "Unauthorized"
// @Failure 404 "Not Found"
// @Failure 500 "Internal Server Error"
// @Router /api/{organization}/{project}/pipelines/{id} [delete]
func (h *deleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	organization := r.PathValue("organization")
	project := r.PathValue("project")
	id := r.PathValue("id")

	apiVersion := os.Getenv("BUILD_DEFINITIONS_API_VERSION")
	if apiVersion == "" {
		h.Log.Print("BUILD_DEFINITIONS_API_VERSION environment variable not set, using default API version")
		apiVersion = "7.2-preview.7" // Default Build Definition API version if not set
	}

	authHeader := r.Header.Get("Authorization")

	// Validate required parameters
	if !h.validateBasicParams(w, organization, project, apiVersion) {
		return
	}

	// This single check handles missing headers, incorrect formats, and empty credentials.
	username, password, ok := r.BasicAuth()
	if !ok || username == "" || password == "" {
		h.writeErrorResponse(w, http.StatusUnauthorized, "Request rejected due to missing or invalid Basic authentication")
		return
	}

	h.Log.Printf("Deleting pipeline with ID %s for organization %s and project %s", id, organization, project)

	// Delete Pipeline and respond
	err := h.deletePipelineAndRespond(w, organization, project, id, apiVersion, authHeader)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Error deleting pipeline: %v", err))
	}
}

func (h *deleteHandler) deletePipelineAndRespond(w http.ResponseWriter, organization, project, id, apiVersion, authHeader string) error {
	// Construct the URL for the Azure DevOps build definitions API
	// the /pipelines/{id} endpoints do not support deletion, so we use the build definitions endpoint
	url := fmt.Sprintf("https://dev.azure.com/%s/%s/_apis/build/definitions/%s?api-version=%s", organization, project, id, apiVersion)

	// Make the DELETE request to Azure DevOps API
	resp, err := h.makeAzuredevopsRequest("DELETE", url, authHeader, nil)
	if err != nil {
		return fmt.Errorf("failed to delete pipeline: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body for error handling
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for successful deletion (204 No Content)
	if resp.StatusCode == http.StatusNoContent {
		h.Log.Printf("Successfully deleted pipeline with ID %s", id)
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	// Handle other response codes
	if resp.StatusCode == http.StatusNotFound {
		h.Log.Printf("Pipeline with ID %s not found", id)
		h.writeJSONResponse(w, http.StatusNotFound, body)
		return nil
	}

	// Log and return error response for other status codes
	h.Log.Printf("Azure DevOps API returned status %d for pipeline deletion: %s", resp.StatusCode, string(body))
	h.writeJSONResponse(w, resp.StatusCode, body)
	return nil
}
