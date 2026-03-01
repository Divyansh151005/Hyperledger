package api

import (
	"context"
	"fmt"

	"github.com/hyperledger/medical-middleware/internal/fabric"
)

// RequestAccessRequest represents a request to request access
type RequestAccessRequest struct {
	RecordID string `json:"record_id" binding:"required"`
	Duration int    `json:"duration" binding:"required"` // Duration in hours
}

// RequestAccessResponse represents the response after requesting access
type RequestAccessResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
}

// ApproveRequest represents a request to approve authorization
type ApproveRequest struct {
	RequestID string `json:"request_id" binding:"required"`
}

// GetAuthorizationResponse represents an authorization request
type GetAuthorizationResponse struct {
	RequestID    string `json:"request_id"`
	RecordID     string `json:"record_id"`
	RequesterOrg string `json:"requester_org"`
	HospitalID   string `json:"hospital_id"`
	PatientID    string `json:"patient_id"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	ExpiresAt    string `json:"expires_at"`
}

// RequestAccess creates a new authorization request
func (s *AuthorizationService) RequestAccess(ctx context.Context, certificateID string, req RequestAccessRequest) (*RequestAccessResponse, error) {
	// Validate identity (must be Research Org)
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.Role != fabric.RoleResearch {
		return nil, fmt.Errorf("only research organizations can request access")
	}

	// Invoke chaincode to create authorization request
	requestID, err := s.chaincodeService.RequestAccess(ctx, req.RecordID, req.Duration)
	if err != nil {
		return nil, fmt.Errorf("failed to request access: %w", err)
	}

	return &RequestAccessResponse{
		RequestID: requestID,
		Status:   "REQUESTED",
	}, nil
}

// ApproveByHospital approves an authorization request by hospital
func (s *AuthorizationService) ApproveByHospital(ctx context.Context, certificateID string, req ApproveRequest) error {
	// Validate identity (must be Hospital)
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.Role != fabric.RoleHospital {
		return fmt.Errorf("only hospitals can approve authorization requests")
	}

	// Get authorization request to verify ownership
	authRequest, err := s.chaincodeService.GetAuthorization(ctx, req.RequestID)
	if err != nil {
		return fmt.Errorf("failed to get authorization request: %w", err)
	}

	hospitalID, _ := authRequest["hospitalID"].(string)
	if identity.MSPID != hospitalID {
		return fmt.Errorf("only the owning hospital can approve this request")
	}

	// Invoke chaincode to approve
	if err := s.chaincodeService.ApproveByHospital(ctx, req.RequestID); err != nil {
		return fmt.Errorf("failed to approve by hospital: %w", err)
	}

	// Check if status is now GRANTED
	updatedAuthRequest, err := s.chaincodeService.GetAuthorization(ctx, req.RequestID)
	if err != nil {
		return fmt.Errorf("failed to get updated authorization request: %w", err)
	}

	status, _ := updatedAuthRequest["status"].(string)
	if status == "GRANTED" {
		// Generate encryption key for this authorization request
		// The key will be used to encrypt the pointer when sharing
		authorizationRequestID := req.RequestID
		_, err := s.encryptionService.Encrypt([]byte("dummy"), authorizationRequestID)
		if err != nil {
			return fmt.Errorf("failed to generate encryption key: %w", err)
		}
	}

	return nil
}

// ApproveByPatient approves an authorization request by patient
func (s *AuthorizationService) ApproveByPatient(ctx context.Context, certificateID string, req ApproveRequest) error {
	// Validate identity (must be Patient)
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.Role != fabric.RolePatient {
		return fmt.Errorf("only patients can approve authorization requests")
	}

	// Get authorization request to verify patient matches
	authRequest, err := s.chaincodeService.GetAuthorization(ctx, req.RequestID)
	if err != nil {
		return fmt.Errorf("failed to get authorization request: %w", err)
	}

	patientID, _ := authRequest["patientID"].(string)
	if identity.UserID != patientID {
		return fmt.Errorf("only the patient can approve this request")
	}

	// Invoke chaincode to approve
	if err := s.chaincodeService.ApproveByPatient(ctx, req.RequestID); err != nil {
		return fmt.Errorf("failed to approve by patient: %w", err)
	}

	// Check if status is now GRANTED
	updatedAuthRequest, err := s.chaincodeService.GetAuthorization(ctx, req.RequestID)
	if err != nil {
		return fmt.Errorf("failed to get updated authorization request: %w", err)
	}

	status, _ := updatedAuthRequest["status"].(string)
	if status == "GRANTED" {
		// Generate encryption key for this authorization request
		authorizationRequestID := req.RequestID
		_, err := s.encryptionService.Encrypt([]byte("dummy"), authorizationRequestID)
		if err != nil {
			return fmt.Errorf("failed to generate encryption key: %w", err)
		}
	}

	return nil
}

// RevokeAuthorization revokes an authorization request
func (s *AuthorizationService) RevokeAuthorization(ctx context.Context, certificateID string, req ApproveRequest) error {
	// Validate identity (must be Patient or Hospital)
	identity, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return fmt.Errorf("failed to get identity: %w", err)
	}

	if identity.Role != fabric.RolePatient && identity.Role != fabric.RoleHospital {
		return fmt.Errorf("only patients or hospitals can revoke authorization requests")
	}

	// Get authorization request to verify ownership
	authRequest, err := s.chaincodeService.GetAuthorization(ctx, req.RequestID)
	if err != nil {
		return fmt.Errorf("failed to get authorization request: %w", err)
	}

	patientID, _ := authRequest["patientID"].(string)
	hospitalID, _ := authRequest["hospitalID"].(string)

	// Verify ownership
	if identity.Role == fabric.RolePatient && identity.UserID != patientID {
		return fmt.Errorf("only the patient can revoke this request")
	}
	if identity.Role == fabric.RoleHospital && identity.MSPID != hospitalID {
		return fmt.Errorf("only the owning hospital can revoke this request")
	}

	// Invoke chaincode to revoke
	if err := s.chaincodeService.RevokeAuthorization(ctx, req.RequestID); err != nil {
		return fmt.Errorf("failed to revoke authorization: %w", err)
	}

	return nil
}

// GetAuthorization retrieves an authorization request
func (s *AuthorizationService) GetAuthorization(ctx context.Context, certificateID, requestID string) (*GetAuthorizationResponse, error) {
	// Validate identity
	_, err := s.identityService.GetIdentity(certificateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity: %w", err)
	}

	// Get authorization request from blockchain
	authRequest, err := s.chaincodeService.GetAuthorization(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization request: %w", err)
	}

	// Extract fields
	requestIDStr, _ := authRequest["requestID"].(string)
	recordID, _ := authRequest["recordID"].(string)
	requesterOrg, _ := authRequest["requesterOrg"].(string)
	hospitalID, _ := authRequest["hospitalID"].(string)
	patientID, _ := authRequest["patientID"].(string)
	status, _ := authRequest["status"].(string)
	createdAt, _ := authRequest["createdAt"].(string)
	expiresAt, _ := authRequest["expiresAt"].(string)

	return &GetAuthorizationResponse{
		RequestID:    requestIDStr,
		RecordID:     recordID,
		RequesterOrg: requesterOrg,
		HospitalID:   hospitalID,
		PatientID:    patientID,
		Status:       status,
		CreatedAt:    createdAt,
		ExpiresAt:    expiresAt,
	}, nil
}
