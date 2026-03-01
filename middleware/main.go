package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"github.com/hyperledger/medical-middleware/internal/api"
	"github.com/hyperledger/medical-middleware/internal/audit"
	"github.com/hyperledger/medical-middleware/internal/config"
	"github.com/hyperledger/medical-middleware/internal/fabric"
	"github.com/hyperledger/medical-middleware/internal/storage"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found, using environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logrus.SetLevel(cfg.LogLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Initialize database
	db, err := storage.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize object storage
	// Map config.S3Config to storage.S3Config (explicit mapping maintains layer separation)
	storageS3Config := storage.S3Config{
		Endpoint:  cfg.S3Config.Endpoint,
		Bucket:    cfg.S3Config.Bucket,
		AccessKey: cfg.S3Config.AccessKey,
		SecretKey: cfg.S3Config.SecretKey,
		Region:    cfg.S3Config.Region,
	}
	objectStorage, err := storage.NewObjectStorage(storageS3Config)
	if err != nil {
		log.Fatalf("Failed to initialize object storage: %v", err)
	}

	// Initialize Fabric gateway
	gateway, err := fabric.NewGateway(cfg.FabricConfig)
	if err != nil {
		log.Fatalf("Failed to initialize Fabric gateway: %v", err)
	}
	defer gateway.Close()

	// Set encryption service database
	// Map config.EncryptionConfig to storage.EncryptionConfig (explicit mapping maintains layer separation)
	storageEncryptionConfig := storage.EncryptionConfig{
		KeyStorePath: cfg.EncryptionConfig.KeyStorePath,
		Algorithm:    cfg.EncryptionConfig.Algorithm,
		KeySize:      cfg.EncryptionConfig.KeySize,
	}
	encryptionService := storage.NewEncryptionService(storageEncryptionConfig)
	encryptionService.SetDatabase(db)

	// Initialize services
	services := initializeServices(cfg, db, objectStorage, gateway, encryptionService)

	// Initialize audit event listener
	eventListener, err := audit.NewEventListener(gateway, services.AuditService, cfg.FabricConfig.ChannelName)
	if err != nil {
		log.Fatalf("Failed to initialize event listener: %v", err)
	}

	// Start event listener in background
	go func() {
		if err := eventListener.Start(context.Background()); err != nil {
			logrus.Errorf("Event listener error: %v", err)
		}
	}()

	// Initialize API router
	router := api.NewRouter(services)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logrus.Infof("Starting API server on %s:%d", cfg.APIHost, cfg.APIPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
	}

	// Stop event listener
	eventListener.Stop()

	logrus.Info("Server exited")
}

func initializeServices(cfg *config.Config, db *storage.Database, objectStorage storage.ObjectStorage, gateway *fabric.Gateway, encryptionService *storage.EncryptionService) *api.Services {
	// Initialize identity service
	identityService := fabric.NewIdentityService(db)

	// Initialize audit service
	auditService := audit.NewAuditService(db)

	// Initialize chaincode services
	medicalRecordsService := fabric.NewMedicalRecordsService(gateway, cfg.FabricConfig.ChannelName, cfg.FabricConfig.ChaincodeNames.MedicalRecords)
	authorizationService := fabric.NewAuthorizationService(gateway, cfg.FabricConfig.ChannelName, cfg.FabricConfig.ChaincodeNames.Authorization)
	sharingService := fabric.NewSharingService(gateway, cfg.FabricConfig.ChannelName, cfg.FabricConfig.ChaincodeNames.AnonymousSharing)

	// Initialize business logic services
	recordService := api.NewRecordService(medicalRecordsService, encryptionService, objectStorage, identityService)
	authService := api.NewAuthorizationService(authorizationService, encryptionService, identityService, auditService)
	shareService := api.NewSharingService(sharingService, authorizationService, encryptionService, objectStorage, identityService)

	return &api.Services{
		RecordService:        recordService,
		AuthorizationService: authService,
		SharingService:       shareService,
		AuditService:         auditService,
		IdentityService:      identityService,
	}
}
