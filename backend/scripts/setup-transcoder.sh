#!/bin/bash
set -e

echo "========================================="
echo "Video Platform - Transcoder API Setup"
echo "========================================="
echo ""

PROJECT_ID=$(gcloud config get-value project)

if [ -z "$PROJECT_ID" ]; then
    echo "Error: No GCP project configured. Please run:"
    echo "  gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

echo "Project ID: $PROJECT_ID"
echo ""

echo "Enabling Transcoder API..."
gcloud services enable transcoder.googleapis.com --project=$PROJECT_ID

echo ""
echo "Granting service account permissions..."

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
    --role="roles/transcoder.admin"

echo ""
echo "Creating transcoder template..."

gcloud transcoder templates create hls-adaptive-template \
    --location=us-central1 \
    --file=internal/config/transcoder/hls_adaptive.json

echo ""
echo "========================================="
echo "Transcoder API Setup Complete!"
echo "========================================="
echo ""
echo "Resources created:"
echo "  ✓ Transcoder API enabled"
echo "  ✓ Service account permissions granted"
echo "  ✓ HLS adaptive template created"
echo ""