# Video Platform Backend - Google Cloud Platform

A scalable, serverless video content distribution platform backend built with Go and Google Cloud Platform services.

## Architecture Overview

This backend follows Google Cloud's recommended architecture for serverless applications.

The system consists of a Cloud Run service handling API requests, with data persistence managed by Firestore for metadata and Cloud Storage for video files. This architecture enables automatic scaling, high availability, and cost-effective operations through serverless infrastructure.

This backend is designed as a standalone service that exposes a REST API. It can be consumed by the accompanying Next.js frontend, mobile applications, or integrated with other backend services.

### GCP Services Used

- **Cloud Run**: Serverless compute platform for containerized applications with automatic scaling
- **Firestore**: NoSQL document database for video metadata storage
- **Cloud Storage**: Object storage for video files with global availability
- **Cloud Build**: Container image building and deployment automation
- **Cloud CDN**: Content delivery network for global video distribution
- **Pub/Sub**: Asynchronous messaging for video processing workflows
- **Transcoder API**: Video transcoding service for format conversion

### Design Rationale

**Cloud Run**
Selected for its serverless architecture, which eliminates infrastructure management overhead. Cloud Run provides automatic scaling from zero to handle variable traffic patterns, built-in HTTPS and health check mechanisms, and seamless regional deployment capabilities.

**Firestore**
Chosen as the metadata store due to its serverless nature, which aligns with the overall architecture philosophy. Firestore offers automatic scaling, real-time synchronization capabilities for future features, and strong consistency guarantees suitable for document-oriented video metadata.

**Cloud Storage**
Purpose-built for large object storage such as video files. Provides seamless integration with Cloud CDN for global content delivery, lifecycle management policies for cost optimization, and eleven nines of durability.

## Project Structure

