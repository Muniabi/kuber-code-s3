package repository

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioRepository struct {
    client *minio.Client
    Bucket string
}

const (
    defaultRegion       = "us-east-1"
    defaultSecure       = false
    connectionTimeout   = 5 * time.Second
    bucketCheckInterval = 1 * time.Second
)

var (
    ErrFileNotFound     = fmt.Errorf("file not found in storage")
    ErrBucketNotCreated = fmt.Errorf("failed to create bucket")
)

// NewMinioRepository создает новое подключение к Minio и проверяет существование бакета
func NewMinioRepository(endpoint, accessKey, secretKey string, useSSL bool, bucketName string) (*MinioRepository, error) {
    ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
    defer cancel()

    // Инициализация клиента Minio
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: useSSL,
        Region: defaultRegion,
    })
    if err != nil {
        return nil, fmt.Errorf("minio connection error: %w", err)
    }

    // Проверка существования бакета
    exists, err := client.BucketExists(ctx, bucketName)
    if err != nil {
        return nil, fmt.Errorf("bucket check error: %w", err)
    }

    // Создание бакета если не существует
    if !exists {
        err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
            Region: defaultRegion,
        })
        if err != nil {
            return nil, ErrBucketNotCreated
        }
    }

    // Ожидание готовности бакета
    for {
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("bucket readiness check timeout")
        default:
            exists, err = client.BucketExists(ctx, bucketName)
            if exists && err == nil {
                return &MinioRepository{
                    client: client,
                    Bucket: bucketName,
                }, nil
            }
            time.Sleep(bucketCheckInterval)
        }
    }
}

// UploadFile загружает файл в Minio и возвращает URL
func (m *MinioRepository) UploadFile(ctx context.Context, objectName, filePath, contentType string) (string, error) {
    // Загрузка файла
    _, err := m.client.FPutObject(ctx, m.Bucket, objectName, filePath, minio.PutObjectOptions{
        ContentType:  contentType,
        UserMetadata: map[string]string{"x-amz-acl": "public-read"},
    })
    if err != nil {
        return "", fmt.Errorf("upload error: %w", err)
    }

    // Генерация публичного URL
    url := fmt.Sprintf("http://%s/%s/%s", m.client.EndpointURL().Host, m.Bucket, objectName)
    return url, err

    return url, nil
}

// DeleteFile удаляет файл из Minio
func (m *MinioRepository) DeleteFile(ctx context.Context, objectName string) error {
    opts := minio.RemoveObjectOptions{
        GovernanceBypass: true,
        VersionID:       "",
    }

    err := m.client.RemoveObject(ctx, m.Bucket, objectName, opts)
    if err != nil {
        if minioErr, ok := err.(minio.ErrorResponse); ok && minioErr.Code == "NoSuchKey" {
            return ErrFileNotFound
        }
        return fmt.Errorf("delete error: %w", err)
    }

    log.Printf("Successfully deleted %s\n", objectName)
    return nil
}

// GetFileURL возвращает публичный URL файла
func (m *MinioRepository) GetFileURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
    if expires <= 0 {
        expires = 7 * 24 * time.Hour // Дефолтный срок жизни ссылки
    }

    // Исправление 1: Используем url.Values вместо map[string]string
    reqParams := make(url.Values)
    
    // Исправление 2: Проверяем схему URL вместо IsSSL()
    if m.client.EndpointURL().Scheme == "https" {
        reqParams.Set("secure", "true")
    }

    url, err := m.client.PresignedGetObject(ctx, m.Bucket, objectName, expires, reqParams)
    if err != nil {
        return "", fmt.Errorf("url generation error: %w", err)
    }

    return url.String(), nil
}

// HealthCheck проверяет соединение с Minio
func (m *MinioRepository) HealthCheck(ctx context.Context) error {
    _, err := m.client.ListBuckets(ctx)
    return err
}