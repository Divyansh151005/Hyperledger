/*
 * Anonymous Information Sharing Chaincode
 *
 * This chaincode implements supervised anonymous information sharing using Private Data Collections (PDCs),
 * aligned with academic medical blockchain paper requirements.
 *
 * Key Design Principles:
 * 1. Privacy by Design: Sensitive sharing payloads stored in Private Data Collections
 * 2. Hash Commitments: Only hash + metadata stored on public ledger
 * 3. Authorization Enforcement: Requires GRANTED authorization before sharing
 * 4. Regulatory Auditability: RegulatorMSP can audit all transactions and access payloads
 * 5. Access Control: MSP-based access control enforced at chaincode level
 *
 * Privacy Model:
 * - Private Data Collection: collection_record_<recordID>
 * - Member Organizations: Owning HospitalMSP, Requesting ResearchOrgMSP, RegulatorMSP
 * - Unrelated organizations see only hash commitments on public ledger
 * - Regulatory Authority can audit metadata and access payloads
 */

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// AnonymousSharingContract provides functions for anonymous information sharing
type AnonymousSharingContract struct {
	contractapi.Contract
}

// SharedRecordPayload represents the private data stored in Private Data Collection
// This data is only visible to: Owning HospitalMSP, Requesting ResearchOrgMSP, RegulatorMSP
type SharedRecordPayload struct {
	RecordID              string `json:"recordID"`              // Unique identifier for the medical record
	AuthorizationRequestID string `json:"authorizationRequestID"` // ID of the authorization request
	EncryptedPointer      string `json:"encryptedPointer"`      // Encrypted pointer (URL/IPFS) to actual data
	SharedAt              string `json:"sharedAt"`              // ISO 8601 timestamp when record was shared
	SharedBy              string `json:"sharedBy"`              // MSP ID of the hospital that shared the record
}

// SharedRecordCommitment represents the public commitment stored on the ledger
// This is visible to all organizations on the channel
type SharedRecordCommitment struct {
	RecordID              string `json:"recordID"`              // Unique identifier for the medical record
	AuthorizationRequestID string `json:"authorizationRequestID"` // ID of the authorization request
	PayloadHash           string `json:"payloadHash"`          // SHA-256 hash of the SharedRecordPayload
	Timestamp             string `json:"timestamp"`           // ISO 8601 timestamp
}

// EventPayload represents the structure of events emitted for auditability
type EventPayload struct {
	RecordID              string `json:"recordID"`
	AuthorizationRequestID string `json:"authorizationRequestID"`
	Action                string `json:"action"`      // e.g., "RecordShared", "RecordAccessed"
	InvokerMSP            string `json:"invokerMSP"` // MSP ID of the transaction invoker
	ClientID              string `json:"clientID"`    // Client ID (X.509 CN) of the invoker
	Timestamp             string `json:"timestamp"`   // ISO 8601 timestamp
	PayloadHash           string `json:"payloadHash,omitempty"`
}

// Constants for MSP names (must match network configuration)
const (
	MSPRegulator = "RegulatorMSP"
	MSPHospital1 = "HospitalMSP1"
	MSPHospital2 = "HospitalMSP2"
	MSPResearch  = "ResearchOrgMSP"
)

// Collection name for Private Data Collections
// Note: In production, you can use either:
// 1. A single collection (as implemented here) - more practical
// 2. Per-record collections (collection_record_<recordID>) - requires pre-defining collections
// The academic requirement specifies pattern "collection_record_<recordID>", but Fabric requires
// collections to be statically defined. For practicality, we use a single collection.
// To match the exact pattern requirement, collections would need to be pre-defined per record.
const collectionName = "shared_records_collection"

// getCollectionName returns the collection name
// For pattern-based approach: return fmt.Sprintf("collection_record_%s", recordID)
// For single collection approach (current): return collectionName
func getCollectionName(recordID string) string {
	// Using single collection for practicality
	// To use pattern-based collections, uncomment below and pre-define collections:
	// return fmt.Sprintf("collection_record_%s", recordID)
	return collectionName
}

