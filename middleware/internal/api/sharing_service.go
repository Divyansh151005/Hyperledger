package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/medical-middleware/internal/fabric"
)

// ShareRecordRequest represents a request to share a record
type ShareRecordRequest struct {
	RecordID              string `json:"record_id" binding:"required"`
	AuthorizationRequestID string `json:"authorization_request_id" binding:"required"`
}

// ShareRecordResponse represents the response after sharing
type ShareRecordResponse struct {
	RecordID              string `json:"record_id"`
	AuthorizationRequestID string `json:"authorization_request_id"`
	Status                string `json:"status"`
}

// GetSharedRecordResponse represents a retrieved shared record
type GetSharedRecordResponse struct {
	RecordID              string                 `json:"record_id"`
	AuthorizationRequestID string                `json:"authorization_request_id"`
	Data                  map[string]interface{} `json:"data"`
	SharedAt              string                 `json:"shared_at"`
}

// ShareRecord shares a medical record with authorized research organization
func (s *SharingService) ShareRecord(ctx context.Context, certificateID string, req ShareRecordRequest) (*ShareRecordResponse, error) {
	// Validate identity (must be Hospital)
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.Role != fabric.RoleHospital {
		return nil, fmt.Errorf("only hospitals can share records")
	}

	// Get authorization request to verify it's GRANTED
	authRequest, err := s.authorizationService.GetAuthorization(ctx, req.AuthorizationRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization request: %w", err)
	}

	status, _ := authRequest["status"].(string)
	if status != "GRANTED" {
		return nil, fmt.Errorf("authorization request must be GRANTED, current status: %s", status)
	}

	recordID, _ := authRequest["recordID"].(string)
	if recordID != req.RecordID {
		return nil, fmt.Errorf("authorization request recordID (%s) does not match provided recordID (%s)", recordID, req.RecordID)
	}

	hospitalID, _ := authRequest["hospitalID"].(string)
	if identity.MSPID != hospitalID {
		return nil, fmt.Errorf("only the owning hospital can share this record")
	}

	// Create encrypted pointer (encrypt the storage location using authorization-specific key)
	// The encrypted data is already stored off-chain, we just need the storage key for the pointer
	storageKey := fmt.Sprintf("records/%s", req.RecordID)
	pointerData := map[string]interface{}{
		"storage_key": storageKey,
		"record_id":   req.RecordID,
	}
	pointerJSON, err := json.Marshal(pointerData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pointer: %w", err)
	}

	// Encrypt pointer using key for this authorization request
	encryptedPointer, err := s.encryptionService.Encrypt(pointerJSON, req.AuthorizationRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt pointer: %w", err)
	}

	// Convert encrypted pointer to base64 for storage in chaincode
	encryptedPointerBase64 := fmt.Sprintf("%x", encryptedPointer)

	// Invoke chaincode to share record
	if err := s.chaincodeService.ShareRecord(ctx, req.RecordID, req.AuthorizationRequestID, encryptedPointerBase64); err != nil {
		return nil, fmt.Errorf("failed to share record: %w", err)
	}

	return &ShareRecordResponse{
		RecordID:              req.RecordID,
		AuthorizationRequestID: req.AuthorizationRequestID,
		Status:                "SHARED",
	}, nil
}

// GetSharedRecord retrieves a shared record
func (s *SharingService) GetSharedRecord(ctx context.Context, certificateID, recordID, authorizationRequestID string) (*GetSharedRecordResponse, error) {
	// Validate identity (must be Research Org or Hospital)
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.Role != fabric.RoleResearch && identity.Role != fabric.RoleHospital {
		return nil, fmt.Errorf("only research organizations or hospitals can retrieve shared records")
	}

	// Get shared record from blockchain (contains encrypted pointer)
	sharedRecord, err := s.chaincodeService.GetSharedRecord(ctx, recordID, authorizationRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shared record: %w", err)
	}

	encryptedPointerHex, _ := sharedRecord["encryptedPointer"].(string)
	if encryptedPointerHex == "" {
		return nil, fmt.Errorf("encrypted pointer not found in shared record")
	}

	// Decode hex to bytes
	encryptedPointer, err := hex.DecodeString(encryptedPointerHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted pointer: %w", err)
	}

	// Decrypt pointer using key for this authorization request
	decryptedPointer, err := s.encryptionService.Decrypt(encryptedPointer, authorizationRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt pointer: %w", err)
	}

	// Parse pointer
	var pointerData map[string]interface{}
	if err := json.Unmarshal(decryptedPointer, &pointerData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pointer: %w", err)
	}

	storageKey, _ := pointerData["storage_key"].(string)
	if storageKey == "" {
		return nil, fmt.Errorf("storage key not found in pointer")
	}

	// Retrieve encrypted medical record from off-chain storage
	encryptedData, err := s.objectStorage.Get(storageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve encrypted data: %w", err)
	}

	// Decrypt medical record (using hospital key for initial encryption)
	// In production, this would use a more sophisticated key management
	hospitalID, _ := sharedRecord["sharedBy"].(string)
	decryptionKeyID := fmt.Sprintf("hospital-%s", hospitalID)
	decryptedData, err := s.encryptionService.Decrypt(encryptedData, decryptionKeyID)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt medical record: %w", err)
	}

	// Parse decrypted data
	var data map[string]interface{}
	if err := json.Unmarshal(decryptedData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted data: %w", err)
	}

	sharedAt, _ := sharedRecord["sharedAt"].(string)

	return &GetSharedRecordResponse{
		RecordID:              recordID,
		AuthorizationRequestID: authorizationRequestID,
		Data:                  data,
		SharedAt:              sharedAt,
	}, nil
}
