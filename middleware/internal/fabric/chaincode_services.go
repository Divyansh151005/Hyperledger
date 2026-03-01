package fabric

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-gateway/pkg/client"
)

// MedicalRecordsService handles medical records chaincode operations
type MedicalRecordsService struct {
	contract *client.Contract
}

// NewMedicalRecordsService creates a new medical records service
func NewMedicalRecordsService(gateway *Gateway, channelName, chaincodeName string) *MedicalRecordsService {
	contract := gateway.GetContract(chaincodeName)
	return &MedicalRecordsService{contract: contract}
}

// CreateMedicalRecord creates a new medical record on the blockchain
func (s *MedicalRecordsService) CreateMedicalRecord(ctx context.Context, recordID, patientID, recordHash string) error {
	_, err := InvokeContract(ctx, s.contract, "CreateMedicalRecord", recordID, patientID, recordHash)
	return err
}

// GetMedicalRecord retrieves a medical record from the blockchain
func (s *MedicalRecordsService) GetMedicalRecord(ctx context.Context, recordID string) (map[string]interface{}, error) {
	result, err := QueryContract(ctx, s.contract, "GetMedicalRecord", recordID)
	if err != nil {
		return nil, err
	}

	var record map[string]interface{}
	if err := json.Unmarshal(result, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	return record, nil
}

// VerifyMedicalRecordHash verifies a medical record hash
func (s *MedicalRecordsService) VerifyMedicalRecordHash(ctx context.Context, recordID, providedHash string) (bool, error) {
	result, err := QueryContract(ctx, s.contract, "VerifyMedicalRecordHash", recordID, providedHash)
	if err != nil {
		return false, err
	}

	// Chaincode returns boolean as JSON string "true" or "false"
	var verified bool
	if err := json.Unmarshal(result, &verified); err != nil {
		// Try parsing as string
		var resultStr string
		if err2 := json.Unmarshal(result, &resultStr); err2 == nil {
			verified = resultStr == "true"
		} else {
			return false, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return verified, nil
}

// AuthorizationService handles authorization chaincode operations
type AuthorizationService struct {
	contract *client.Contract
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(gateway *Gateway, channelName, chaincodeName string) *AuthorizationService {
	contract := gateway.GetContract(chaincodeName)
	return &AuthorizationService{contract: contract}
}

// RequestAccess creates a new authorization request
func (s *AuthorizationService) RequestAccess(ctx context.Context, recordID string, duration int) (string, error) {
	result, err := InvokeContract(ctx, s.contract, "RequestAccess", recordID, fmt.Sprintf("%d", duration))
	if err != nil {
		return "", err
	}

	// The chaincode returns the requestID as a string directly
	requestID := string(result)
	if requestID == "" {
		return "", fmt.Errorf("empty requestID returned from chaincode")
	}

	return requestID, nil
}

// ApproveByHospital approves an authorization request by hospital
func (s *AuthorizationService) ApproveByHospital(ctx context.Context, requestID string) error {
	_, err := InvokeContract(ctx, s.contract, "ApproveByHospital", requestID)
	return err
}

// ApproveByPatient approves an authorization request by patient
func (s *AuthorizationService) ApproveByPatient(ctx context.Context, requestID string) error {
	_, err := InvokeContract(ctx, s.contract, "ApproveByPatient", requestID)
	return err
}

// RevokeAuthorization revokes an authorization request
func (s *AuthorizationService) RevokeAuthorization(ctx context.Context, requestID string) error {
	_, err := InvokeContract(ctx, s.contract, "RevokeAuthorization", requestID)
	return err
}

// GetAuthorization retrieves an authorization request
func (s *AuthorizationService) GetAuthorization(ctx context.Context, requestID string) (map[string]interface{}, error) {
	result, err := QueryContract(ctx, s.contract, "GetAuthorization", requestID)
	if err != nil {
		return nil, err
	}

	var authRequest map[string]interface{}
	if err := json.Unmarshal(result, &authRequest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal authorization request: %w", err)
	}

	return authRequest, nil
}

// SharingService handles anonymous sharing chaincode operations
type SharingService struct {
	contract *client.Contract
}

// NewSharingService creates a new sharing service
func NewSharingService(gateway *Gateway, channelName, chaincodeName string) *SharingService {
	contract := gateway.GetContract(chaincodeName)
	return &SharingService{contract: contract}
}

// ShareRecord shares a medical record
func (s *SharingService) ShareRecord(ctx context.Context, recordID, authorizationRequestID, encryptedPointer string) error {
	_, err := InvokeContract(ctx, s.contract, "ShareRecord", recordID, authorizationRequestID, encryptedPointer)
	return err
}

// GetSharedRecord retrieves a shared record
func (s *SharingService) GetSharedRecord(ctx context.Context, recordID, authorizationRequestID string) (map[string]interface{}, error) {
	result, err := QueryContract(ctx, s.contract, "GetSharedRecord", recordID, authorizationRequestID)
	if err != nil {
		return nil, err
	}

	var sharedRecord map[string]interface{}
	if err := json.Unmarshal(result, &sharedRecord); err != nil {
		return nil, fmt.Errorf("failed to unmarshal shared record: %w", err)
	}

	return sharedRecord, nil
}

// VerifySharedRecord verifies a shared record hash
func (s *SharingService) VerifySharedRecord(ctx context.Context, recordID, authorizationRequestID, providedHash string) (bool, error) {
	result, err := QueryContract(ctx, s.contract, "VerifySharedRecord", recordID, authorizationRequestID, providedHash)
	if err != nil {
		return false, err
	}

	// Chaincode returns boolean as JSON string "true" or "false"
	var verified bool
	if err := json.Unmarshal(result, &verified); err != nil {
		// Try parsing as string
		var resultStr string
		if err2 := json.Unmarshal(result, &resultStr); err2 == nil {
			verified = resultStr == "true"
		} else {
			return false, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return verified, nil
}
