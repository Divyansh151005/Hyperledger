/*
 * Authorization and Consent Management Chaincode
 *
 * This chaincode implements authorization and consent management for a medical
 * information sharing platform, aligned with academic consortium blockchain design.
 *
 * Key Design Principles:
 * 1. Multi-party authorization: Requires both hospital and patient approval
 * 2. Time-bounded access: All authorizations have expiration dates
 * 3. State machine integrity: Enforces valid state transitions
 * 4. MSP and client identity-based access control: Fine-grained permissions
 * 5. Full auditability: All state transitions emit events
 *
 * Authorization Flow:
 * REQUESTED -> HOSPITAL_APPROVED -> PATIENT_APPROVED -> GRANTED
 * Any state can transition to REVOKED
 */

package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// AuthorizationConsentContract provides functions for managing authorization requests
type AuthorizationConsentContract struct {
	contractapi.Contract
}

// AuthorizationRequest represents an authorization request for accessing a medical record
type AuthorizationRequest struct {
	RequestID    string `json:"requestID"`    // Unique identifier for the authorization request
	RecordID     string `json:"recordID"`     // ID of the medical record being requested
	RequesterOrg string `json:"requesterOrg"` // MSP ID of the organization requesting access (ResearchOrgMSP)
	HospitalID   string `json:"hospitalID"`   // MSP ID of the hospital that owns the record
	PatientID    string `json:"patientID"`    // Client ID (X.509 certificate CN) of the patient
	Status       string `json:"status"`       // Current status: REQUESTED, HOSPITAL_APPROVED, PATIENT_APPROVED, GRANTED, REVOKED
	CreatedAt    string `json:"createdAt"`    // ISO 8601 timestamp of request creation
	ExpiresAt    string `json:"expiresAt"`    // ISO 8601 timestamp when authorization expires
}

// EventPayload represents the structure of events emitted for auditability
type EventPayload struct {
	RequestID   string `json:"requestID"`
	RecordID    string `json:"recordID"`
	Action      string `json:"action"`      // e.g., "AuthorizationRequested", "HospitalApproved", etc.
	InvokerMSP  string `json:"invokerMSP"` // MSP ID of the transaction invoker
	ClientID    string `json:"clientID"`    // Client ID (X.509 CN) of the invoker
	Timestamp   string `json:"timestamp"`   // ISO 8601 timestamp
	PatientID   string `json:"patientID,omitempty"`
	HospitalID  string `json:"hospitalID,omitempty"`
	RequesterOrg string `json:"requesterOrg,omitempty"`
	Status      string `json:"status,omitempty"`
}

// Constants for authorization status
const (
	StatusRequested        = "REQUESTED"
	StatusHospitalApproved = "HOSPITAL_APPROVED"
	StatusPatientApproved  = "PATIENT_APPROVED"
	StatusGranted          = "GRANTED"
	StatusRevoked          = "REVOKED"
)

// Constants for MSP names (must match network configuration)
const (
	MSPRegulator = "RegulatorMSP"
	MSPHospital1 = "HospitalMSP1"
	MSPHospital2 = "HospitalMSP2"
	MSPResearch  = "ResearchOrgMSP"
)