// ShareRecord shares a medical record with authorized research organization
//
// Access Control:
// - ONLY HospitalMSP organizations can invoke this function
// - Verifies that authorization status is GRANTED
// - Stores payload in Private Data Collection
// - Stores hash commitment on public ledger
//
// Parameters:
//   - recordID: Unique identifier for the medical record
//   - authorizationRequestID: ID of the authorization request (must be GRANTED)
//   - encryptedPointer: Encrypted pointer to the actual data (URL/IPFS, already encrypted by middleware)
//
// Returns: Success message or error
//
// Events: Emits "RecordShared" event for auditability
//
// Privacy:
// - Payload stored in Private Data Collection (visible only to HospitalMSP, ResearchOrgMSP, RegulatorMSP)
// - Hash commitment stored on public ledger (visible to all)
func (s *AnonymousSharingContract) ShareRecord(ctx contractapi.TransactionContextInterface, recordID string, authorizationRequestID string, encryptedPointer string) error {
	// ========== ACCESS CONTROL: Verify invoker is a Hospital MSP ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	if !isHospitalMSP(invokerMSP) {
		return fmt.Errorf("access denied: ShareRecord can only be invoked by HospitalMSP organizations, got %s", invokerMSP)
	}

	// ========== VALIDATION: Verify input parameters ==========
	if recordID == "" {
		return fmt.Errorf("recordID cannot be empty")
	}
	if authorizationRequestID == "" {
		return fmt.Errorf("authorizationRequestID cannot be empty")
	}
	if encryptedPointer == "" {
		return fmt.Errorf("encryptedPointer cannot be empty")
	}

	// ========== BUSINESS LOGIC: Verify authorization status is GRANTED ==========
	// Query the authorization-consent chaincode to verify authorization
	authRequest, err := getAuthorizationRequest(ctx, authorizationRequestID)
	if err != nil {
		return fmt.Errorf("failed to get authorization request: %v", err)
	}

	// Verify authorization status is GRANTED
	if authRequest.Status != "GRANTED" {
		return fmt.Errorf("authorization request %s is not GRANTED (current status: %s)", authorizationRequestID, authRequest.Status)
	}

	// Verify the recordID matches
	if authRequest.RecordID != recordID {
		return fmt.Errorf("authorization request recordID (%s) does not match provided recordID (%s)", authRequest.RecordID, recordID)
	}

	// Verify the invoker is the owning hospital
	if invokerMSP != authRequest.HospitalID {
		return fmt.Errorf("access denied: only the owning hospital (%s) can share this record, invoker is %s", authRequest.HospitalID, invokerMSP)
	}

	// Check if authorization has expired
	if isExpired(authRequest.ExpiresAt) {
		return fmt.Errorf("authorization request has expired (expiresAt: %s)", authRequest.ExpiresAt)
	}

	// ========== BUSINESS LOGIC: Check if record already shared ==========
	// Check if commitment already exists on public ledger
	commitmentKey := fmt.Sprintf("SHARED_%s_%s", recordID, authorizationRequestID)
	existingCommitment, err := ctx.GetStub().GetState(commitmentKey)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingCommitment != nil {
		return fmt.Errorf("record %s has already been shared for authorization request %s", recordID, authorizationRequestID)
	}

	// ========== PRIVACY: Create payload and compute hash ==========
	timestamp := time.Now().UTC().Format(time.RFC3339)
	payload := SharedRecordPayload{
		RecordID:              recordID,
		AuthorizationRequestID: authorizationRequestID,
		EncryptedPointer:      encryptedPointer,
		SharedAt:              timestamp,
		SharedBy:              invokerMSP,
	}

	// Compute SHA-256 hash of the payload
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}
	payloadHash := computeHash(payloadJSON)

	// ========== PRIVACY: Store payload in Private Data Collection ==========
	collectionName := getCollectionName(recordID)
	payloadKey := fmt.Sprintf("%s_%s", recordID, authorizationRequestID)

	// Put private data in collection
	err = ctx.GetStub().PutPrivateData(collectionName, payloadKey, payloadJSON)
	if err != nil {
		return fmt.Errorf("failed to put private data: %v", err)
	}

	// ========== PUBLIC LEDGER: Store hash commitment ==========
	commitment := SharedRecordCommitment{
		RecordID:              recordID,
		AuthorizationRequestID: authorizationRequestID,
		PayloadHash:           payloadHash,
		Timestamp:             timestamp,
	}

	commitmentJSON, err := json.Marshal(commitment)
	if err != nil {
		return fmt.Errorf("failed to marshal commitment: %v", err)
	}

	err = ctx.GetStub().PutState(commitmentKey, commitmentJSON)
	if err != nil {
		return fmt.Errorf("failed to put commitment to world state: %v", err)
	}

	// ========== AUDITABILITY: Emit event for compliance tracking ==========
	clientID, _ := getClientID(ctx)
	eventPayload := EventPayload{
		RecordID:              recordID,
		AuthorizationRequestID: authorizationRequestID,
		Action:                "RecordShared",
		InvokerMSP:            invokerMSP,
		ClientID:              clientID,
		Timestamp:             timestamp,
		PayloadHash:           payloadHash,
	}

	eventJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}

	err = ctx.GetStub().SetEvent("RecordShared", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to emit event: %v", err)
	}

	return nil
}

