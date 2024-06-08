#!/bin/bash

set -e

if [[ -z "$1" ]]; then
    echo "build_image.sh <pkg>"
    exit 1
fi

CI_REGISTRY=${CI_REGISTRY-"ko.local"}
CI_IMAGE=$(basename "$1")
export KO_DOCKER_REPO=${KO_DOCKER_REPO-"$CI_REGISTRY/$CI_IMAGE"}
echo "build image: $KO_DOCKER_REPO"
ko build --bare --sbom none "$1"
