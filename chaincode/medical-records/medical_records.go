/*
 * Medical Records Chaincode
 *
 * This chaincode implements a blockchain-based medical information sharing system
 * aligned with academic paper design for regulatory compliance and patient privacy.
 *
 * Key Design Principles:
 * 1. Metadata-only storage: Only record metadata is stored on-chain, not raw medical data
 * 2. MSP-based access control: Enforces access at the chaincode level using client identity
 * 3. Auditability: All state-changing operations emit events for compliance tracking
 * 4. Regulatory compliance: Aligned with healthcare regulations (HIPAA, GDPR considerations)
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

// MedicalRecordContract provides functions for managing medical records
type MedicalRecordContract struct {
	contractapi.Contract
}

// MedicalRecord represents the metadata structure for a medical record
// Note: This stores only metadata, not raw medical data, for privacy compliance
type MedicalRecord struct {
	RecordID   string `json:"recordID"`   // Unique identifier for the medical record
	PatientID  string `json:"patientID"`  // Patient identifier (anonymized or pseudonymized)
	HospitalID string `json:"hospitalID"` // MSP ID of the hospital that created the record
	RecordHash string `json:"recordHash"` // SHA-256 hash of the off-chain medical record
	Timestamp  string `json:"timestamp"`  // ISO 8601 timestamp of record creation
	Status     string `json:"status"`     // ACTIVE or REVOKED
}

// EventPayload represents the structure of events emitted for auditability
type EventPayload struct {
	RecordID    string `json:"recordID"`
	Action      string `json:"action"`      // e.g., "MedicalRecordCreated"
	InvokerMSP  string `json:"invokerMSP"` // MSP ID of the transaction invoker
	Timestamp   string `json:"timestamp"`  // ISO 8601 timestamp
	PatientID   string `json:"patientID,omitempty"`
	HospitalID  string `json:"hospitalID,omitempty"`
}

// Constants for record status
const (
	StatusActive  = "ACTIVE"
	StatusRevoked = "REVOKED"
)

// Constants for MSP names (must match network configuration)
const (
	MSPRegulator = "RegulatorMSP"
	MSPHospital1 = "HospitalMSP1"
	MSPHospital2 = "HospitalMSP2"
	MSPResearch  = "ResearchOrgMSP"
)

// CreateMedicalRecord creates a new medical record metadata entry
//
// Access Control:
// - ONLY HospitalMSP organizations (HospitalMSP1, HospitalMSP2) can invoke this function
// - This enforces that only authorized medical institutions can create records
// - Regulatory compliance: Ensures medical records are created by legitimate healthcare providers
//
// Parameters:
//   - recordID: Unique identifier for the medical record
//   - patientID: Patient identifier (should be pseudonymized for privacy)
//   - recordHash: SHA-256 hash of the off-chain medical record data
//
// Returns: Success message or error
//
// Events: Emits "MedicalRecordCreated" event for auditability
func (s *MedicalRecordContract) CreateMedicalRecord(ctx contractapi.TransactionContextInterface, recordID string, patientID string, recordHash string) error {
	// ========== ACCESS CONTROL: Verify invoker is a Hospital MSP ==========
	// This is a critical security check that must be performed at the chaincode level
	// Endorsement policies alone are not sufficient for fine-grained access control
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	if !isHospitalMSP(invokerMSP) {
		return fmt.Errorf("access denied: CreateMedicalRecord can only be invoked by HospitalMSP organizations, got %s", invokerMSP)
	}

	// ========== VALIDATION: Check if record already exists ==========
	existingRecord, err := ctx.GetStub().GetState(recordID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingRecord != nil {
		return fmt.Errorf("medical record %s already exists", recordID)
	}

	// ========== VALIDATION: Verify input parameters ==========
	if recordID == "" {
		return fmt.Errorf("recordID cannot be empty")
	}
	if patientID == "" {
		return fmt.Errorf("patientID cannot be empty")
	}
	if recordHash == "" {
		return fmt.Errorf("recordHash cannot be empty")
	}

	// Validate hash format (should be hex-encoded SHA-256, 64 characters)
	if len(recordHash) != 64 {
		return fmt.Errorf("invalid recordHash format: expected 64-character hex string (SHA-256)")
	}
	// Verify it's valid hex
	if _, err := hex.DecodeString(recordHash); err != nil {
		return fmt.Errorf("invalid recordHash format: not a valid hex string")
	}

	// ========== BUSINESS LOGIC: Create medical record ==========
	timestamp := time.Now().UTC().Format(time.RFC3339)
	record := MedicalRecord{
		RecordID:   recordID,
		PatientID:  patientID,
		HospitalID: invokerMSP, // Set to the MSP that created the record
		RecordHash: recordHash,
		Timestamp:  timestamp,
		Status:     StatusActive,
	}

	recordJSON, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal medical record: %v", err)
	}

	// Store in world state
	err = ctx.GetStub().PutState(recordID, recordJSON)
	if err != nil {
		return fmt.Errorf("failed to put medical record to world state: %v", err)
	}

	// ========== AUDITABILITY: Emit event for compliance tracking ==========
	// Events are critical for regulatory compliance and audit trails
	// They allow external systems to track all medical record operations
	eventPayload := EventPayload{
		RecordID:   recordID,
		Action:     "MedicalRecordCreated",
		InvokerMSP: invokerMSP,
		Timestamp:  timestamp,
		PatientID:  patientID,
		HospitalID: invokerMSP,
	}

	eventJSON, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %v", err)
	}

	err = ctx.GetStub().SetEvent("MedicalRecordCreated", eventJSON)
	if err != nil {
		return fmt.Errorf("failed to emit event: %v", err)
	}

	return nil
}

// GetMedicalRecord retrieves a medical record from the ledger
//
// Access Control:
// - RegulatorMSP: Can access any medical record (regulatory oversight)
// - HospitalMSP: Can only access records they created (own records)
// - ResearchOrgMSP: Not authorized (as per requirements, only Regulator and owning Hospital)
//
// Parameters:
//   - recordID: Unique identifier for the medical record
//
// Returns: MedicalRecord JSON or error
func (s *MedicalRecordContract) GetMedicalRecord(ctx contractapi.TransactionContextInterface, recordID string) (*MedicalRecord, error) {
	// ========== ACCESS CONTROL: Verify invoker has access ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	// Read the record first to check ownership
	recordJSON, err := ctx.GetStub().GetState(recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if recordJSON == nil {
		return nil, fmt.Errorf("medical record %s does not exist", recordID)
	}

	var record MedicalRecord
	err = json.Unmarshal(recordJSON, &record)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal medical record: %v", err)
	}

	// Access control logic:
	// 1. RegulatorMSP can access any record (regulatory oversight)
	// 2. HospitalMSP can only access records they created
	if invokerMSP != MSPRegulator && invokerMSP != record.HospitalID {
		return nil, fmt.Errorf("access denied: GetMedicalRecord can only be invoked by RegulatorMSP or the owning HospitalMSP (record owned by %s, invoker is %s)", record.HospitalID, invokerMSP)
	}

	return &record, nil
}

// VerifyMedicalRecordHash verifies if a provided hash matches the stored hash for a medical record
//
// Access Control:
// - This is a read-only verification function
// - Access control follows the same rules as GetMedicalRecord
// - Used for integrity verification of off-chain medical data
//
// Parameters:
//   - recordID: Unique identifier for the medical record
//   - providedHash: The hash to verify against the stored record hash
//
// Returns: true if hashes match, false otherwise, or error
func (s *MedicalRecordContract) VerifyMedicalRecordHash(ctx contractapi.TransactionContextInterface, recordID string, providedHash string) (bool, error) {
	// ========== VALIDATION: Verify input parameters ==========
	if recordID == "" {
		return false, fmt.Errorf("recordID cannot be empty")
	}
	if providedHash == "" {
		return false, fmt.Errorf("providedHash cannot be empty")
	}

	// Validate hash format
	if len(providedHash) != 64 {
		return false, fmt.Errorf("invalid providedHash format: expected 64-character hex string (SHA-256)")
	}
	if _, err := hex.DecodeString(providedHash); err != nil {
		return false, fmt.Errorf("invalid providedHash format: not a valid hex string")
	}

	// ========== ACCESS CONTROL: Verify invoker has access ==========
	invokerMSP, err := getInvokerMSP(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get invoker MSP: %v", err)
	}

	// Read the record to check access and get stored hash
	recordJSON, err := ctx.GetStub().GetState(recordID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	if recordJSON == nil {
		return false, fmt.Errorf("medical record %s does not exist", recordID)
	}

	var record MedicalRecord
	err = json.Unmarshal(recordJSON, &record)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal medical record: %v", err)
	}

	// Access control: Same as GetMedicalRecord
	if invokerMSP != MSPRegulator && invokerMSP != record.HospitalID {
		return false, fmt.Errorf("access denied: VerifyMedicalRecordHash can only be invoked by RegulatorMSP or the owning HospitalMSP (record owned by %s, invoker is %s)", record.HospitalID, invokerMSP)
	}

	// ========== BUSINESS LOGIC: Compare hashes ==========
	// Case-insensitive comparison (hashes are typically lowercase)
	// Normalize both hashes to lowercase for comparison
	storedHashLower := strings.ToLower(record.RecordHash)
	providedHashLower := strings.ToLower(providedHash)
	match := storedHashLower == providedHashLower

	return match, nil
}

// Helper function to compute SHA-256 hash (for reference, not used in current implementation)
// This can be used by clients to compute hashes before calling CreateMedicalRecord
func ComputeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// ========== ACCESS CONTROL HELPER FUNCTIONS ==========

// getInvokerMSP retrieves the MSP ID of the transaction invoker
// This uses Fabric's Client Identity (CID) library to get the invoker's MSP
func getInvokerMSP(ctx contractapi.TransactionContextInterface) (string, error) {
	// Get the client identity from the transaction context
	clientIdentity, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get client identity MSP ID: %v", err)
	}
	return clientIdentity, nil
}

// isHospitalMSP checks if the given MSP is a Hospital MSP
// This allows for extensibility if more hospitals are added
func isHospitalMSP(mspID string) bool {
	return mspID == MSPHospital1 || mspID == MSPHospital2
}

// ========== MAIN FUNCTION ==========

func main() {
	medicalRecordContract, err := contractapi.NewChaincode(&MedicalRecordContract{})
	if err != nil {
		fmt.Printf("Error creating medical-records chaincode: %v", err)
		return
	}

	if err := medicalRecordContract.Start(); err != nil {
		fmt.Printf("Error starting medical-records chaincode: %v", err)
	}
}
