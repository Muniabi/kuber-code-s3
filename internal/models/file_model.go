package models

import "time"

type FileMetadata struct {
    ID          string    `bson:"_id"`
    OriginalName string   `bson:"original_name"`
    FileSize    int64     `bson:"file_size"`
    ContentType string    `bson:"content_type"`
    BucketName  string    `bson:"bucket_name"`
    UploadDate  time.Time `bson:"upload_date"`
    URL         string    `bson:"url"`
}