// GetSharedRecord retrieves a shared record from the Private Data Collection
//
// Access Control:
// - Owning HospitalMSP: Can access records they shared
// - Requesting ResearchOrgMSP: Can access records for which they have GRANTED authorization
// - RegulatorMSP: Can access any shared record (regulatory oversight)
//
// Parameters:
//   - recordID: Unique identifier for the medical record
//   - authorizationRequestID: ID of the authorization request
//
// Returns: SharedRecordPayload JSON or error
//
// Privacy:
// - Reads from Private Data Collection
// - Only accessible to authorized organizations (HospitalMSP, ResearchOrgMSP, RegulatorMSP)
func (s *AnonymousSharingContract) GetSharedRecord(ctx contractapi.TransactionContextInterface, recordID string, authorizationRequestID string) (*SharedRecordPayload, error) {
	// ========== VALIDATION: Verify input parameters ==========
	if recordID == "" {
		return nil, fmt.Errorf("recordID cannot be empty")
	}
	if authorizationRequestID == "" {
		return nil, fmt.Errorf("authorizationRequestID cannot be empty")
	}

	// ========== ACCESS CONTROL: Verify invoker has access ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	// Get authorization request to verify access
	authRequest, err := getAuthorizationRequest(ctx, authorizationRequestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization request: %v", err)
	}

	// Verify recordID matches
	if authRequest.RecordID != recordID {
		return nil, fmt.Errorf("authorization request recordID (%s) does not match provided recordID (%s)", authRequest.RecordID, recordID)
	}

	// Access control logic:
	// 1. RegulatorMSP can access any shared record (regulatory oversight)
	// 2. Owning HospitalMSP can access records they shared
	// 3. Requesting ResearchOrgMSP can access records for which they have authorization
	hasAccess := false
	if invokerMSP == MSPRegulator {
		hasAccess = true
	} else if invokerMSP == authRequest.HospitalID && isHospitalMSP(invokerMSP) {
		hasAccess = true
	} else if invokerMSP == authRequest.RequesterOrg {
		hasAccess = true
	}

	if !hasAccess {
		return nil, fmt.Errorf("access denied: GetSharedRecord can only be invoked by RegulatorMSP, owning HospitalMSP (%s), or requesting ResearchOrgMSP (%s). Invoker is %s", authRequest.HospitalID, authRequest.RequesterOrg, invokerMSP)
	}

	// ========== PRIVACY: Read from Private Data Collection ==========
	collectionName := getCollectionName(recordID)
	payloadKey := fmt.Sprintf("%s_%s", recordID, authorizationRequestID)

	payloadBytes, err := ctx.GetStub().GetPrivateData(collectionName, payloadKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get private data: %v", err)
	}
	if payloadBytes == nil {
		return nil, fmt.Errorf("shared record not found for recordID %s and authorizationRequestID %s", recordID, authorizationRequestID)
	}

	var payload SharedRecordPayload
	err = json.Unmarshal(payloadBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %v", err)
	}

	// ========== AUDITABILITY: Emit access event ==========
	clientID, _ := getClientID(ctx)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	eventPayload := EventPayload{
		RecordID:              recordID,
		AuthorizationRequestID: authorizationRequestID,
		Action:                "RecordAccessed",
		InvokerMSP:            invokerMSP,
		ClientID:              clientID,
		Timestamp:             timestamp,
	}

	eventJSON, err := json.Marshal(eventPayload)
	if err != nil {
		// Log error but don't fail the transaction
		_ = fmt.Errorf("failed to marshal event payload: %v", err)
	} else {
		_ = ctx.GetStub().SetEvent("RecordAccessed", eventJSON)
	}

	return &payload, nil
}