```
backend/
├── cmd/
│   └── api/              # Main API server entrypoint
│       └── main.go
├── internal/             # Private application code
│   ├── config/          # Configuration management
│   ├── handlers/        # HTTP handlers (controllers)
│   ├── services/        # Business logic layer
│   ├── storage/         # GCS operations
│   ├── database/        # Firestore operations
│   └── middleware/      # HTTP middleware (auth, logging, etc.)
├── pkg/                 # Public libraries (if any)
├── .env.example         # Environment variables template
├── .gitignore
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

## Prerequisites

### Local Development

- Go 1.26 or later
- Google Cloud SDK (`gcloud` CLI)
- A GCP project with billing enabled
- Docker (for local container testing)

### GCP Setup

1. **Install and authenticate gcloud CLI:**

   ```bash
   # Install: https://cloud.google.com/sdk/docs/install

   # Authenticate
   gcloud auth login

   # List projects to check ID
   gcloud projects list

   # Set your project
   gcloud config set project YOUR_PROJECT_ID
   ```

2. **Enable required APIs:**

   ```bash
   gcloud services enable \
     run.googleapis.com \
     firestore.googleapis.com \
     storage.googleapis.com \
     cloudbuild.googleapis.com \
     pubsub.googleapis.com \
     transcoder.googleapis.com
   ```

3. **Create Firestore database:**

   **IMPORTANT**: If you already have a Firestore database in your project or want to use a non-default database name, see the configuration section below.

   ```bash
   # Check if you already have Firestore databases
   gcloud firestore databases list

   # If no databases exist, create the default database
   # Create in Native mode (required for this app)
   gcloud firestore databases create --location=us-central1

   # Optional: Create a named database for this project
   # gcloud firestore databases create --database=video-platform --location=us-central1
   ```

   **Multiple Database Scenario**: If your GCP project uses Firestore for multiple applications, you can create a dedicated database for this project and specify its name in the `FIRESTORE_DATABASE_ID` environment variable. The application defaults to `(default)` if not specified.

4. **Create Cloud Storage buckets:**

   ```bash
   export PROJECT_ID=$(gcloud config get-value project)

   # Source videos bucket (user uploads)
   gsutil mb -l us-central1 gs://${PROJECT_ID}-videos-source

   # Processed videos bucket (transcoded - future)
   gsutil mb -l us-central1 gs://${PROJECT_ID}-videos-processed

   # Enable uniform bucket-level access (recommended)
   gsutil uniformbucketlevelaccess set on gs://${PROJECT_ID}-videos-source
   gsutil uniformbucketlevelaccess set on gs://${PROJECT_ID}-videos-processed
   ```

5. **Create service account for local development:**

   **IMPORTANT**: Signed URL generation requires service account credentials with signing capabilities. User credentials alone cannot sign URLs. We'll use service account impersonation, which is the recommended approach and complies with organizational policies that prohibit service account key generation.

   ```bash
   export PROJECT_ID=$(gcloud config get-value project)

   # Create service account
   gcloud iam service-accounts create video-platform-dev \
       --display-name="Video Platform Development"

   # Grant Storage Admin role (for signed URLs and file operations)
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member="serviceAccount:video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
       --role="roles/storage.admin"

   # Grant Firestore User role (for database operations)
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member="serviceAccount:video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
       --role="roles/datastore.user"

   # Grant Pub/Sub Publisher role (for event publishing)
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member="serviceAccount:video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
       --role="roles/pubsub.publisher"

   # Grant Pub/Sub Subscriber role (for processor service)
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member="serviceAccount:video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
       --role="roles/pubsub.subscriber"

   # Grant Transcoder Admin role (for video processing)
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member="serviceAccount:video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com" \
       --role="roles/transcoder.admin"
   ```

6. **Configure service account impersonation:**

   **Step 1: Grant yourself permission to impersonate the service account**

   ```bash
   # Automatically use your current gcloud account email
   export USER_EMAIL=$(gcloud config get-value account)

   gcloud iam service-accounts add-iam-policy-binding \
       video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com \
       --member="user:${USER_EMAIL}" \
       --role="roles/iam.serviceAccountTokenCreator"
   ```

   **Step 2: Configure Application Default Credentials (ADC) with impersonation**

   ```bash
   gcloud auth application-default login \
       --impersonate-service-account=video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com
   ```

   This will:
   - Open your browser for authentication
   - Save credentials to `~/.config/gcloud/application_default_credentials.json`
   - Configure ADC to impersonate the service account automatically
   - Use short-lived tokens instead of long-lived keys
   - No JSON key file needed
   - Complies with organization policies that prohibit key creation

   **Benefits of this approach:**
   - No service account keys to manage or rotate
   - Complies with `constraints/iam.disableServiceAccountKeyCreation` policy
   - Uses your user credentials with time-limited tokens
   - Automatic token refresh
   - More secure than long-lived key files

7. **Set up Pub/Sub topics and subscriptions:**

   ```bash
   cd scripts
   ./setup-pubsub.sh
   ```

   This script will:
   - Create `video-uploaded` topic for upload events
   - Create `video-processing-complete` topic for completion events
   - Create subscriptions for the processor service
   - Configure appropriate acknowledgment deadlines and retention periods

8. **Set up Transcoder API and template:**

   ```bash
   cd scripts
   ./setup-transcoder.sh
   ```

   This script will:
   - Enable Transcoder API
   - Grant service account `roles/transcoder.admin` permission
   - Create `hls-adaptive-template` for video transcoding
   - Configure template with 1080p, 720p, and 480p resolutions

### Production Deployment Notes

When deploying to Cloud Run, the service account is automatically available through the Cloud Run service identity. No key file is needed in production:

```bash
# Deploy to Cloud Run (production)
gcloud run deploy video-platform-api \
    --source . \
    --region us-central1 \
    --service-account video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com
```

The Cloud Run service will use the service account's identity automatically for signing URLs and accessing GCP services.

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and fill in your values:

```bash
cp .env.example .env
```

Required variables:

- `GCP_PROJECT_ID`: Your GCP project ID
- `GCP_REGION`: Deployment region (e.g., us-central1)
- `SERVICE_ACCOUNT_EMAIL`: Service account email for signed URL generation (e.g., video-platform-dev@your-project-id.iam.gserviceaccount.com)
- `SOURCE_BUCKET_NAME`: Cloud Storage bucket for uploads
- `PROCESSED_BUCKET_NAME`: Cloud Storage bucket for processed videos
- `PORT`: API server port (default: 8080)
- `ENVIRONMENT`: dev/staging/production

Optional variables:

- `FIRESTORE_DATABASE_ID`: Firestore database name (default: "(default)")
- `LOG_LEVEL`: Logging level (default: "info")
- `MAX_UPLOAD_SIZE_MB`: Maximum upload size (default: 500)
- `ALLOWED_VIDEO_FORMATS`: Allowed formats (default: mp4,mov,avi,mkv)
- `UPLOAD_URL_EXPIRY_HOURS`: Signed URL expiry (default: 1)
- `PUBSUB_VIDEO_UPLOADED_TOPIC`: Topic for upload events (default: "video-uploaded")
- `PUBSUB_VIDEO_PROCESSING_COMPLETE_TOPIC`: Topic for processing completion (default: "video-processing-complete")
- `ENABLE_AUTO_PROCESSING`: Enable automatic processing on upload (default: true)
- `TRANSCODER_LOCATION`: Transcoder API location (default: "us-central1")
- `TRANSCODER_TEMPLATE_ID`: Transcoder template ID (default: "hls-adaptive-template")

### Production Deployment

In Cloud Run, environment variables should be set via:

```bash
gcloud run deploy --set-env-vars KEY=VALUE
```

or through the Cloud Console environment variables panel.

**Do not upload a .env file to production.**

### Configuration Validation

The application validates all required configuration on startup and fails fast with clear error messages if misconfigured.

## Getting Started

### 1. Clone and Install Dependencies

```bash
cd backend
go mod download
```

### 2. Set Up Environment

```bash
cp .env.example .env
# Edit .env with your values
# Make sure to set SERVICE_ACCOUNT_EMAIL to your service account email
# Format: video-platform-dev@your-project-id.iam.gserviceaccount.com
```

### 3. Configure Service Account Impersonation

Make sure you've completed steps 5 and 6 from GCP Setup above. Verify the impersonation is configured:

```bash
# Check the ADC configuration
cat ~/.config/gcloud/application_default_credentials.json | grep service_account_impersonation_url

