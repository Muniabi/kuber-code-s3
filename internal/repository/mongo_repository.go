package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"kuber-code-s3/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
    client *mongo.Client
    dbName string
}

var (
    ErrDocumentNotFound = errors.New("document not found")
)

// NewMongoRepository создает новый репозиторий для работы с MongoDB
func NewMongoRepository(uri, dbName string) (*MongoRepository, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
    if err != nil {
        return nil, err
    }

    // Проверка подключения
    ctxPing, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancelPing()
    if err = client.Ping(ctxPing, nil); err != nil {
        return nil, err
    }

    return &MongoRepository{
        client: client,
        dbName: dbName,
    }, nil
}

// SaveMetadata сохраняет метаданные файла в MongoDB
func (m *MongoRepository) SaveMetadata(ctx context.Context, metadata *models.FileMetadata) error {
    collection := m.client.Database(m.dbName).Collection("files")

    log.Printf("Saving metadata: %+v", metadata) // Логируем данные перед сохранением

    result, err := collection.InsertOne(ctx, metadata)
    if err != nil {
        log.Printf("MongoDB insert error: %v", err) // Логируем ошибку
        return err
    }

    log.Printf("Inserted document ID: %v", result.InsertedID) // Логируем ID документа
    return nil
}

// GetMetadata возвращает метаданные файла по ID
func (m *MongoRepository) GetMetadata(ctx context.Context, fileID string) (*models.FileMetadata, error) {
    collection := m.client.Database(m.dbName).Collection("files")

    var result models.FileMetadata
    filter := bson.D{{Key: "_id", Value: fileID}}

    err := collection.FindOne(ctx, filter).Decode(&result)
    if err != nil {
        if errors.Is(err, mongo.ErrNoDocuments) {
            return nil, ErrDocumentNotFound
        }
        return nil, err
    }

    return &result, nil
}

// DeleteMetadata удаляет метаданные файла по ID
func (m *MongoRepository) DeleteMetadata(ctx context.Context, fileID string) error {
    collection := m.client.Database(m.dbName).Collection("files")

    filter := bson.D{{Key: "_id", Value: fileID}}
    result, err := collection.DeleteOne(ctx, filter)
    if err != nil {
        return err
    }

    if result.DeletedCount == 0 {
        return ErrDocumentNotFound
    }

    return nil
}

// UpdateMetadata обновляет метаданные файла
func (m *MongoRepository) UpdateMetadata(ctx context.Context, fileID string, metadata *models.FileMetadata) error {
    collection := m.client.Database(m.dbName).Collection("files")

    filter := bson.D{{Key: "_id", Value: fileID}}
    update := bson.D{
        {Key: "$set", Value: bson.D{
            {Key: "original_name", Value: metadata.OriginalName},
            {Key: "file_size", Value: metadata.FileSize},
            {Key: "content_type", Value: metadata.ContentType},
            {Key: "bucket_name", Value: metadata.BucketName},
            {Key: "upload_date", Value: metadata.UploadDate},
            {Key: "url", Value: metadata.URL},
        }},
    }

    opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
    result := collection.FindOneAndUpdate(ctx, filter, update, opts)

    if result.Err() != nil {
        if errors.Is(result.Err(), mongo.ErrNoDocuments) {
            return ErrDocumentNotFound
        }
        return result.Err()
    }

    return nil
}

// Close закрывает подключение к MongoDB
func (m *MongoRepository) Close() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return m.client.Disconnect(ctx)
}