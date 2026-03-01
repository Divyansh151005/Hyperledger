package api

import (
	"github.com/hyperledger/medical-middleware/internal/audit"
	"github.com/hyperledger/medical-middleware/internal/fabric"
	"github.com/hyperledger/medical-middleware/internal/storage"
)

// Services holds all business logic services
type Services struct {
	RecordService      *RecordService
	AuthorizationService *AuthorizationService
	SharingService     *SharingService
	AuditService      *audit.AuditService
	IdentityService   *fabric.IdentityService
}

// RecordService handles medical record operations
type RecordService struct {
	chaincodeService  *fabric.MedicalRecordsService
	encryptionService *storage.EncryptionService
	objectStorage     storage.ObjectStorage
	identityService   *fabric.IdentityService
}

// NewRecordService creates a new record service
func NewRecordService(
	chaincodeService *fabric.MedicalRecordsService,
	encryptionService *storage.EncryptionService,
	objectStorage storage.ObjectStorage,
	identityService *fabric.IdentityService,
) *RecordService {
	return &RecordService{
		chaincodeService:  chaincodeService,
		encryptionService: encryptionService,
		objectStorage:     objectStorage,
		identityService:   identityService,
	}
}

// AuthorizationService handles authorization operations
type AuthorizationService struct {
	chaincodeService  *fabric.AuthorizationService
	encryptionService *storage.EncryptionService
	identityService   *fabric.IdentityService
	auditService      *audit.AuditService
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(
	chaincodeService *fabric.AuthorizationService,
	encryptionService *storage.EncryptionService,
	identityService *fabric.IdentityService,
	auditService *audit.AuditService,
) *AuthorizationService {
	return &AuthorizationService{
		chaincodeService:  chaincodeService,
		encryptionService: encryptionService,
		identityService:   identityService,
		auditService:      auditService,
	}
}

// SharingService handles sharing operations
type SharingService struct {
	chaincodeService     *fabric.SharingService
	authorizationService *fabric.AuthorizationService
	encryptionService    *storage.EncryptionService
	objectStorage        storage.ObjectStorage
	identityService      *fabric.IdentityService
}

// NewSharingService creates a new sharing service
func NewSharingService(
	chaincodeService *fabric.SharingService,
	authorizationService *fabric.AuthorizationService,
	encryptionService *storage.EncryptionService,
	objectStorage storage.ObjectStorage,
	identityService *fabric.IdentityService,
) *SharingService {
	return &SharingService{
		chaincodeService:     chaincodeService,
		authorizationService: authorizationService,
		encryptionService:    encryptionService,
		objectStorage:        objectStorage,
		identityService:      identityService,
	}
}
