package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"

	"kuber-code-s3/internal/models"
	"kuber-code-s3/internal/repository"
)

var (
    ErrFileNotFound = errors.New("file not found")
    ErrInvalidFile  = errors.New("invalid file")
)

type FileService struct {
    minioRepo *repository.MinioRepository
    mongoRepo *repository.MongoRepository
}

func NewFileService(minio *repository.MinioRepository, mongo *repository.MongoRepository) *FileService {
    return &FileService{
        minioRepo: minio,
        mongoRepo: mongo,
    }
}

func (s *FileService) UploadFile(ctx context.Context, file *multipart.FileHeader) (string, error) {
    // Генерация уникального имени файла
    fileID := uuid.New().String()
    ext := filepath.Ext(file.Filename)
    objectName := fileID + ext
    
    // Сохранение временного файла
    localPath := filepath.Join(os.TempDir(), objectName)
    if err := saveUploadedFile(file, localPath); err != nil {
        return "", err
    }
    defer os.Remove(localPath) // Очистка временного файла

    // Загрузка в Minio
    url, err := s.minioRepo.UploadFile(ctx, objectName, localPath, file.Header.Get("Content-Type"))
    if err != nil {
        return "", err
    }

    // Сохранение метаданных
    metadata := &models.FileMetadata{
        ID:           fileID,
        OriginalName: strings.TrimSuffix(file.Filename, ext),
        FileSize:     file.Size,
        ContentType:  file.Header.Get("Content-Type"),
        BucketName:   s.minioRepo.Bucket,
        UploadDate:   time.Now(),
        URL:          url,
    }

    if err := s.mongoRepo.SaveMetadata(ctx, metadata); err != nil {
        // Откат: удаляем файл из Minio при ошибке сохранения метаданных
        _ = s.minioRepo.DeleteFile(ctx, objectName)
        return "", err
    }

    return url, nil
}

func (s *FileService) DeleteFile(ctx context.Context, fileID string) error {
    // Получение метаданных
    metadata, err := s.mongoRepo.GetMetadata(ctx, fileID)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return ErrFileNotFound
        }
        return err
    }

    // Удаление из Minio
    objectName := fileID + filepath.Ext(metadata.OriginalName)
    if err := s.minioRepo.DeleteFile(ctx, objectName); err != nil {
        return err
    }

    // Удаление метаданных
    return s.mongoRepo.DeleteMetadata(ctx, fileID)
}

func (s *FileService) ReplaceFile(ctx context.Context, fileID string, newFile *multipart.FileHeader) (string, error) {
    // Получение текущих метаданных
    oldMetadata, err := s.mongoRepo.GetMetadata(ctx, fileID)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return "", ErrFileNotFound
        }
        return "", err
    }

    // Удаление старого файла
    oldObjectName := fileID + filepath.Ext(oldMetadata.OriginalName)
    if err := s.minioRepo.DeleteFile(ctx, oldObjectName); err != nil {
        return "", err
    }

    // Загрузка нового файла
    newExt := filepath.Ext(newFile.Filename)
    newObjectName := fileID + newExt
    localPath := filepath.Join(os.TempDir(), newObjectName)
    
    if err := saveUploadedFile(newFile, localPath); err != nil {
        return "", err
    }
    defer os.Remove(localPath)

    // Загрузка в Minio
    url, err := s.minioRepo.UploadFile(ctx, newObjectName, localPath, newFile.Header.Get("Content-Type"))
    if err != nil {
        return "", err
    }

    // Обновление метаданных
    newMetadata := &models.FileMetadata{
        ID:           fileID,
        OriginalName: strings.TrimSuffix(newFile.Filename, newExt),
        FileSize:     newFile.Size,
        ContentType:  newFile.Header.Get("Content-Type"),
        BucketName:   s.minioRepo.Bucket,
        UploadDate:   time.Now(),
        URL:          url,
    }

    if err := s.mongoRepo.UpdateMetadata(ctx, fileID, newMetadata); err != nil {
        _ = s.minioRepo.DeleteFile(ctx, newObjectName)
        return "", err
    }

    return url, nil
}

func (s *FileService) GetFileMetadata(ctx context.Context, fileID string) (*models.FileMetadata, error) {
    return s.mongoRepo.GetMetadata(ctx, fileID)
}

// saveUploadedFile сохраняет загруженный файл во временную директорию
func saveUploadedFile(file *multipart.FileHeader, dst string) error {
    src, err := file.Open()
    if err != nil {
        return err
    }
    defer src.Close()

    if err := os.MkdirAll(filepath.Dir(dst), 0750); err != nil {
        return err
    }

    out, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer out.Close()

    _, err = io.Copy(out, src)
    return err
}