# Should show: "service_account_impersonation_url": "https://iamcredentials.googleapis.com/..."
```

### 4. Run Locally

**API Server:**
```bash
go run cmd/api/main.go
```

The API will start on `http://localhost:8080` (or your configured PORT).

**Processor Service (for automatic video processing):**

In a separate terminal:
```bash
go run cmd/processor/main.go
```

The processor listens for video upload events and automatically triggers transcoding jobs.

**Note:** Both services can run simultaneously. The API handles uploads, the processor handles transcoding.

### 5. Test the Health Check

```bash
curl http://localhost:8080/health
```

Expected response:

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "0.1.0",
  "environment": "dev"
}
```

## API Endpoints

### Health & Status

- `GET /health` - Health check endpoint
- `GET /ready` - Readiness probe (checks GCP client connections)

### Videos

- `POST /api/v1/videos/upload-url` - Generate signed upload URL
- `POST /api/v1/videos/{id}/confirm` - Confirm successful upload
- `POST /api/v1/videos/{id}/fail` - Mark upload as failed
- `GET /api/v1/videos/{id}` - Get video details
- `GET /api/v1/videos` - List videos (paginated)
- `DELETE /api/v1/videos/{id}` - Delete video and associated storage objects

## Architecture Decisions

### Standalone REST API Design

This backend is intentionally decoupled from any specific frontend implementation. It exposes standard REST endpoints that can be consumed by:

- The Next.js frontend in this repository
- Mobile applications (iOS, Android)
- Other backend services
- Third-party integrations
- CLI tools

All responses follow consistent JSON formatting and standard HTTP status codes.

### Signed URL Upload Strategy

Videos are uploaded directly to Cloud Storage using signed URLs rather than passing through the backend. This approach:

- Reduces Cloud Run compute and memory costs
- Eliminates request timeout concerns for large files
- Scales better as Cloud Storage handles upload traffic
- Provides better performance for end users

### Event-Driven Video Processing

The system uses an event-driven architecture for video processing:

**Workflow:**
```
1. Upload Confirmed → Pub/Sub Event Published
2. Processor Service Receives Event
3. Transcoder Job Submitted (3 resolutions + HLS manifest)
4. Job Monitoring (polls every 15 seconds)
5. Firestore Updated with Processed Video URLs
6. Status: "ready" (available for streaming)
```

**Benefits:**
- **Decoupled**: API and processor are independent services
- **Reliable**: Pub/Sub ensures message delivery with retries
- **Scalable**: Multiple processor instances can run concurrently
- **Asynchronous**: Upload confirmation returns immediately
- **Fault-tolerant**: Failed jobs are tracked and can be retried

**Components:**
- **API Server** (`cmd/api`): Handles HTTP requests, publishes events
- **Processor Service** (`cmd/processor`): Subscribes to events, manages transcoding
- **Pub/Sub**: Message queue for event delivery
- **Transcoder API**: Converts videos to multiple resolutions
- **Firestore**: Stores metadata and processing status

### Internal Package Structure

Following Go best practices, we use `internal/` for private application code that shouldn't be imported by other projects. This enforces encapsulation.

### Service Layer Pattern

Separating business logic from HTTP handlers:

- Easier testing (mock services vs handlers)
- Reusable logic across different handlers
- Clearer separation of concerns
- Follows Clean Architecture principles

### Centralized Configuration

All configuration is managed through the `config` package:

- Single source of truth
- Validation at startup (fail fast)
- Easy to test different configurations
- Type-safe access to config values

### Error Handling Strategy

- Use custom error types for different scenarios
- Wrap errors with context using `fmt.Errorf` with `%w`
- Log errors with appropriate severity
- Return user-friendly error messages without exposing internals

## Best Practices Implemented

### 12-Factor App Principles

- Configuration via environment variables
- Explicit dependencies (go.mod)
- Stateless processes
- Logs to stdout

### Google Cloud Best Practices

- Service accounts with minimal permissions
- Regional deployment (can expand to multi-region)
- Structured logging for Cloud Logging
- Health check endpoints for Cloud Run
- Support for multiple Firestore databases
- Signed URLs for secure, time-limited access

### Go Best Practices

- Standard project layout
- Context propagation
- Proper error handling
- Clear package organization

## Cost Optimization Considerations

- **Cloud Run**: Scales to zero when not in use
- **Firestore**: Optimize queries to minimize document reads/writes
- **Cloud Storage**: Use lifecycle policies (Nearline/Coldline for old videos)
- **Signed URLs**: Files don't pass through Cloud Run, reducing compute costs
- **CDN**: Reduces egress costs (planned implementation)

## Security Considerations

- Never commit `.env` file (included in .gitignore)
- Use service account impersonation instead of key files
- Service accounts configured with minimal IAM permissions
- Enable uniform bucket-level access on Cloud Storage buckets
- Validate all user inputs
- Time-limited signed URLs (1 hour expiry)
- HTTPS only (enforced by Cloud Run)
- Support for named Firestore databases prevents conflicts
- Complies with `constraints/iam.disableServiceAccountKeyCreation` organizational policy

## Troubleshooting

### Signed URL Generation Error

If you see an error about "unable to detect default GoogleAccessID":

```
storage: unable to detect default GoogleAccessID
```

This means service account impersonation is not properly configured. Follow steps 5 and 6 in the GCP Setup section above.

Verify impersonation is active:
```bash
cat ~/.config/gcloud/application_default_credentials.json | grep service_account_impersonation_url
```

If not configured, run:
```bash
gcloud auth application-default login \
    --impersonate-service-account=video-platform-dev@${PROJECT_ID}.iam.gserviceaccount.com
