package fabric

import (
	"database/sql"
	"fmt"

	"github.com/hyperledger/medical-middleware/internal/storage"
)

// IdentityService handles identity mapping and role management
type IdentityService struct {
	db *storage.Database
}

// Role represents user roles
type Role string

const (
	RolePatient   Role = "PATIENT"
	RoleHospital  Role = "HOSPITAL"
	RoleResearch  Role = "RESEARCH"
	RoleRegulator Role = "REGULATOR"
)

// Identity represents a user identity
type Identity struct {
	CertificateID string
	MSPID         string
	Role          Role
	UserID        string
}

// NewIdentityService creates a new identity service
func NewIdentityService(db *storage.Database) *IdentityService {
	return &IdentityService{db: db}
}

// RegisterIdentity registers a new identity mapping
func (s *IdentityService) RegisterIdentity(certificateID, mspID string, role Role, userID string) error {
	_, err := s.db.GetConnection().Exec(
		"INSERT INTO identity_mappings (certificate_id, msp_id, role, user_id) VALUES ($1, $2, $3, $4) ON CONFLICT (certificate_id) DO UPDATE SET msp_id = $2, role = $3, user_id = $4",
		certificateID, mspID, role, userID,
	)
	return err
}

// GetIdentity retrieves an identity by certificate ID
func (s *IdentityService) GetIdentity(certificateID string) (*Identity, error) {
	var identity Identity
	err := s.db.GetConnection().QueryRow(
		"SELECT certificate_id, msp_id, role, user_id FROM identity_mappings WHERE certificate_id = $1",
		certificateID,
	).Scan(&identity.CertificateID, &identity.MSPID, &identity.Role, &identity.UserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("identity not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query identity: %w", err)
	}

	return &identity, nil
}

// GetIdentityByMSPID retrieves identities by MSP ID
func (s *IdentityService) GetIdentityByMSPID(mspID string) ([]*Identity, error) {
	rows, err := s.db.GetConnection().Query(
		"SELECT certificate_id, msp_id, role, user_id FROM identity_mappings WHERE msp_id = $1",
		mspID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query identities: %w", err)
	}
	defer rows.Close()

	var identities []*Identity
	for rows.Next() {
		var identity Identity
		if err := rows.Scan(&identity.CertificateID, &identity.MSPID, &identity.Role, &identity.UserID); err != nil {
			return nil, fmt.Errorf("failed to scan identity: %w", err)
		}
		identities = append(identities, &identity)
	}

	return identities, nil
}

// ValidateRole validates that an identity has the required role
func (s *IdentityService) ValidateRole(certificateID string, requiredRole Role) error {
	identity, err := s.GetIdentity(certificateID)
	if err != nil {
		return err
	}

	if identity.Role != requiredRole {
		return fmt.Errorf("insufficient permissions: required role %s, got %s", requiredRole, identity.Role)
	}

	return nil
}

// ValidateMSP validates that an identity belongs to the required MSP
func (s *IdentityService) ValidateMSP(certificateID string, requiredMSP string) error {
	identity, err := s.GetIdentity(certificateID)
	if err != nil {
		return err
	}

	if identity.MSPID != requiredMSP {
		return fmt.Errorf("insufficient permissions: required MSP %s, got %s", requiredMSP, identity.MSPID)
	}

	return nil
}

// IsHospitalMSP checks if an MSP is a hospital MSP
func IsHospitalMSP(mspID string) bool {
	return mspID == "HospitalMSP1" || mspID == "HospitalMSP2"
}

// IsResearchMSP checks if an MSP is a research MSP
func IsResearchMSP(mspID string) bool {
	return mspID == "ResearchOrgMSP"
}

// IsRegulatorMSP checks if an MSP is a regulator MSP
func IsRegulatorMSP(mspID string) bool {
	return mspID == "RegulatorMSP"
}
