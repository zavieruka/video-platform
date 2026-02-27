package storage

import (
	"context"
	"fmt"
	"time"

	iamcredentials "cloud.google.com/go/iam/credentials/apiv1"
	credentialspb "google.golang.org/genproto/googleapis/iam/credentials/v1"

	"cloud.google.com/go/storage"
	"github.com/zavieruka/video-platform/backend/internal/errors"
)

type VideoStorage interface {
	GenerateSignedUploadURL(ctx context.Context, objectName string, mimeType string, expiryDuration time.Duration) (string, error)
	FileExists(ctx context.Context, objectName string) (bool, error)
	GetFileSize(ctx context.Context, objectName string) (int64, error)
	DeleteFile(ctx context.Context, objectName string) error
	GetPublicURL(objectName string) string
	GetStorageURL(objectName string) string
}

type GCSVideoStorage struct {
	client              *storage.Client
	bucketName          string
	serviceAccountEmail string
}

func NewGCSVideoStorage(client *storage.Client, bucketName string, serviceAccountEmail string) *GCSVideoStorage {
	return &GCSVideoStorage{
		client:              client,
		bucketName:          bucketName,
		serviceAccountEmail: serviceAccountEmail,
	}
}

func (s *GCSVideoStorage) GenerateSignedUploadURL(
	ctx context.Context,
	objectName string,
	mimeType string,
	expiryDuration time.Duration,
) (string, error) {

	iamClient, err := iamcredentials.NewIamCredentialsClient(ctx)
	if err != nil {
		return "", err
	}

	signBytes := func(b []byte) ([]byte, error) {
		req := &credentialspb.SignBlobRequest{
			Name:    "projects/-/serviceAccounts/" + s.serviceAccountEmail,
			Payload: b,
		}

		resp, err := iamClient.SignBlob(ctx, req)
		if err != nil {
			return nil, err
		}

		return resp.SignedBlob, nil
	}

	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		Expires:        time.Now().Add(expiryDuration),
		ContentType:    mimeType,
		GoogleAccessID: s.serviceAccountEmail,
		SignBytes:      signBytes,
	}

	url, err := storage.SignedURL(s.bucketName, objectName, opts)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (s *GCSVideoStorage) FileExists(ctx context.Context, objectName string) (bool, error) {
	bucket := s.client.Bucket(s.bucketName)
	object := bucket.Object(objectName)

	_, err := object.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, errors.NewStorageError("Failed to check file existence", err)
	}

	return true, nil
}

func (s *GCSVideoStorage) GetFileSize(ctx context.Context, objectName string) (int64, error) {
	bucket := s.client.Bucket(s.bucketName)
	object := bucket.Object(objectName)

	attrs, err := object.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return 0, errors.NewNotFoundError("File", objectName)
	}
	if err != nil {
		return 0, errors.NewStorageError("Failed to get file size", err)
	}

	return attrs.Size, nil
}

func (s *GCSVideoStorage) DeleteFile(ctx context.Context, objectName string) error {
	bucket := s.client.Bucket(s.bucketName)
	object := bucket.Object(objectName)

	err := object.Delete(ctx)
	if err != nil && err != storage.ErrObjectNotExist {
		return errors.NewStorageError("Failed to delete file", err)
	}

	return nil
}

// GetPublicURL returns the public URL for accessing a file
func (s *GCSVideoStorage) GetPublicURL(objectName string) string {
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", s.bucketName, objectName)
}

// GetStorageURL returns the gs:// URL for a file
func (s *GCSVideoStorage) GetStorageURL(objectName string) string {
	return fmt.Sprintf("gs://%s/%s", s.bucketName, objectName)
}
