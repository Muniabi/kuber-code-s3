// @title File Storage API
// @version 1.0
// @description Secure file storage service with Minio and MongoDB
// @host localhost:8080
// @BasePath /api/v1
// @schemes http

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
package main

import (
	"kuber-code-s3/internal/config"
	"kuber-code-s3/internal/handler"
	"kuber-code-s3/internal/repository"
	"kuber-code-s3/internal/service"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_  "kuber-code-s3/docs"
)

// @title           File Storage Service API
// @version         1.0
// @description     Secure microservice for storing and managing files with Minio and MongoDB
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@filestorage.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        Authorization
func main() {
	// Установка режима Release
    gin.SetMode(gin.ReleaseMode)

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Minio repository
	minioRepo, err := repository.NewMinioRepository(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioSSL,
		"user-uploads",
	)
	if err != nil {
		log.Fatalf("Failed to initialize Minio client: %v", err)
	}

	// Initialize MongoDB repository
	mongoRepo, err := repository.NewMongoRepository(cfg.MongoURI, cfg.MongoDatabase)
	if err != nil {
		log.Fatalf("Failed to initialize MongoDB client: %v", err)
	}

	// Create services
	fileService := service.NewFileService(minioRepo, mongoRepo)

	// Create handlers
	fileHandler := handler.NewFileHandler(fileService)

	// Setup Gin router
	router := gin.Default()

	router.MaxMultipartMemory = 1024 << 20 // 1 GB

	// Доверяем только локальному прокси
	router.SetTrustedProxies([]string{"127.0.0.1"})

	// CORS configuration
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// API routes
	api := router.Group("/api/v1")
	{
		// Authentication middleware
		api.Use(apiKeyAuth())

		// File operations
		api.POST("/upload", fileHandler.UploadFile)
		api.GET("/files/:id", fileHandler.GetFileMetadata)
		api.PUT("/files/:id", fileHandler.ReplaceFile)
		api.DELETE("/files/:id", fileHandler.DeleteFile)
	}

	// Swagger documentation
	if os.Getenv("GIN_MODE") != "release" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// router.GET("/swagger/*", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// apiKeyAuth middleware для проверки API ключа
func apiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("Authorization")
		if apiKey != os.Getenv("API_KEY") {
			c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	}
}