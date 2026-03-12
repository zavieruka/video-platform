#!/bin/bash
set -e

echo "========================================="
echo "Video Platform - Pub/Sub Setup"
echo "========================================="
echo ""

# Get project ID
PROJECT_ID=$(gcloud config get-value project)

if [ -z "$PROJECT_ID" ]; then
    echo "Error: No GCP project configured. Please run:"
    echo "  gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

echo "Project ID: $PROJECT_ID"
echo ""

# Enable Pub/Sub API
echo "Enabling Pub/Sub API..."
gcloud services enable pubsub.googleapis.com --project=$PROJECT_ID

echo ""
echo "Creating Pub/Sub topics..."

# Create video-uploaded topic
echo "  - Creating topic: video-uploaded"
if gcloud pubsub topics describe video-uploaded --project=$PROJECT_ID &>/dev/null; then
    echo "    ✓ Topic already exists"
else
    gcloud pubsub topics create video-uploaded --project=$PROJECT_ID
    echo "    ✓ Topic created"
fi

# Create video-processing-complete topic
echo "  - Creating topic: video-processing-complete"
if gcloud pubsub topics describe video-processing-complete --project=$PROJECT_ID &>/dev/null; then
    echo "    ✓ Topic already exists"
else
    gcloud pubsub topics create video-processing-complete --project=$PROJECT_ID
    echo "    ✓ Topic created"
fi

echo ""
echo "Creating Pub/Sub subscriptions..."

# Create subscription for video processor
echo "  - Creating subscription: video-processor-sub"
if gcloud pubsub subscriptions describe video-processor-sub --project=$PROJECT_ID &>/dev/null; then
    echo "    ✓ Subscription already exists"
else
    gcloud pubsub subscriptions create video-processor-sub \
        --topic=video-uploaded \
        --ack-deadline=600 \
        --message-retention-duration=7d \
        --project=$PROJECT_ID
    echo "    ✓ Subscription created"
fi

# Create subscription for completion handler
echo "  - Creating subscription: video-completion-sub"
if gcloud pubsub subscriptions describe video-completion-sub --project=$PROJECT_ID &>/dev/null; then
    echo "    ✓ Subscription already exists"
else
    gcloud pubsub subscriptions create video-completion-sub \
        --topic=video-processing-complete \
        --ack-deadline=60 \
        --message-retention-duration=7d \
        --project=$PROJECT_ID
    echo "    ✓ Subscription created"
fi

echo ""
echo "========================================="
echo "Pub/Sub Setup Complete!"
echo "========================================="
echo ""
echo "Topics created:"
echo "  ✓ video-uploaded"
echo "  ✓ video-processing-complete"
echo ""
echo "Subscriptions created:"
echo "  ✓ video-processor-sub (for processing service)"
echo "  ✓ video-completion-sub (for completion handler)"
echo ""