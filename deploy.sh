#!/bin/bash
# filepath: ./deploy
# This script builds a Docker image, pushes it to Google Container Registry, and creates a
# Compute Engine instance running your container. It uses default values if parameters aren’t provided.
# Default values are read from cactus-gcp-credentials.json and hardcoded defaults.
#
# Usage:
#   ./deploy [PROJECT_ID] [INSTANCE_NAME] [ZONE]
#
# If no arguments are provided, defaults will be used:
#   PROJECT_ID: cactus-v1-452518
#   INSTANCE_NAME: tango-instance
#   ZONE: us-central1-a

set -e

DEFAULT_PROJECT_ID="cactus-v1-452518"
DEFAULT_INSTANCE_NAME="tango-instance"
DEFAULT_ZONE="us-central1-a"

PROJECT_ID=${1:-$DEFAULT_PROJECT_ID}
INSTANCE_NAME=${2:-$DEFAULT_INSTANCE_NAME}
ZONE=${3:-$DEFAULT_ZONE}

IMAGE_NAME="tango:latest"
FULL_IMAGE_NAME="gcr.io/${PROJECT_ID}/tango:latest"

echo "Using Project ID: ${PROJECT_ID}"
echo "Using Instance Name: ${INSTANCE_NAME}"
echo "Using Zone: ${ZONE}"

echo "Building Docker image..."
docker build -t ${IMAGE_NAME} .

echo "Tagging Docker image..."
docker tag ${IMAGE_NAME} ${FULL_IMAGE_NAME}

echo "Pushing Docker image to Google Container Registry..."
docker push ${FULL_IMAGE_NAME}

echo "Creating a Compute Engine instance with the container..."
gcloud compute instances create-with-container ${INSTANCE_NAME} \
  --container-image=${FULL_IMAGE_NAME} \
  --zone=${ZONE}

echo "Deployment completed!"