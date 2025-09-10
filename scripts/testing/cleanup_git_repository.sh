#!/bin/bash

# Set your environment variables
export ORGANIZATION="krateo-kog"
export PROJECT_NAME="test-project-1-classic"
export API_VERSION="7.2-preview.2"
#export API_VERSION="7.1"
export USERNAME="your-username"
export PAT="${AZURE_DEVOPS_PAT:-your-personal-access-token}"

export PROJECT=11790bc5-82bd-4cdc-b6a6-47bcb7187051

# List of repository names to exclude from deletion
EXCLUDE_REPOS=("test-project-1-classic" "test-repo-1-classic")  # Replace with the names of repositories to keep

# Base64-encoded PAT for authentication
AUTH=$(echo -n ":$PAT" | base64)

# Fetch all repositories in the project
echo "Fetching repositories from project '$PROJECT'..."

REPOS_JSON=$(curl -s -H "Authorization: Basic $AUTH" \
  "https://dev.azure.com/$ORGANIZATION/$PROJECT/_apis/git/repositories?api-version=$API_VERSION")

# Debug: Check if the response is valid JSON
echo "API Response:"
echo "$REPOS_JSON"
echo "---"

# Check if jq can parse the response
if ! echo "$REPOS_JSON" | jq empty 2>/dev/null; then
    echo "Error: Invalid JSON response from API"
    echo "Response received: $REPOS_JSON"
    exit 1
fi

# Check if the response contains an error
if echo "$REPOS_JSON" | jq -e '.message' >/dev/null 2>&1; then
    echo "API Error: $(echo "$REPOS_JSON" | jq -r '.message')"
    exit 1
fi

# Extract repository names and IDs
REPO_NAMES=($(echo "$REPOS_JSON" | jq -r '.value[].name'))
REPO_IDS=($(echo "$REPOS_JSON" | jq -r '.value[].id'))

# Check if we got any repositories
if [ ${#REPO_NAMES[@]} -eq 0 ]; then
    echo "No repositories found in the project."
    exit 0
fi

echo "Found ${#REPO_NAMES[@]} repositories:"
for i in "${!REPO_NAMES[@]}"; do
    echo "  ${REPO_NAMES[$i]} (${REPO_IDS[$i]})"
done
echo "---"

# Iterate over repositories
for i in "${!REPO_NAMES[@]}"; do
  REPO_NAME="${REPO_NAMES[$i]}"
  REPO_ID="${REPO_IDS[$i]}"

  # Check if the repository is in the exclusion list
  if [[ " ${EXCLUDE_REPOS[@]} " =~ " $REPO_NAME " ]]; then
    echo "Skipping repository '$REPO_NAME' (excluded)."
    continue
  fi

  # Delete the repository
  echo "Deleting repository '$REPO_NAME'..."
  RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE \
    -H "Authorization: Basic $AUTH" \
    "https://dev.azure.com/$ORGANIZATION/$PROJECT/_apis/git/repositories/$REPO_ID?api-version=$API_VERSION")

  if [[ "$RESPONSE" == "204" ]]; then
    echo "Repository '$REPO_NAME' deleted successfully."
  else
    echo "Failed to delete repository '$REPO_NAME'. HTTP status code: $RESPONSE"
    
    # Get more detailed error information
    ERROR_RESPONSE=$(curl -s -X DELETE \
      -H "Authorization: Basic $AUTH" \
      "https://dev.azure.com/$ORGANIZATION/$PROJECT/_apis/git/repositories/$REPO_ID?api-version=$API_VERSION")
    echo "Error details: $ERROR_RESPONSE"
  fi
done
echo "Cleanup completed."