```

### Permission Denied Errors

Ensure your service account has the correct IAM roles:
- `roles/storage.admin` for Cloud Storage operations
- `roles/datastore.user` for Firestore operations
- `roles/pubsub.publisher` for Pub/Sub event publishing
- `roles/pubsub.subscriber` for Pub/Sub event subscription (processor service)
- `roles/transcoder.admin` for Transcoder API operations (processor service)

### Processor Service Not Receiving Events

**Check subscription exists:**
```bash
gcloud pubsub subscriptions describe video-processor-sub
```

If not found, run:
```bash
cd scripts
./setup-pubsub.sh
```

**Check service account permissions:**
```bash
gcloud projects get-iam-policy $(gcloud config get-value project) \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:video-platform-dev*" \
  --format="table(bindings.role)"
```

Should include `roles/pubsub.subscriber`.

### Video Stuck in "processing" Status

**Check processor is running:**
```bash
ps aux | grep "go run cmd/processor"
```

**Check transcoder template exists:**
```bash
gcloud transcoder templates describe hls-adaptive-template --location=us-central1
```

If not found, run:
```bash
cd scripts
./setup-transcoder.sh
```

**Check job status manually:**
```bash
# Get job ID from video metadata
curl http://localhost:8080/api/v1/videos/<videoId> | jq '.processingStatus.jobId'

# Check job status
gcloud transcoder jobs describe <jobId> --location=us-central1
```

## Contributing

This is a learning project following GCP Professional Cloud Architect best practices. Contributions are welcome through issues and pull requests.

## Resources

- [Google Cloud Architecture Center](https://cloud.google.com/architecture)
- [Cloud Run Best Practices](https://cloud.google.com/run/docs/tips/general)
- [Firestore Best Practices](https://cloud.google.com/firestore/docs/best-practices)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
- [Firestore Multi-Database Support](https://cloud.google.com/firestore/docs/manage-databases)
- [Cloud Storage Signed URLs](https://cloud.google.com/storage/docs/access-control/signed-urls)