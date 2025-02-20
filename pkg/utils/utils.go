package utils

import "github.com/google/uuid"

func GenerateFileID() string {
    return uuid.New().String()
}