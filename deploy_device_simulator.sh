#!/bin/bash
# This script builds the device simulator Docker image (targeting linux/amd64),
# pushes it to Google Container Registry (GCR), and creates a Compute Engine instance running the simulator.

set -e

DEFAULT_PROJECT_ID="cactus-v1-452518"
DEFAULT_INSTANCE_NAME="tango-device-simulator"
DEFAULT_ZONE="us-central1-c"
DEFAULT_MACHINE_TYPE="n1-standard-1"

PROJECT_ID=${1:-$DEFAULT_PROJECT_ID}
INSTANCE_NAME=${2:-$DEFAULT_INSTANCE_NAME}
ZONE=${3:-$DEFAULT_ZONE}
MACHINE_TYPE=${4:-$DEFAULT_MACHINE_TYPE}

IMAGE_NAME="tango-device-simulator:latest"
FULL_IMAGE_NAME="gcr.io/${PROJECT_ID}/tango-device-simulator:latest"

TANGO_IP=$(gcloud compute addresses describe tango-static-ip \
  --project=${PROJECT_ID} \
  --region=us-central1 \
  --format="value(address)")

echo "Using Project ID: ${PROJECT_ID}"
echo "Using Instance Name: ${INSTANCE_NAME}"
echo "Using Zone: ${ZONE}"
echo "Using Tango IP: ${TANGO_IP}"

echo "Starting Docker..."
open -a "Docker Desktop" || true

while ! docker info > /dev/null 2>&1; do
  echo "Waiting for Docker daemon to start..."
  sleep 1
done

echo "Docker daemon is running."

echo "Building Docker image for linux/amd64 using Dockerfile.device_simulator..."
docker build --platform linux/amd64 -f Dockerfile.device_simulator -t ${IMAGE_NAME} .

echo "Tagging Docker image..."
docker tag ${IMAGE_NAME} ${FULL_IMAGE_NAME}

echo "Pushing Docker image to Google Container Registry..."
docker push ${FULL_IMAGE_NAME}

echo "Creating Compute Engine instance with the container..."
gcloud compute instances create-with-container ${INSTANCE_NAME} \
    --project=${PROJECT_ID} \
    --machine-type=${MACHINE_TYPE} \
    --container-image=${FULL_IMAGE_NAME} \
    --zone=${ZONE} \
    --service-account=tango-service-acount@${PROJECT_ID}.iam.gserviceaccount.com \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --container-arg="--tango-address=${TANGO_IP}"

echo "Device simulator instance created."
echo "To SSH into the instance, run: gcloud compute ssh ${INSTANCE_NAME} --zone=${ZONE} --project=${PROJECT_ID}"
