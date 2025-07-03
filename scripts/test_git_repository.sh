#!/bin/bash

# Set your environment variables
export ORGANIZATION="krateo-kog"
export PROJECT_ID="test-project-1-classic"
export API_VERSION="7.2-preview.2"
export USERNAME="your-username"
export PAT="${AZURE_DEVOPS_PAT:-your-personal-access-token}"
export PROXY_BASE_URL="http://localhost:8080"

echo "Testing GitRepository API..."

echo "1. Testing basic repository creation..."
curl -s -v -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_ID}/git/repositories?api-version=${API_VERSION}" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-basic-'$(date +%s)'"
  }'

echo -e "\n2. Testing custom default branch..."
curl -s -X POST \
  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_ID}/git/repositories?api-version=${API_VERSION}" \
  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-repo-custom-'$(date +%s)'",
    "defaultBranch": "refs/heads/develop"
  }' | jq .

#echo -e "\n3. Testing repository creation with a parent repository (fork)..."
#curl -s -X POST \
#  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_ID}/git/repositories?api-version=${API_VERSION}" \
#  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
#  -H "Content-Type: application/json" \
#  -d '{
#    "name": "test-repo-fork",
#    "project": {
#      "id": "'${PROJECT_ID}'"
#    },
#    "parentRepository": {
#      "id": "parent-repo-id",
#      "project": {
#        "id": "'${PROJECT_ID}'"
#      }
#    }
#  }' | jq .
#

#echo -e "\n3. Testing repository creation with a parent repository (fork) and custom default branch..."
#curl -s -X POST \
#  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_ID}/git/repositories?api-version=${API_VERSION}" \
#  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
#  -H "Content-Type: application/json" \
#  -d '{
#    "name": "test-repo-fork",
#    "defaultBranch": "refs/heads/test-branch-1",
#    "project": {
#      "id": "'${PROJECT_ID}'"
#    },
#    "parentRepository": {
#      "id": "parent-repo-id",
#      "project": {
#        "id": "'${PROJECT_ID}'"
#      }
#    }
#  }' | jq .
#
#echo -e "\n4. Testing error case - missing auth..."
#curl -s -X POST \
#  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_ID}/git/repositories?api-version=${API_VERSION}" \
#  -H "Content-Type: application/json" \
#  -d '{
#    "name": "test-repo-no-auth",
#    "project": {
#      "id": "'${PROJECT_ID}'"
#    }
#  }'
#
#
#echo -e "\n5. Testing error case - invalid repository name..."
#curl -s -X POST \
#  "${PROXY_BASE_URL}/api/${ORGANIZATION}/${PROJECT_ID}/git/repositories?api-version=${API_VERSION}" \
#  -H "Authorization: Basic $(echo -n "${USERNAME}:${PAT}" | base64)" \
#  -H "Content-Type: application/json" \
#  -d '{
#    "name": "test-repo-invalid",
#    "project": {
#      "id": "'${PROJECT_ID}'"
#    },
#    "invalidField": "value"
#  }' | jq .
#
#
echo -e "\nTesting completed."
