package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NewRouter creates a new API router
func NewRouter(services *Services) *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", healthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Medical Records
		v1.POST("/records", authenticateMiddleware(services), createRecord(services))
		v1.GET("/records/:recordID", authenticateMiddleware(services), getRecord(services))

		// Authorization
		v1.POST("/authorizations/request", authenticateMiddleware(services), requestAccess(services))
		v1.POST("/authorizations/:requestID/approve/hospital", authenticateMiddleware(services), approveByHospital(services))
		v1.POST("/authorizations/:requestID/approve/patient", authenticateMiddleware(services), approveByPatient(services))
		v1.POST("/authorizations/:requestID/revoke", authenticateMiddleware(services), revokeAuthorization(services))
		v1.GET("/authorizations/:requestID", authenticateMiddleware(services), getAuthorization(services))

		// Sharing
		v1.POST("/sharing/share", authenticateMiddleware(services), shareRecord(services))
		v1.GET("/sharing/:recordID/:authRequestID", authenticateMiddleware(services), getSharedRecord(services))

		// Audit
		v1.GET("/audit/logs", authenticateMiddleware(services), getAuditLogs(services))
	}

	return router
}

// healthCheck handles health check requests
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// authenticateMiddleware extracts and validates certificate ID from request
// In production, this should extract from X.509 certificate or JWT token
func authenticateMiddleware(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract certificate ID from header (in production, validate X.509 cert)
		certificateID := c.GetHeader("X-Certificate-ID")
		if certificateID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "missing certificate ID",
			})
			c.Abort()
			return
		}

		// Validate identity exists
		identity, err := services.IdentityService.GetIdentity(certificateID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid certificate ID",
			})
			c.Abort()
			return
		}

		// Store identity in context
		c.Set("certificateID", certificateID)
		c.Set("identity", identity)
		c.Next()
	}
}

// Handlers

func createRecord(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)

		var req CreateRecordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := services.RecordService.CreateRecord(c.Request.Context(), certificateID, req)
		if err != nil {
			logrus.Errorf("Failed to create record: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

func getRecord(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)
		recordID := c.Param("recordID")

		resp, err := services.RecordService.GetRecord(c.Request.Context(), certificateID, recordID)
		if err != nil {
			logrus.Errorf("Failed to get record: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func requestAccess(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)

		var req RequestAccessRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := services.AuthorizationService.RequestAccess(c.Request.Context(), certificateID, req)
		if err != nil {
			logrus.Errorf("Failed to request access: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

func approveByHospital(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)
		requestID := c.Param("requestID")

		req := ApproveRequest{RequestID: requestID}
		if err := services.AuthorizationService.ApproveByHospital(c.Request.Context(), certificateID, req); err != nil {
			logrus.Errorf("Failed to approve by hospital: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "approved"})
	}
}

func approveByPatient(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)
		requestID := c.Param("requestID")

		req := ApproveRequest{RequestID: requestID}
		if err := services.AuthorizationService.ApproveByPatient(c.Request.Context(), certificateID, req); err != nil {
			logrus.Errorf("Failed to approve by patient: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "approved"})
	}
}

func revokeAuthorization(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)
		requestID := c.Param("requestID")

		req := ApproveRequest{RequestID: requestID}
		if err := services.AuthorizationService.RevokeAuthorization(c.Request.Context(), certificateID, req); err != nil {
			logrus.Errorf("Failed to revoke authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "revoked"})
	}
}

func getAuthorization(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)
		requestID := c.Param("requestID")

		resp, err := services.AuthorizationService.GetAuthorization(c.Request.Context(), certificateID, requestID)
		if err != nil {
			logrus.Errorf("Failed to get authorization: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func shareRecord(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)

		var req ShareRecordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := services.SharingService.ShareRecord(c.Request.Context(), certificateID, req)
		if err != nil {
			logrus.Errorf("Failed to share record: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

func getSharedRecord(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		certificateID := c.MustGet("certificateID").(string)
		recordID := c.Param("recordID")
		authRequestID := c.Param("authRequestID")

		resp, err := services.SharingService.GetSharedRecord(c.Request.Context(), certificateID, recordID, authRequestID)
		if err != nil {
			logrus.Errorf("Failed to get shared record: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func getAuditLogs(services *Services) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventType := c.Query("event_type")
		chaincodeName := c.Query("chaincode_name")
		limit := c.DefaultQuery("limit", "100")
		offset := c.DefaultQuery("offset", "0")

		logs, err := services.AuditService.GetAuditLogs(eventType, chaincodeName, nil, nil, parseInt(limit), parseInt(offset))
		if err != nil {
			logrus.Errorf("Failed to get audit logs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, logs)
	}
}

// Helper function
func parseInt(s string) int {
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}
