// Package gcsifaces provides interfaces for Google Cloud Storage.
package gcsifaces

import (
	"context"
	"io"
)

// StorageClient represents a GCS client.
type StorageClient interface {
	Bucket(name string) StorageBucket
}

// StorageBucket represents a GCS bucket.
type StorageBucket interface {
	// Object returns a StorageObject for a key.
	Object(key string) StorageObject
}

// StorageObject represents a GCS object.
type StorageObject interface {
	// NewWriter returns a new StorageWriter.
	NewWriter(ctx context.Context) StorageWriter
}

// StorageWriter represents a GCS writer.
type StorageWriter interface {
	io.WriteCloser

	// SetACL sets a pre-defined ACL.
	SetACL(acl string)
}
