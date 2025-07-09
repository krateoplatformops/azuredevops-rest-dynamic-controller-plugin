package pipeline

import "time"

// Pipeline represents the response from:
// GET /{organization}/{project}/_apis/pipelines/{id}
// POST /{organization}/{project}/_apis/pipelines
type Pipeline struct {
	Links         *ReferenceLinks        `json:"_links,omitempty"`
	Configuration *PipelineConfiguration `json:"configuration,omitempty"`
	URL           string                 `json:"url,omitempty"`
	// Embedded fields from PipelineBase
	Folder   string `json:"folder,omitempty"`
	ID       int32  `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Revision int32  `json:"revision,omitempty"`
}

// CreatePipelineParametersComplete represents the request body for:
// POST /{organization}/{project}/_apis/pipelines
type CreatePipelineParametersComplete struct {
	Configuration *CreatePipelineConfigurationParametersComplete `json:"configuration"` // Required
	Folder        string                                         `json:"folder,omitempty"`
	Name          string                                         `json:"name"` // Required
}

// CreatePipelineConfigurationParametersComplete represents the configuration for creating a pipeline
type CreatePipelineConfigurationParametersComplete struct {
	Type       string           `json:"type"` // Required - enum: unknown, yaml, designerJson, justInTime, designerHyphenJson
	Path       string           `json:"path,omitempty"`
	Repository *BuildRepository `json:"repository"` // Required
}

// BuildRepository represents repository information for the pipeline
type BuildRepository struct {
	ID   string `json:"id"`   // Required
	Type string `json:"type"` // Required - enum: unknown, gitHub, azureReposGit, azureReposGitHyphenated
}

// PipelineConfiguration represents the configuration of a pipeline
type PipelineConfiguration struct {
	Type string `json:"type,omitempty"` // enum: unknown, yaml, designerJson, justInTime, designerHyphenJson
}

// ReferenceLinks represents a collection of REST reference links
type ReferenceLinks struct {
	Links map[string]interface{} `json:"links,omitempty"`
}

// BuildDefinition represents the response from:
// GET /{organization}/{project}/_apis/build/definitions/{id}
// PUT /{organization}/{project}/_apis/build/definitions/{id}
type BuildDefinition struct {
	Links                     *ReferenceLinks                    `json:"_links,omitempty"`
	BadgeEnabled              bool                               `json:"badgeEnabled,omitempty"`
	BuildNumberFormat         string                             `json:"buildNumberFormat,omitempty"`
	Comment                   string                             `json:"comment,omitempty"`
	CreatedDate               *time.Time                         `json:"createdDate,omitempty"`
	DateCreated               *time.Time                         `json:"dateCreated,omitempty"`
	DraftOf                   *BuildDefinitionReference          `json:"draftOf,omitempty"`
	Drafts                    []BuildDefinitionReference         `json:"drafts,omitempty"`
	DropLocation              string                             `json:"dropLocation,omitempty"`
	ID                        int32                              `json:"id,omitempty"`
	JobAuthorizationScope     string                             `json:"jobAuthorizationScope,omitempty"`
	JobCancelTimeoutInMinutes int32                              `json:"jobCancelTimeoutInMinutes,omitempty"`
	JobTimeoutInMinutes       int32                              `json:"jobTimeoutInMinutes,omitempty"`
	LatestBuild               *Build                             `json:"latestBuild,omitempty"`
	LatestCompletedBuild      *Build                             `json:"latestCompletedBuild,omitempty"`
	Metrics                   []BuildMetric                      `json:"metrics,omitempty"`
	Name                      string                             `json:"name,omitempty"`
	Path                      string                             `json:"path,omitempty"`
	Process                   interface{}                        `json:"process,omitempty"`
	ProcessParameters         *ProcessParameters                 `json:"processParameters,omitempty"`
	Project                   *ProjectReference                  `json:"project,omitempty"`
	Properties                map[string]interface{}             `json:"properties,omitempty"`
	Quality                   string                             `json:"quality,omitempty"`
	Repository                *BuildRepository                   `json:"repository,omitempty"`
	Revision                  int32                              `json:"revision,omitempty"`
	Tags                      []string                           `json:"tags,omitempty"`
	Type                      string                             `json:"type,omitempty"`
	URI                       string                             `json:"uri,omitempty"`
	URL                       string                             `json:"url,omitempty"`
	VariableGroups            []VariableGroup                    `json:"variableGroups,omitempty"`
	Variables                 map[string]BuildDefinitionVariable `json:"variables,omitempty"`
}

// BuildDefinitionReference represents a reference to a build definition
type BuildDefinitionReference struct {
	ID      int32             `json:"id,omitempty"`
	Name    string            `json:"name,omitempty"`
	URI     string            `json:"uri,omitempty"`
	URL     string            `json:"url,omitempty"`
	Path    string            `json:"path,omitempty"`
	Type    string            `json:"type,omitempty"`
	Project *ProjectReference `json:"project,omitempty"`
}

// Build represents a build
type Build struct {
	ID          int32      `json:"id,omitempty"`
	BuildNumber string     `json:"buildNumber,omitempty"`
	Status      string     `json:"status,omitempty"`
	Result      string     `json:"result,omitempty"`
	QueueTime   *time.Time `json:"queueTime,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	FinishTime  *time.Time `json:"finishTime,omitempty"`
	URI         string     `json:"uri,omitempty"`
	URL         string     `json:"url,omitempty"`
}

