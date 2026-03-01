package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hyperledger/medical-middleware/internal/fabric"
)

// CreateRecordRequest represents a request to create a medical record
type CreateRecordRequest struct {
	PatientID string                 `json:"patient_id" binding:"required"`
	Data      map[string]interface{} `json:"data" binding:"required"`
}

// CreateRecordResponse represents the response after creating a record
type CreateRecordResponse struct {
	RecordID   string `json:"record_id"`
	RecordHash string `json:"record_hash"`
	Status     string `json:"status"`
}

// GetRecordResponse represents a retrieved medical record
type GetRecordResponse struct {
	RecordID   string                 `json:"record_id"`
	PatientID  string                 `json:"patient_id"`
	HospitalID string                 `json:"hospital_id"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  string                 `json:"timestamp"`
	Status     string                 `json:"status"`
}

// CreateRecord creates a new medical record
func (s *RecordService) CreateRecord(ctx context.Context, certificateID string, req CreateRecordRequest) (*CreateRecordResponse, error) {
	// Validate identity (must be Hospital)
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.Role != fabric.RoleHospital {
		return nil, fmt.Errorf("only hospitals can create medical records")
	}

	// Generate record ID
	recordID := uuid.New().String()

	// Serialize medical data
	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encrypt medical record (using a temporary key - in production, use proper key management)
	// For initial creation, we'll encrypt with a hospital-specific key
	encryptedData, err := s.encryptionService.Encrypt(dataJSON, fmt.Sprintf("hospital-%s", identity.MSPID))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Store encrypted data off-chain
	storageKey := fmt.Sprintf("records/%s", recordID)
	if err := s.objectStorage.Put(storageKey, encryptedData); err != nil {
		return nil, fmt.Errorf("failed to store encrypted data: %w", err)
	}

	// Compute hash of encrypted data
	recordHash := s.encryptionService.ComputeHash(encryptedData)

	// Invoke chaincode to create medical record metadata on blockchain
	if err := s.chaincodeService.CreateMedicalRecord(ctx, recordID, req.PatientID, recordHash); err != nil {
		// Rollback: delete stored data
		_ = s.objectStorage.Delete(storageKey)
		return nil, fmt.Errorf("failed to create medical record on blockchain: %w", err)
	}

	return &CreateRecordResponse{
		RecordID:   recordID,
		RecordHash: recordHash,
		Status:     "ACTIVE",
	}, nil
}

// GetRecord retrieves a medical record
func (s *RecordService) GetRecord(ctx context.Context, certificateID, recordID string) (*GetRecordResponse, error) {
	// Validate identity
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	// Get record metadata from blockchain
	recordMetadata, err := s.chaincodeService.GetMedicalRecord(ctx, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get medical record from blockchain: %w", err)
	}

	// Extract fields from metadata
	patientID, _ := recordMetadata["patientID"].(string)
	hospitalID, _ := recordMetadata["hospitalID"].(string)
	timestamp, _ := recordMetadata["timestamp"].(string)
	status, _ := recordMetadata["status"].(string)

	// Check access: Regulator or owning Hospital
	if identity.Role != fabric.RoleRegulator && identity.MSPID != hospitalID {
		return nil, fmt.Errorf("access denied: only regulator or owning hospital can access this record")
	}

	// Retrieve encrypted data from off-chain storage
	storageKey := fmt.Sprintf("records/%s", recordID)
	encryptedData, err := s.objectStorage.Get(storageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve encrypted data: %w", err)
	}

	// Decrypt data
	decryptionKeyID := fmt.Sprintf("hospital-%s", hospitalID)
	decryptedData, err := s.encryptionService.Decrypt(encryptedData, decryptionKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Parse decrypted data
	var data map[string]interface{}
	if err := json.Unmarshal(decryptedData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted data: %w", err)
	}

	return &GetRecordResponse{
		RecordID:   recordID,
		PatientID:  patientID,
		HospitalID: hospitalID,
		Data:       data,
		Timestamp:  timestamp,
		Status:     status,
	}, nil
}
