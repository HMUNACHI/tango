#!/bin/bash
# filepath: /Users/henry/Desktop/tango/deploy.sh
# This script builds a Docker image (targeting linux/amd64),
# pushes it to Google Container Registry, and creates a Compute Engine instance.
# Defaults: PROJECT_ID: tango-v1, INSTANCE_NAME: tango-instance, ZONE: us-central1-a

set -e

DEFAULT_PROJECT_ID="tango-v1"
DEFAULT_INSTANCE_NAME="tango-v1"
DEFAULT_ZONE="us-central1-c"
DEFAULT_MACHINE_TYPE="c2d-standard-2"

PROJECT_ID=${1:-$DEFAULT_PROJECT_ID}
INSTANCE_NAME=${2:-$DEFAULT_INSTANCE_NAME}
ZONE=${3:-$DEFAULT_ZONE}
MACHINE_TYPE=${4:-$DEFAULT_MACHINE_TYPE}

IMAGE_NAME="tango:latest"
FULL_IMAGE_NAME="gcr.io/${PROJECT_ID}/tango:latest"

STATIC_IP=$(gcloud compute addresses describe tango-static-ip \
  --project=${PROJECT_ID} \
  --region=us-central1 \
  --format="value(address)")

echo "Using Project ID: ${PROJECT_ID}"
echo "Using Instance Name: ${INSTANCE_NAME}"
echo "Using Zone: ${ZONE}"

echo "Starting Docker..."
open -a "Docker Desktop"

while ! docker info > /dev/null 2>&1; do
  echo "Waiting for Docker daemon to start..."
  sleep 1
done

echo "Docker daemon is running."

echo "Building Docker image for linux/amd64..."
docker build --platform linux/amd64 -t ${IMAGE_NAME} .

echo "Tagging Docker image..."
docker tag ${IMAGE_NAME} ${FULL_IMAGE_NAME}

echo "Pushing Docker image to Google Container Registry..."
docker push ${FULL_IMAGE_NAME}

echo "Creating a Compute Engine instance with the container..."
gcloud compute instances create-with-container ${INSTANCE_NAME} \
    --project=${PROJECT_ID} \
    --machine-type=${MACHINE_TYPE} \
    --container-image=${FULL_IMAGE_NAME} \
    --zone=${ZONE} \
    --address=${STATIC_IP} \
    --service-account=tango-service-acount@tango-v1-452518.iam.gserviceaccount.com \
    --scopes=https://www.googleapis.com/auth/cloud-platform \
    --tags=http-server,https-server,grpc-server \
    --create-disk=auto-delete=yes,device-name=tango,image=projects/debian-cloud/global/images/debian-12-bookworm-v20250212,mode=rw,size=10,type=pd-balanced

# echo "Creating a firewall rule to allow gRPC traffic..."
# gcloud compute firewall-rules create allow-grpc \
#     --allow=tcp:50051 \
#     --target-tags=grpc-server \
#     --project=${PROJECT_ID}


echo "Instance created. You can access it at http://${STATIC_IP}:50051"
echo "To SSH into the instance, run: gcloud compute ssh ${INSTANCE_NAME} --zone=${ZONE} --project=${PROJECT_ID}"
echo "To delete the instance, run: gcloud compute instances delete ${INSTANCE_NAME} --zone=${ZONE} --project=${PROJECT_ID}"
echo "To delete the static IP, run: gcloud compute addresses delete tango-static-ip --region=us-central1 --project=${PROJECT_ID}"