// BuildMetric represents a build metric
type BuildMetric struct {
	Name  string      `json:"name,omitempty"`
	Scope string      `json:"scope,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

// ProcessParameters represents process parameters for a build definition
type ProcessParameters struct {
	Inputs []ProcessParameterInput `json:"inputs,omitempty"`
}

// ProcessParameterInput represents a process parameter input
type ProcessParameterInput struct {
	Name         string                 `json:"name,omitempty"`
	Label        string                 `json:"label,omitempty"`
	DefaultValue string                 `json:"defaultValue,omitempty"`
	Required     bool                   `json:"required,omitempty"`
	Type         string                 `json:"type,omitempty"`
	HelpMarkDown string                 `json:"helpMarkDown,omitempty"`
	VisibleRule  string                 `json:"visibleRule,omitempty"`
	Properties   map[string]interface{} `json:"properties,omitempty"`
}

// ProjectReference represents a reference to a project
type ProjectReference struct {
	ID             string     `json:"id,omitempty"`
	Name           string     `json:"name,omitempty"`
	Description    string     `json:"description,omitempty"`
	URL            string     `json:"url,omitempty"`
	State          string     `json:"state,omitempty"`
	Revision       int32      `json:"revision,omitempty"`
	Visibility     string     `json:"visibility,omitempty"`
	LastUpdateTime *time.Time `json:"lastUpdateTime,omitempty"`
}

// VariableGroup represents a variable group
type VariableGroup struct {
	ID          int32               `json:"id,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	Type        string              `json:"type,omitempty"`
	Variables   map[string]Variable `json:"variables,omitempty"`
}

// Variable represents a variable
type Variable struct {
	Value         string `json:"value,omitempty"`
	IsSecret      bool   `json:"isSecret,omitempty"`
	AllowOverride bool   `json:"allowOverride,omitempty"`
}

// BuildDefinitionVariable represents a build definition variable
type BuildDefinitionVariable struct {
	Value         string `json:"value,omitempty"`
	AllowOverride bool   `json:"allowOverride,omitempty"`
	IsSecret      bool   `json:"isSecret,omitempty"`
}

// UpdatePipelineRequest represents the request body for updating a pipeline
// This maps to the build definition structure since we use the build definitions endpoint
type UpdatePipelineRequest struct {
	BadgeEnabled      bool                               `json:"badgeEnabled,omitempty"`
	BuildNumberFormat string                             `json:"buildNumberFormat,omitempty"`
	Comment           string                             `json:"comment,omitempty"`
	ID                int32                              `json:"id,omitempty"`
	Name              string                             `json:"name"` // Required
	Path              string                             `json:"path,omitempty"`
	Process           interface{}                        `json:"process,omitempty"`
	ProcessParameters *ProcessParameters                 `json:"processParameters,omitempty"`
	Properties        map[string]interface{}             `json:"properties,omitempty"`
	Quality           string                             `json:"quality,omitempty"`
	Repository        *BuildRepository                   `json:"repository,omitempty"`
	Revision          int32                              `json:"revision,omitempty"`
	Tags              []string                           `json:"tags,omitempty"`
	Type              string                             `json:"type,omitempty"`
	VariableGroups    []VariableGroup                    `json:"variableGroups,omitempty"`
	Variables         map[string]BuildDefinitionVariable `json:"variables,omitempty"`
}

// ConfigurationType enum values
const (
	ConfigurationTypeUnknown            = "unknown"
	ConfigurationTypeYAML               = "yaml"
	ConfigurationTypeDesignerJSON       = "designerJson"
	ConfigurationTypeJustInTime         = "justInTime"
	ConfigurationTypeDesignerHyphenJSON = "designerHyphenJson"
)

// RepositoryType enum values
const (
	RepositoryTypeUnknown                 = "unknown"
	RepositoryTypeGitHub                  = "gitHub"
	RepositoryTypeAzureReposGit           = "azureReposGit"
	RepositoryTypeAzureReposGitHyphenated = "azureReposGitHyphenated"
)

// BuildDefinitionQuality enum values
const (
	BuildDefinitionQualityDefinition = "definition"
	BuildDefinitionQualityDraft      = "draft"
)

// BuildDefinitionType enum values
const (
	BuildDefinitionTypeXaml  = "xaml"
	BuildDefinitionTypeBuild = "build"
)

// JobAuthorizationScope enum values
const (
	JobAuthorizationScopeProjectCollection = "projectCollection"
	JobAuthorizationScopeProject           = "project"
)

// GetPipelineResponse represents the response for getting a single pipeline
type GetPipelineResponse Pipeline

// CreatePipelineRequest represents the request for creating a pipeline
type CreatePipelineRequest CreatePipelineParametersComplete

// CreatePipelineResponse represents the response for creating a pipeline
type CreatePipelineResponse Pipeline

// UpdatePipelineResponse represents the response for updating a pipeline
type UpdatePipelineResponse BuildDefinition