// VerifySharedRecord verifies if a provided hash matches the stored hash commitment
//
// Access Control:
// - Public function: Any organization can verify hash commitments
// - This enables verification without revealing the payload
//
// Parameters:
//   - recordID: Unique identifier for the medical record
//   - authorizationRequestID: ID of the authorization request
//   - providedHash: The hash to verify against the stored commitment
//
// Returns: true if hashes match, false otherwise, or error
//
// Privacy:
// - Only verifies hash commitments on public ledger
// - Does not access Private Data Collection
// - Enables integrity verification without revealing payload
func (s *AnonymousSharingContract) VerifySharedRecord(ctx contractapi.TransactionContextInterface, recordID string, authorizationRequestID string, providedHash string) (bool, error) {
	// ========== VALIDATION: Verify input parameters ==========
	if recordID == "" {
		return false, fmt.Errorf("recordID cannot be empty")
	}
	if authorizationRequestID == "" {
		return false, fmt.Errorf("authorizationRequestID cannot be empty")
	}
	if providedHash == "" {
		return false, fmt.Errorf("providedHash cannot be empty")
	}

	// Validate hash format (should be hex-encoded SHA-256, 64 characters)
	if len(providedHash) != 64 {
		return false, fmt.Errorf("invalid providedHash format: expected 64-character hex string (SHA-256)")
	}
	// Verify it's valid hex
	if _, err := hex.DecodeString(providedHash); err != nil {
		return false, fmt.Errorf("invalid providedHash format: not a valid hex string")
	}

	// ========== BUSINESS LOGIC: Read commitment from public ledger ==========
	commitmentKey := fmt.Sprintf("SHARED_%s_%s", recordID, authorizationRequestID)
	commitmentBytes, err := ctx.GetStub().GetState(commitmentKey)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	if commitmentBytes == nil {
		return false, fmt.Errorf("shared record commitment not found for recordID %s and authorizationRequestID %s", recordID, authorizationRequestID)
	}

	var commitment SharedRecordCommitment
	err = json.Unmarshal(commitmentBytes, &commitment)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal commitment: %v", err)
	}

	// ========== BUSINESS LOGIC: Compare hashes ==========
	// Case-insensitive comparison (hashes are typically lowercase)
	// Both are already hex-encoded strings, so compare directly
	storedHashLower := strings.ToLower(commitment.PayloadHash)
	providedHashLower := strings.ToLower(providedHash)
	match := storedHashLower == providedHashLower

	return match, nil
}

// ========== HELPER FUNCTIONS ==========

// getAuthorizationRequest retrieves an authorization request from the authorization-consent chaincode
// This queries the same channel's world state to get authorization information
func getAuthorizationRequest(ctx contractapi.TransactionContextInterface, requestID string) (*AuthorizationRequest, error) {
	// Read from world state (authorization-consent chaincode stores requests on the same channel)
	requestJSON, err := ctx.GetStub().GetState(requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to read authorization request from world state: %v", err)
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

// AuthorizationRequest represents the structure from authorization-consent chaincode
// This must match the structure in authorization_consent.go
type AuthorizationRequest struct {
	RequestID    string `json:"requestID"`
	RecordID     string `json:"recordID"`
	RequesterOrg string `json:"requesterOrg"`
	HospitalID   string `json:"hospitalID"`
	PatientID    string `json:"patientID"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
	ExpiresAt    string `json:"expiresAt"`
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
func getClientID(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity ID: %v", err)
	}
	return id, nil
}

// isHospitalMSP checks if the given MSP is a Hospital MSP
func isHospitalMSP(mspID string) bool {
	return mspID == MSPHospital1 || mspID == MSPHospital2
}

// computeHash computes SHA-256 hash of data and returns hex-encoded string
func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
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
	anonymousSharingContract, err := contractapi.NewChaincode(&AnonymousSharingContract{})
	if err != nil {
		fmt.Printf("Error creating anonymous-sharing chaincode: %v", err)
		return
	}

	if err := anonymousSharingContract.Start(); err != nil {
		fmt.Printf("Error starting anonymous-sharing chaincode: %v", err)
	}
}
