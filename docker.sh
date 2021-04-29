#!/usr/bin/env bash
SRC_PATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

DOCKER_IMAGE=localhost:32000/proxy:latest
docker build -f "${SRC_PATH}/docker/Dockerfile" -t ${DOCKER_IMAGE} .
docker push ${DOCKER_IMAGE}
