#!/bin/bash

# Set your environment variables
export ORGANIZATION="krateo-kog"
export PROJECT_NAME="test-project-1-classic"
export API_VERSION="7.2-preview.2"
#export API_VERSION="7.1"
export USERNAME="your-username"
export PAT="${AZURE_DEVOPS_PAT:-your-personal-access-token}"
export PROXY_BASE_URL="http://localhost:8080"

export PROJECT_ID=11790bc5-82bd-4cdc-b6a6-47bcb7187051
export PARENT_REPO_ID=58877fa0-7bd2-4f23-959a-7e276d0ee87c

echo "Testing GitRepository API..."

echo "1. New repository creation"
curl -s -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_NAME}/git/repositories?api-version=${API_VERSION}" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-basic-'$(date +%s)'"
  }' | jq .

sleep 2

echo -e "\n2. New repository with custom default branch..."
curl -s -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_NAME}/git/repositories?api-version=${API_VERSION}" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-custom-'$(date +%s)'",
    "defaultBranch": "refs/heads/develop"
  }' | jq .

sleep 2

echo -e "\n3. Repository creation with a parent repository (fork) without sourceRef"
curl -s -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_NAME}/git/repositories?api-version=${API_VERSION}" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-fork-'$(date +%s)'",
    "project": {
      "id": "'${PROJECT_ID}'"
    },
    "parentRepository": {
      "id": "'${PARENT_REPO_ID}'",
      "project": {
        "id": "'${PROJECT_ID}'"
      }
    }
  }' | jq .

sleep 2

echo -e "\n4. Repository creation with a parent repository (fork) with sourceRef"
curl -s -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_NAME}/git/repositories?api-version=${API_VERSION}&sourceRef=refs/heads/main" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-fork-'$(date +%s)'",
    "project": {
      "id": "'${PROJECT_ID}'"
    },
    "parentRepository": {
      "id": "'${PARENT_REPO_ID}'",
      "project": {
        "id": "'${PROJECT_ID}'"
      }
    }
  }' | jq .

sleep 2

echo -e "\n5. Testing repository creation with a parent repository (fork) and NOT-EXISTING custom default branch"
# repo is still created but default branch is not set, so it defaults to 'main'
# maybe return 202 
# TODO: add the `message` field (that will go into CR status) to inform the user that the default branch was not set
# then user can change the CR, and the update action od RDC will triggered
curl -s -v -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_NAME}/git/repositories?api-version=${API_VERSION}" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-fork-with-default-'$(date +%s)'",
    "defaultBranch": "refs/heads/test-branch-not-exist",
    "project": {
      "id": "'${PROJECT_ID}'"
    },
    "parentRepository": {
      "id": "'${PARENT_REPO_ID}'",
      "project": {
        "id": "'${PROJECT_ID}'"
      }
    }
  }' 

sleep 2

echo -e "\n6. Testing repository creation with a parent repository (fork) and EXISTING custom default branch"
# Should create the repository successfully with correct default branch because the default branch exists in the parent repository
# Note: The parent repository must have the branch 'test-branch' for this to succeed
curl -s -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_NAME}/git/repositories?api-version=${API_VERSION}" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-fork-with-default-'$(date +%s)'",
    "defaultBranch": "refs/heads/test-branch",
    "project": {
      "id": "'${PROJECT_ID}'"
    },
    "parentRepository": {
      "id": "'${PARENT_REPO_ID}'",
      "project": {
        "id": "'${PROJECT_ID}'"
      }
    }
  }' | jq .

sleep 2

echo -e "\n7. Test error case - missing auth"
curl -s -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_NAME}/git/repositories?api-version=${API_VERSION}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-no-auth",
    "project": {
      "id": "'${PROJECT_ID}'"
    }
  }'

echo -e "\nTesting completed."
