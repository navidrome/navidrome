#!/bin/bash

GIT_TAG="${GITHUB_REF##refs/tags/}"
GIT_BRANCH="${GITHUB_REF##refs/heads/}"
GIT_SHA=$(git rev-parse --short HEAD)
PR_NUM=$(jq --raw-output .pull_request.number "$GITHUB_EVENT_PATH")

DOCKERHUB_LATEST_TAG="--tag ${DOCKER_IMAGE}:latest"
DOCKER_IMAGE="ghrc.io/${DOCKER_IMAGE}"
DOCKER_IMAGE_TAG="--tag ${DOCKER_IMAGE}:sha-${GIT_SHA}"

# Check if it's a pull request
if [[ $PR_NUM != "null" ]]; then
  DOCKER_IMAGE_TAG="${DOCKER_IMAGE_TAG} --tag ${DOCKER_IMAGE}:pr-${PR_NUM}"
fi

# Check if it's a version tag
if [[ $GITHUB_REF != "$GIT_TAG" ]]; then
    # Append the git tag without 'v' and the latest tag to the Docker image tag
    DOCKER_IMAGE_TAG="${DOCKER_IMAGE_TAG} --tag ${DOCKER_IMAGE}:${GIT_TAG#v}  --tag ${DOCKER_IMAGE}:latest"
    DOCKER_IMAGE_TAG="${DOCKER_IMAGE_TAG} ${DOCKERHUB_LATEST_TAG}"
# Check if it's the master branch
elif [[ $GITHUB_REF == "refs/heads/master" ]]; then
    # Append the develop tag to the Docker image tag
    DOCKER_IMAGE_TAG="${DOCKER_IMAGE_TAG} --tag ${DOCKER_IMAGE}:develop"
# Check if it's a feature branch
elif [[ $GIT_BRANCH = feature/* ]]; then
    # Append the branch name with slashes replaced by hyphens to the Docker image tag
    DOCKER_IMAGE_TAG="${DOCKER_IMAGE_TAG} --tag ${DOCKER_IMAGE}:$(echo $GIT_BRANCH | tr / -)"
fi

echo ${DOCKER_IMAGE_TAG}
