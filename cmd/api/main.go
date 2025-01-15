// @title Example Go Project API
// @version 1.0
// @description A RESTful API server with user authentication and MongoDB integration
// @termsOfService https://mywebideal.work

// @contact.name API Support
// @contact.email champuplove@gmail.com

// @host ${DOMAIN}
// @BasePath /api/v1
// @schemes http https
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Enter the token with the `Bearer: ` prefix, e.g. "Bearer abcde12345".

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"

	"example-go-project/docs"
	"example-go-project/internal/handlers"
	"example-go-project/internal/repository"
	"example-go-project/internal/routers"
	"example-go-project/internal/service"
	"example-go-project/pkg/config"
	"example-go-project/pkg/database"
	"example-go-project/pkg/utils"
)

func setupMongoDB(cfg *config.Config) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := database.ConnectMongoDB(cfg.MongoDBURI)
	if err != nil {
		return nil, err
	}

	// Verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Println("Connected to MongoDB successfully")
	return client, nil
}

func setupServer(cfg *config.Config) (*routers.Application, error) {
	// Set Gin mode to release
	gin.SetMode(gin.DebugMode)

	docs.UpdateSwaggerHost(cfg.ServerHost, cfg.ServerPort)

	allowCredentials := false
	if cfg.ServerState == "production" {
		gin.SetMode(gin.ReleaseMode)
		allowCredentials = true
	}

	// Create Gin router
	router := gin.Default()

	utils.SetupValidator()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: allowCredentials,
		MaxAge:           12 * time.Hour,
	}))

	// Setup MongoDB
	mongoClient, err := setupMongoDB(cfg)
	if err != nil {
		return nil, err
	}

	// Initialize repositories
	db := mongoClient.Database(cfg.MongoDBDatabase)
	userRepo := repository.NewUserRepository(db)
	// productRepo := repository.NewProductRepository(db, userRepo)
	// serviceRepo := repository.NewHttpServiceRepository()
	// fileRepo := repository.NewLocalFileRepository(db)

	// Initialize services
	// fileService := service.NewFileService(fileRepo)
	userService := service.NewUserService(userRepo)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService)
	// productHandler := handlers.NewProductHandler(productRepo, userRepo)
	// pingHandler := handlers.NewPingHandler(serviceRepo)
	// uploadHandler := handlers.NewUploadHandler(fileService, userRepo)

	// Create application instance with all dependencies
	application := &routers.Application{
		Router:         router,
		UserHandler:    userHandler,
		// ProductHandler: productHandler,
		// PingHandler:    pingHandler,
		// UploadHandler:  uploadHandler,
		Config:         cfg,
	}

	// Setup routes
	application.SetupRoutes()

	return application, nil
}

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Setup server with all dependencies
	application, err := setupServer(cfg)
	if err != nil {
		log.Fatal("Failed to setup server:", err)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    cfg.ServerHost + ":" + cfg.ServerPort,
		Handler: application.Router,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s:%s", cfg.ServerHost, cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give the server 5 seconds to finish current requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}