// RequestAccess creates a new authorization request for accessing a medical record
//
// Access Control:
// - ONLY ResearchOrgMSP can invoke this function
// - This enforces that only research organizations can request access
//
// Parameters:
//   - recordID: Unique identifier for the medical record being requested
//   - duration: Duration in hours for which access is requested (must be > 0)
//
// Returns: requestID (generated) or error
//
// Events: Emits "AuthorizationRequested" event for auditability
//
// State: Creates authorization request in REQUESTED status
func (s *AuthorizationConsentContract) RequestAccess(ctx contractapi.TransactionContextInterface, recordID string, duration int) (string, error) {
	// ========== ACCESS CONTROL: Verify invoker is ResearchOrgMSP ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	if invokerMSP != MSPResearch {
		return "", fmt.Errorf("access denied: RequestAccess can only be invoked by ResearchOrgMSP, got %s", invokerMSP)
	}

	// ========== VALIDATION: Verify input parameters ==========
	if recordID == "" {
		return "", fmt.Errorf("recordID cannot be empty")
	}
	if duration <= 0 {
		return "", fmt.Errorf("duration must be greater than 0 hours")
	}

	// ========== BUSINESS LOGIC: Verify medical record exists ==========
	// Query the medical-records chaincode to verify the record exists
	// Note: This requires the medical-records chaincode to be deployed on the same channel
	recordBytes, err := ctx.GetStub().GetState(recordID)
	if err != nil {
		return "", fmt.Errorf("failed to read medical record from world state: %v", err)
	}
	if recordBytes == nil {
		return "", fmt.Errorf("medical record %s does not exist", recordID)
	}

	// Parse medical record to extract hospitalID and patientID
	// Note: This assumes the medical record structure matches the medical-records chaincode
	var medicalRecord struct {
		RecordID   string `json:"recordID"`
		PatientID  string `json:"patientID"`
		HospitalID string `json:"hospitalID"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(recordBytes, &medicalRecord); err != nil {
		return "", fmt.Errorf("failed to parse medical record: %v", err)
	}

	if medicalRecord.Status != "ACTIVE" {
		return "", fmt.Errorf("medical record %s is not active (status: %s)", recordID, medicalRecord.Status)
	}

	// ========== BUSINESS LOGIC: Generate requestID and create authorization request ==========
	// Generate unique requestID: timestamp-based with recordID prefix
	timestamp := time.Now().UTC()
	requestID := fmt.Sprintf("AUTH-%s-%d", recordID, timestamp.UnixNano())

	// Check if requestID already exists (extremely unlikely, but check for safety)
	existingRequest, err := ctx.GetStub().GetState(requestID)
	if err != nil {
		return "", fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingRequest != nil {
		return "", fmt.Errorf("authorization request %s already exists", requestID)
	}

	// Calculate expiration time
	expiresAt := timestamp.Add(time.Duration(duration) * time.Hour)

	// Create authorization request
	authRequest := AuthorizationRequest{
		RequestID:    requestID,
		RecordID:     recordID,
		RequesterOrg: invokerMSP,
		HospitalID:   medicalRecord.HospitalID,
		PatientID:    medicalRecord.PatientID,
		Status:       StatusRequested,
		CreatedAt:    timestamp.Format(time.RFC3339),
		ExpiresAt:    expiresAt.Format(time.RFC3339),
	}

	authRequestJSON, err := json.Marshal(authRequest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal authorization request: %v", err)
	}

	// Store in world state
	err = ctx.GetStub().PutState(requestID, authRequestJSON)
	if err != nil {
		return "", fmt.Errorf("failed to put authorization request to world state: %v", err)
	}

	// ========== AUDITABILITY: Emit event for compliance tracking ==========
	clientID, _ := getClientID(ctx) // Best effort to get client ID
	eventPayload := EventPayload{
		RequestID:    requestID,
		RecordID:     recordID,
		Action:       "AuthorizationRequested",
		InvokerMSP:   invokerMSP,
		ClientID:     clientID,
		Timestamp:    timestamp.Format(time.RFC3339),
		PatientID:    medicalRecord.PatientID,
		HospitalID:   medicalRecord.HospitalID,
		RequesterOrg: invokerMSP,
		Status:       StatusRequested,
	}

	eventJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal event payload: %v", err)
	}

	err = ctx.GetStub().SetEvent("AuthorizationRequested", eventJSON)
	if err != nil {
		return "", fmt.Errorf("failed to emit event: %v", err)
	}

	return requestID, nil
}

// ApproveByHospital approves an authorization request by the hospital
//
// Access Control:
// - ONLY the owning HospitalMSP (the hospital that created the medical record) can invoke this
// - Verifies that the invoker MSP matches the HospitalID in the authorization request
//
// Parameters:
//   - requestID: Unique identifier for the authorization request
//
// Returns: Success message or error
//
// Events: Emits "HospitalApproved" event for auditability
//
// State Transitions:
//   - REQUESTED -> HOSPITAL_APPROVED (if patient hasn't approved yet)
//   - PATIENT_APPROVED -> GRANTED (if patient already approved)
func (s *AuthorizationConsentContract) ApproveByHospital(ctx contractapi.TransactionContextInterface, requestID string) error {
	// ========== ACCESS CONTROL: Verify invoker is the owning HospitalMSP ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	if !isHospitalMSP(invokerMSP) {
		return fmt.Errorf("access denied: ApproveByHospital can only be invoked by HospitalMSP organizations, got %s", invokerMSP)
	}

	// ========== VALIDATION: Verify input parameters ==========
	if requestID == "" {
		return fmt.Errorf("requestID cannot be empty")
	}

	// ========== BUSINESS LOGIC: Read and validate authorization request ==========
	authRequest, err := getAuthorizationRequest(ctx, requestID)
	if err != nil {
		return err
	}

	// Verify invoker is the owning hospital
	if invokerMSP != authRequest.HospitalID {
		return fmt.Errorf("access denied: only the owning hospital (%s) can approve this request, invoker is %s", authRequest.HospitalID, invokerMSP)
	}

	// ========== STATE MACHINE: Validate state transition ==========
	// Hospital can approve if:
	// 1. Status is REQUESTED (hospital approves first) -> HOSPITAL_APPROVED
	// 2. Status is PATIENT_APPROVED (patient approved first) -> GRANTED
	if authRequest.Status != StatusRequested && authRequest.Status != StatusPatientApproved {
		return fmt.Errorf("invalid state transition: authorization request must be in REQUESTED or PATIENT_APPROVED status, current status is %s", authRequest.Status)
	}

	// Check if authorization has expired
	if isExpired(authRequest.ExpiresAt) {
		return fmt.Errorf("authorization request has expired (expiresAt: %s)", authRequest.ExpiresAt)
	}

	// ========== BUSINESS LOGIC: Update status based on current state ==========
	var newStatus string
	if authRequest.Status == StatusRequested {
		newStatus = StatusHospitalApproved
	} else if authRequest.Status == StatusPatientApproved {
		newStatus = StatusGranted
	}

	authRequest.Status = newStatus

	authRequestJSON, err := json.Marshal(authRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal authorization request: %v", err)
	}

	err = ctx.GetStub().PutState(requestID, authRequestJSON)
	if err != nil {
		return fmt.Errorf("failed to update authorization request in world state: %v", err)
	}

	// ========== AUDITABILITY: Emit event for compliance tracking ==========
	clientID, _ := getClientID(ctx)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	eventPayload := EventPayload{
		RequestID:    requestID,
		RecordID:     authRequest.RecordID,
		Action:       "HospitalApproved",
		InvokerMSP:   invokerMSP,
		ClientID:     clientID,
		Timestamp:    timestamp,
		PatientID:    authRequest.PatientID,
		HospitalID:   authRequest.HospitalID,
		RequesterOrg: authRequest.RequesterOrg,
		Status:       newStatus,
	}

	eventJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}

	err = ctx.GetStub().SetEvent("HospitalApproved", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to emit event: %v", err)
	}

	return nil
}

// ApproveByPatient approves an authorization request by the patient
//
// Access Control:
// - ONLY the patient (client identity matching patientID) can invoke this
// - Verifies that the invoker's client ID matches the patientID in the authorization request
// - This is a critical security check using Fabric's Client Identity (CID) APIs
//
// Parameters:
//   - requestID: Unique identifier for the authorization request
//
// Returns: Success message or error
//
// Events: Emits "PatientApproved" event for auditability
//
// State Transition: HOSPITAL_APPROVED -> GRANTED (or REQUESTED -> PATIENT_APPROVED if hospital hasn't approved yet)
func (s *AuthorizationConsentContract) ApproveByPatient(ctx contractapi.TransactionContextInterface, requestID string) error {
	// ========== VALIDATION: Verify input parameters ==========
	if requestID == "" {
		return fmt.Errorf("requestID cannot be empty")
	}

	// ========== BUSINESS LOGIC: Read and validate authorization request ==========
	authRequest, err := getAuthorizationRequest(ctx, requestID)
	if err != nil {
		return err
	}

	// ========== ACCESS CONTROL: Verify invoker is the patient ==========
	// This is a critical security check - we must verify the client identity matches the patientID
	clientID, err := getClientID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client ID: %v", err)
	}

	if clientID != authRequest.PatientID {
		return fmt.Errorf("access denied: only the patient (patientID: %s) can approve this request, invoker client ID is %s", authRequest.PatientID, clientID)
	}

	// ========== STATE MACHINE: Validate state transition ==========
	// Patient can approve if:
	// 1. Status is REQUESTED (patient approves before hospital) -> PATIENT_APPROVED
	// 2. Status is HOSPITAL_APPROVED (hospital approved first) -> GRANTED
	if authRequest.Status != StatusRequested && authRequest.Status != StatusHospitalApproved {
		return fmt.Errorf("invalid state transition: authorization request must be in REQUESTED or HOSPITAL_APPROVED status, current status is %s", authRequest.Status)
	}

	// Check if authorization has expired
	if isExpired(authRequest.ExpiresAt) {
		return fmt.Errorf("authorization request has expired (expiresAt: %s)", authRequest.ExpiresAt)
	}

	// ========== BUSINESS LOGIC: Update status based on current state ==========
	var newStatus string
	if authRequest.Status == StatusRequested {
		newStatus = StatusPatientApproved
	} else if authRequest.Status == StatusHospitalApproved {
		newStatus = StatusGranted
	}

	authRequest.Status = newStatus

	authRequestJSON, err := json.Marshal(authRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal authorization request: %v", err)
	}

	err = ctx.GetStub().PutState(requestID, authRequestJSON)
	if err != nil {
		return fmt.Errorf("failed to update authorization request in world state: %v", err)
	}

	// ========== AUDITABILITY: Emit event for compliance tracking ==========
	invokerMSP, _ := getInvokerMSP(ctx) // Best effort
	timestamp := time.Now().UTC().Format(time.RFC3339)
	eventPayload := EventPayload{
		RequestID:    requestID,
		RecordID:     authRequest.RecordID,
		Action:       "PatientApproved",
		InvokerMSP:   invokerMSP,
		ClientID:     clientID,
		Timestamp:    timestamp,
		PatientID:    authRequest.PatientID,
		HospitalID:   authRequest.HospitalID,
		RequesterOrg: authRequest.RequesterOrg,
		Status:       newStatus,
	}

	eventJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}

	err = ctx.GetStub().SetEvent("PatientApproved", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to emit event: %v", err)
	}

	return nil
}

// RevokeAuthorization revokes an authorization request
//
// Access Control:
// - Patient (client identity matching patientID) can revoke
// - Owning HospitalMSP can revoke
// - This allows either party to revoke access at any time
//
// Parameters:
//   - requestID: Unique identifier for the authorization request
//
// Returns: Success message or error
//
// Events: Emits "AuthorizationRevoked" event for auditability
//
// State Transition: Any status -> REVOKED
func (s *AuthorizationConsentContract) RevokeAuthorization(ctx contractapi.TransactionContextInterface, requestID string) error {
	// ========== VALIDATION: Verify input parameters ==========
	if requestID == "" {
		return fmt.Errorf("requestID cannot be empty")
	}

	// ========== BUSINESS LOGIC: Read and validate authorization request ==========
	authRequest, err := getAuthorizationRequest(ctx, requestID)
	if err != nil {
		return err
	}

	// ========== ACCESS CONTROL: Verify invoker is patient or owning hospital ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	clientID, err := getClientID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get client ID: %v", err)
	}

	isPatient := clientID == authRequest.PatientID
	isOwningHospital := invokerMSP == authRequest.HospitalID && isHospitalMSP(invokerMSP)

	if !isPatient && !isOwningHospital {
		return fmt.Errorf("access denied: only the patient (patientID: %s) or owning hospital (%s) can revoke this authorization, invoker is %s (clientID: %s)", authRequest.PatientID, authRequest.HospitalID, invokerMSP, clientID)
	}

	// ========== STATE MACHINE: Check if already revoked ==========
	if authRequest.Status == StatusRevoked {
		return fmt.Errorf("authorization request is already revoked")
	}

	// ========== BUSINESS LOGIC: Update status to REVOKED ==========
	authRequest.Status = StatusRevoked

	authRequestJSON, err := json.Marshal(authRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal authorization request: %v", err)
	}

	err = ctx.GetStub().PutState(requestID, authRequestJSON)
	if err != nil {
		return fmt.Errorf("failed to update authorization request in world state: %v", err)
	}

	// ========== AUDITABILITY: Emit event for compliance tracking ==========
	timestamp := time.Now().UTC().Format(time.RFC3339)
	eventPayload := EventPayload{
		RequestID:    requestID,
		RecordID:     authRequest.RecordID,
		Action:       "AuthorizationRevoked",
		InvokerMSP:   invokerMSP,
		ClientID:     clientID,
		Timestamp:    timestamp,
		PatientID:    authRequest.PatientID,
		HospitalID:   authRequest.HospitalID,
		RequesterOrg: authRequest.RequesterOrg,
		Status:       StatusRevoked,
	}

	eventJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}

	err = ctx.GetStub().SetEvent("AuthorizationRevoked", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to emit event: %v", err)
	}

	return nil
}

// GetAuthorization retrieves an authorization request from the ledger
//
// Access Control:
// - RegulatorMSP: Can access any authorization request (regulatory oversight)
// - Owning HospitalMSP: Can access authorization requests for their records
// - Patient: Can access authorization requests where they are the patient
// - Requesting ResearchOrg: Can access authorization requests they created
//
// Parameters:
//   - requestID: Unique identifier for the authorization request
//
// Returns: AuthorizationRequest JSON or error
func (s *AuthorizationConsentContract) GetAuthorization(ctx contractapi.TransactionContextInterface, requestID string) (*AuthorizationRequest, error) {
	// ========== VALIDATION: Verify input parameters ==========
	if requestID == "" {
		return nil, fmt.Errorf("requestID cannot be empty")
	}

	// ========== BUSINESS LOGIC: Read authorization request ==========
	authRequest, err := getAuthorizationRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}

	// ========== ACCESS CONTROL: Verify invoker has access ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	clientID, err := getClientID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %v", err)
	}

	// Access control logic:
	// 1. RegulatorMSP can access any authorization request
	// 2. Owning HospitalMSP can access authorization requests for their records
	// 3. Patient can access authorization requests where they are the patient
	// 4. Requesting ResearchOrg can access authorization requests they created
	hasAccess := false
	if invokerMSP == MSPRegulator {
		hasAccess = true
	} else if invokerMSP == authRequest.HospitalID && isHospitalMSP(invokerMSP) {
		hasAccess = true
	} else if clientID == authRequest.PatientID {
		hasAccess = true
	} else if invokerMSP == authRequest.RequesterOrg {
		hasAccess = true
	}

	if !hasAccess {
		return nil, fmt.Errorf("access denied: GetAuthorization can only be invoked by RegulatorMSP, owning HospitalMSP (%s), patient (patientID: %s), or requesting ResearchOrg (%s). Invoker is %s (clientID: %s)", authRequest.HospitalID, authRequest.PatientID, authRequest.RequesterOrg, invokerMSP, clientID)
	}

	return authRequest, nil
}

// ========== HELPER FUNCTIONS ==========

// getAuthorizationRequest retrieves an authorization request from the ledger
func getAuthorizationRequest(ctx contractapi.TransactionContextInterface, requestID string) (*AuthorizationRequest, error) {
	requestJSON, err := ctx.GetStub().GetState(requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if requestJSON == nil {
		return nil, fmt.Errorf("authorization request %s does not exist", requestID)
	}

	var authRequest AuthorizationRequest
	err = json.Unmarshal(requestJSON, &authRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal authorization request: %v", err)
	}

	return &authRequest, nil
}

// getInvokerMSP retrieves the MSP ID of the transaction invoker
func getInvokerMSP(ctx contractapi.TransactionContextInterface) (string, error) {
	clientIdentity, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity MSP ID: %v", err)
	}
	return clientIdentity, nil
}

// getClientID retrieves the client ID (X.509 certificate CN) of the transaction invoker
// This is critical for patient identity verification
func getClientID(ctx contractapi.TransactionContextInterface) (string, error) {
	// Get the X.509 certificate attributes
	// The client ID is typically the CN (Common Name) from the certificate
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity ID: %v", err)
	}

	// Extract CN from the ID (format: "x509::CN=<CN>,OU=<OU>,O=<O>::CN=<CN>,OU=<OU>,O=<O>")
	// For simplicity, we'll use the full ID, but in practice you might want to parse the CN
	// The client ID should be set to the CN when the identity is created
	return id, nil
}

// isHospitalMSP checks if the given MSP is a Hospital MSP
func isHospitalMSP(mspID string) bool {
	return mspID == MSPHospital1 || mspID == MSPHospital2
}

// isExpired checks if an expiration timestamp has passed
func isExpired(expiresAt string) bool {
	expirationTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		// If we can't parse the timestamp, consider it expired for safety
		return true
	}
	return time.Now().UTC().After(expirationTime)
}

// ========== MAIN FUNCTION ==========

func main() {
	authorizationConsentContract, err := contractapi.NewChaincode(&AuthorizationConsentContract{})
	if err != nil {
		fmt.Printf("Error creating authorization-consent chaincode: %v", err)
		return
	}

	if err := authorizationConsentContract.Start(); err != nil {
		fmt.Printf("Error starting authorization-consent chaincode: %v", err)
	}
}
