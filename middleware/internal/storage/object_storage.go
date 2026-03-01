package storage

import (
	"bytes"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

// ObjectStorage interface for storing medical records
type ObjectStorage interface {
	Put(key string, data []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	Exists(key string) (bool, error)
}

// S3Storage implements ObjectStorage using AWS S3
type S3Storage struct {
	client *s3.S3
	bucket string
}

// S3Config holds S3 configuration
type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

// NewObjectStorage creates a new object storage instance
func NewObjectStorage(config S3Config) (ObjectStorage, error) {
	if config.Endpoint == "" {
		// Return in-memory storage for development/testing
		logrus.Warn("S3 endpoint not configured, using in-memory storage")
		return NewInMemoryStorage(), nil
	}

	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(config.Region),
		Endpoint:    aws.String(config.Endpoint),
		Credentials: credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true), // Required for S3-compatible services
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	return &S3Storage{
		client: s3.New(sess),
		bucket: config.Bucket,
	}, nil
}

// Put stores data in S3
func (s *S3Storage) Put(key string, data []byte) error {
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

// Get retrieves data from S3
func (s *S3Storage) Get(key string) ([]byte, error) {
	result, err := s.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// Delete removes data from S3
func (s *S3Storage) Delete(key string) error {
	_, err := s.client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// Exists checks if an object exists in S3
func (s *S3Storage) Exists(key string) (bool, error) {
	_, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error (404)
		// In production, use proper error type checking
		return false, nil
	}
	return true, nil
}

// InMemoryStorage implements ObjectStorage using in-memory map (for testing/development)
type InMemoryStorage struct {
	storage map[string][]byte
}

// NewInMemoryStorage creates a new in-memory storage
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		storage: make(map[string][]byte),
	}
}

// Put stores data in memory
func (i *InMemoryStorage) Put(key string, data []byte) error {
	i.storage[key] = data
	return nil
}

// Get retrieves data from memory
func (i *InMemoryStorage) Get(key string) ([]byte, error) {
	data, exists := i.storage[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}
	return data, nil
}

// Delete removes data from memory
func (i *InMemoryStorage) Delete(key string) error {
	delete(i.storage, key)
	return nil
}

// Exists checks if a key exists in memory
func (i *InMemoryStorage) Exists(key string) (bool, error) {
	_, exists := i.storage[key]
	return exists, nil
}
