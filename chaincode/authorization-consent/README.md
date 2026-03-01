# Authorization and Consent Management Chaincode

## Overview

This chaincode implements authorization and consent management for a medical information sharing platform on Hyperledger Fabric. It provides a multi-party authorization system where access to medical records requires both hospital and patient approval, with time-bounded validity and full audit trails.

**Chaincode Name**: `authorization-consent`  
**Language**: Go  
**Fabric Version**: v2.x  
**Channel**: `medical-main-channel`

## Design Principles

### 1. Multi-Party Authorization
- **Hospital Approval Required**: The hospital that owns the medical record must approve access requests
- **Patient Approval Required**: The patient must explicitly approve access to their medical records
- **Research Organization Requests**: Only research organizations can request access
- **Dual Consent Model**: Both parties must approve before access is granted

### 2. Time-Bounded Access
- **Expiration Dates**: All authorization requests have an expiration timestamp
- **Duration-Based**: Requesting organization specifies access duration in hours
- **Automatic Expiration**: Expired authorizations cannot be approved or used

### 3. State Machine Integrity
- **Valid State Transitions**: Enforces strict state machine rules
- **Status Progression**: REQUESTED → HOSPITAL_APPROVED → PATIENT_APPROVED → GRANTED
- **Revocation**: Any state can transition to REVOKED
- **Order Independence**: Patient and hospital can approve in any order

### 4. MSP and Client Identity-Based Access Control
- **MSP Verification**: Uses Fabric's Client Identity API to verify invoker MSP
- **Patient Identity Verification**: Verifies client ID (X.509 CN) matches patientID
- **Fine-Grained Control**: Different functions have different access requirements
- **Defense in Depth**: Chaincode-level checks complement endorsement policies

### 5. Full Auditability
- **Event Emission**: All state transitions emit events
- **Complete Context**: Events include invoker MSP, client ID, timestamp, and status
- **Compliance Tracking**: Enables regulatory compliance and audit requirements

## Data Model

### AuthorizationRequest Structure

```json
{
  "requestID": "AUTH-REC-2024-001-1234567890",
  "recordID": "REC-2024-001",
  "requesterOrg": "ResearchOrgMSP",
  "hospitalID": "HospitalMSP1",
  "patientID": "x509::CN=patient123,OU=client,O=hospital1::CN=patient123,OU=client,O=hospital1",
  "status": "GRANTED",
  "createdAt": "2024-01-15T10:30:00Z",
  "expiresAt": "2024-01-22T10:30:00Z"
}
```

**Fields**:
- `requestID`: Unique identifier for the authorization request (used as ledger key)
- `recordID`: ID of the medical record being requested (must exist in medical-records chaincode)
- `requesterOrg`: MSP ID of the organization requesting access (ResearchOrgMSP)
- `hospitalID`: MSP ID of the hospital that owns the medical record
- `patientID`: Client ID (X.509 certificate CN) of the patient
- `status`: Current authorization status (REQUESTED, HOSPITAL_APPROVED, PATIENT_APPROVED, GRANTED, REVOKED)
- `createdAt`: ISO 8601 timestamp of request creation
- `expiresAt`: ISO 8601 timestamp when authorization expires

### Status State Machine

```
REQUESTED
    ├──> HOSPITAL_APPROVED ──> GRANTED (when patient approves)
    ├──> PATIENT_APPROVED ──> GRANTED (when hospital approves)
    └──> REVOKED

HOSPITAL_APPROVED
    ├──> GRANTED (when patient approves)
    └──> REVOKED

PATIENT_APPROVED
    ├──> GRANTED (when hospital approves)
    └──> REVOKED

GRANTED
    └──> REVOKED
```

## Chaincode Functions

### 1. RequestAccess

Creates a new authorization request for accessing a medical record.

**Function Signature**:
```go
RequestAccess(ctx, recordID, duration) (requestID, error)
```

**Parameters**:
- `recordID` (string): Unique identifier for the medical record being requested
- `duration` (int): Duration in hours for which access is requested (must be > 0)

**Returns**: `requestID` (string) - Generated unique identifier for the authorization request

**Access Control**:
- ✅ **Allowed**: ResearchOrgMSP only
- ❌ **Denied**: All other MSPs (RegulatorMSP, HospitalMSPs, patients)

**Validation**:
- Verifies invoker is ResearchOrgMSP
- Validates recordID is non-empty
- Validates duration > 0
- Verifies medical record exists (queries medical-records chaincode)
- Verifies medical record is ACTIVE
- Generates unique requestID

**Event**: Emits `AuthorizationRequested` event

**State**: Creates authorization request in `REQUESTED` status

**Example**:
```bash
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"RequestAccess","Args":["REC-2024-001","168"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### 2. ApproveByHospital

Approves an authorization request by the hospital that owns the medical record.

**Function Signature**:
```go
ApproveByHospital(ctx, requestID) error
```

**Parameters**:
- `requestID` (string): Unique identifier for the authorization request

**Access Control**:
- ✅ **Allowed**: Owning HospitalMSP (the hospital that created the medical record)
- ❌ **Denied**: 
  - Other HospitalMSPs (cannot approve requests for other hospitals' records)
  - RegulatorMSP
  - ResearchOrgMSP
  - Patients

**Validation**:
- Verifies invoker is a Hospital MSP
- Verifies invoker is the owning hospital (matches HospitalID in request)
- Validates request exists
- Validates current status is REQUESTED
- Checks authorization has not expired

**Event**: Emits `HospitalApproved` event

**State Transition**: `REQUESTED` → `HOSPITAL_APPROVED`

**Example**:
```bash
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByHospital","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### 3. ApproveByPatient

Approves an authorization request by the patient.

**Function Signature**:
```go
ApproveByPatient(ctx, requestID) error
```

**Parameters**:
- `requestID` (string): Unique identifier for the authorization request

**Access Control**:
- ✅ **Allowed**: Patient (client ID must match patientID in the authorization request)
- ❌ **Denied**: 
  - All MSPs (hospitals, regulator, research)
  - Other patients

**Validation**:
- Verifies invoker's client ID matches patientID in the request
- Validates request exists
- Validates current status is REQUESTED or HOSPITAL_APPROVED
- Checks authorization has not expired

**Event**: Emits `PatientApproved` event

**State Transitions**:
- `REQUESTED` → `PATIENT_APPROVED` (if hospital hasn't approved yet)
- `HOSPITAL_APPROVED` → `GRANTED` (if hospital already approved)

**Example**:
```bash
# Patient must use their client identity (X.509 certificate)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByPatient","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### 4. RevokeAuthorization

Revokes an authorization request. Can be called by either the patient or the owning hospital.

**Function Signature**:
```go
RevokeAuthorization(ctx, requestID) error
```

**Parameters**:
- `requestID` (string): Unique identifier for the authorization request

**Access Control**:
- ✅ **Allowed**: 
  - Patient (client ID must match patientID)
  - Owning HospitalMSP (must match HospitalID)
- ❌ **Denied**: 
  - Other HospitalMSPs
  - RegulatorMSP
  - ResearchOrgMSP

**Validation**:
- Verifies invoker is patient or owning hospital
- Validates request exists
- Checks request is not already revoked

**Event**: Emits `AuthorizationRevoked` event

**State Transition**: Any status → `REVOKED`

**Example**:
```bash
# As patient
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"RevokeAuthorization","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

### 5. GetAuthorization

Retrieves an authorization request from the ledger (read-only).

**Function Signature**:
```go
GetAuthorization(ctx, requestID) (*AuthorizationRequest, error)
```

**Parameters**:
- `requestID` (string): Unique identifier for the authorization request

**Access Control**:
- ✅ **Allowed**: 
  - RegulatorMSP (can access any authorization request - regulatory oversight)
  - Owning HospitalMSP (can access authorization requests for their records)
  - Patient (can access authorization requests where they are the patient)
  - Requesting ResearchOrg (can access authorization requests they created)
- ❌ **Denied**: 
  - Other HospitalMSPs (cannot access requests for other hospitals' records)
  - Other ResearchOrgs (cannot access requests created by other research orgs)

**Returns**: AuthorizationRequest JSON object

**Example**:
```bash
peer chaincode query \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"GetAuthorization","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

## Access Control Architecture

### Design Philosophy

The chaincode implements **multi-layer access control** with both MSP-based and client identity-based checks:

1. **MSP-Level Control**: Verifies which organization is invoking (ResearchOrgMSP, HospitalMSP, RegulatorMSP)
2. **Client Identity Control**: Verifies specific client identity (patient ID matching)
3. **Ownership-Based Control**: Verifies ownership relationships (hospital owns record, patient owns authorization)

### Why Chaincode-Level Access Control?

#### Limitations of Endorsement Policies Alone

Endorsement policies control transaction validation but cannot enforce:
- Function-specific access rules (e.g., only ResearchOrg can request)
- Ownership-based access (e.g., only owning hospital can approve)
- Client identity matching (e.g., patient ID must match)

#### Benefits of Chaincode-Level Checks

1. **Function-Specific Rules**: Different functions have different access requirements
   - RequestAccess: Only ResearchOrgMSP
   - ApproveByHospital: Only owning HospitalMSP
   - ApproveByPatient: Only matching patient client ID

2. **Ownership Verification**: Hospitals can only approve requests for their own records
   - Prevents HospitalMSP2 from approving HospitalMSP1's record requests

3. **Patient Identity Verification**: Uses Fabric's Client Identity API to verify patient
   - Critical for consent management
   - Ensures only the actual patient can approve

4. **Multi-Party Authorization**: Enforces dual consent model
   - Both hospital and patient must approve
   - Order-independent approval flow

### Implementation Details

#### MSP Identification

```go
func getInvokerMSP(ctx contractapi.TransactionContextInterface) (string, error) {
    clientIdentity, err := ctx.GetClientIdentity().GetMSPID()
    if err != nil {
        return "", fmt.Errorf("failed to get client identity MSP ID: %v", err)
    }
    return clientIdentity, nil
}
```

#### Client Identity (Patient) Verification

```go
func getClientID(ctx contractapi.TransactionContextInterface) (string, error) {
    id, err := ctx.GetClientIdentity().GetID()
    if err != nil {
        return "", fmt.Errorf("failed to get client identity ID: %v", err)
    }
    return id, nil
}
```

**Critical Security Check**:
```go
if clientID != authRequest.PatientID {
    return fmt.Errorf("access denied: only the patient can approve")
}
```

### Access Control Matrix

| Function | RegulatorMSP | HospitalMSP (own) | HospitalMSP (other) | ResearchOrgMSP | Patient (own) | Patient (other) |
|----------|-------------|-------------------|---------------------|----------------|--------------|------------------|
| **RequestAccess** | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ |
| **ApproveByHospital** | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| **ApproveByPatient** | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ |
| **RevokeAuthorization** | ❌ | ✅ (own) | ❌ | ❌ | ✅ (own) | ❌ |
| **GetAuthorization** | ✅ (all) | ✅ (own) | ❌ | ✅ (own requests) | ✅ (own) | ❌ |

**Legend**:
- ✅ = Allowed
- ❌ = Denied
- (own) = Only for own records/requests
- (all) = Can access any authorization request

## Events

### AuthorizationRequested Event

Emitted when a new authorization request is created.

**Event Name**: `AuthorizationRequested`

**Event Payload**:
```json
{
  "requestID": "AUTH-REC-2024-001-1234567890",
  "recordID": "REC-2024-001",
  "action": "AuthorizationRequested",
  "invokerMSP": "ResearchOrgMSP",
  "clientID": "x509::CN=researcher1...",
  "timestamp": "2024-01-15T10:30:00Z",
  "patientID": "x509::CN=patient123...",
  "hospitalID": "HospitalMSP1",
  "requesterOrg": "ResearchOrgMSP",
  "status": "REQUESTED"
}
```

### HospitalApproved Event

Emitted when a hospital approves an authorization request.

**Event Name**: `HospitalApproved`

**Event Payload**: Similar structure with `action: "HospitalApproved"` and `status: "HOSPITAL_APPROVED"`

### PatientApproved Event

Emitted when a patient approves an authorization request.

**Event Name**: `PatientApproved`

**Event Payload**: Similar structure with `action: "PatientApproved"` and `status: "PATIENT_APPROVED"` or `"GRANTED"`

### AuthorizationRevoked Event

Emitted when an authorization is revoked.

**Event Name**: `AuthorizationRevoked`

**Event Payload**: Similar structure with `action: "AuthorizationRevoked"` and `status: "REVOKED"`

## End-to-End Authorization Flow

### Complete Workflow Example

**Scenario**: ResearchOrgMSP requests access to a medical record, hospital and patient approve, then patient revokes.

#### Step 1: Research Organization Requests Access

```bash
# As ResearchOrgMSP
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"RequestAccess","Args":["REC-2024-001","168"]}' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Authorization request created with status `REQUESTED`, requestID returned.

#### Step 2: Hospital Approves

```bash
# As HospitalMSP1 (owner of REC-2024-001)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByHospital","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Status changes to `HOSPITAL_APPROVED`.

#### Step 3: Patient Approves

```bash
# As Patient (using patient client identity)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"ApproveByPatient","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Status changes to `GRANTED` (both parties approved).

#### Step 4: Patient Revokes (Optional)

```bash
# As Patient
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{"function":"RevokeAuthorization","Args":["AUTH-REC-2024-001-1234567890"]}' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt
```

**Result**: Status changes to `REVOKED`.

### Alternative Flow: Patient Approves First

The approval order is flexible:

1. ResearchOrg requests → Status: `REQUESTED`
2. Patient approves → Status: `PATIENT_APPROVED`
3. Hospital approves → Status: `GRANTED`

## Deployment

### Prerequisites

- Hyperledger Fabric v2.x network running
- Channel `medical-main-channel` created and peers joined
- `medical-records` chaincode deployed (authorization-consent queries it)
- Go 1.20+ installed
- Fabric chaincode dependencies

### Build Chaincode

```bash
cd chaincode/authorization-consent
go mod download
go mod vendor  # Optional: vendor dependencies
```

### Package Chaincode

```bash
peer lifecycle chaincode package authorization-consent.tar.gz \
  --path ./chaincode/authorization-consent \
  --lang golang \
  --label authorization-consent_1.0
```

### Install Chaincode

Install on all peer organizations:

```bash
# Install on RegulatorMSP peer
peer lifecycle chaincode install authorization-consent.tar.gz

# Install on HospitalMSP1 peer
peer lifecycle chaincode install authorization-consent.tar.gz

# Install on HospitalMSP2 peer
peer lifecycle chaincode install authorization-consent.tar.gz

# Install on ResearchOrgMSP peer
peer lifecycle chaincode install authorization-consent.tar.gz
```

### Approve Chaincode

Approve from each organization (example for RegulatorMSP):

```bash
peer lifecycle chaincode approveformyorg \
  -o orderer0.medical-network.com:7050 \
  --channelID medical-main-channel \
  --name authorization-consent \
  --version 1.0 \
  --package-id <PACKAGE_ID> \
  --sequence 1 \
  --tls \
  --cafile /path/to/orderer/tls/ca.crt
```

### Commit Chaincode

```bash
peer lifecycle chaincode commit \
  -o orderer0.medical-network.com:7050 \
  --channelID medical-main-channel \
  --name authorization-consent \
  --version 1.0 \
  --sequence 1 \
  --tls \
  --cafile /path/to/orderer/tls/ca.crt \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/regulator/tls/ca.crt \
  --tlsRootCertFiles /path/to/hospital1/tls/ca.crt \
  --tlsRootCertFiles /path/to/hospital2/tls/ca.crt \
  --tlsRootCertFiles /path/to/research/tls/ca.crt
```

## Security Considerations

### Access Control
- ✅ MSP-based access control enforced at chaincode level
- ✅ Client Identity API used to verify patient identity
- ✅ Function-specific access rules
- ✅ Ownership-based authorization access
- ✅ Patient identity matching for consent

### Consent Management
- ✅ Dual consent model (hospital + patient)
- ✅ Patient identity verification using X.509 CN
- ✅ Revocation capability for both parties
- ✅ Time-bounded access with expiration

### Auditability
- ✅ All state transitions emit events
- ✅ Events include invoker MSP, client ID, and timestamp
- ✅ Complete transaction history on ledger
- ✅ Full audit trail for compliance

### Future Enhancements
- Private data collections for sensitive authorization data
- Encryption for additional security layers
- Payment integration (out of scope for now)
- Advanced consent management features

## Compliance Alignment

This chaincode design aligns with:

- **HIPAA**: Patient consent requirements, access control, audit trails
- **GDPR**: Right to consent, right to revoke, access control
- **Medical Data Privacy Laws**: Multi-party authorization, patient consent
- **Regulatory Oversight**: RegulatorMSP can access all authorization requests

## Troubleshooting

### Access Denied Errors

If you receive "access denied" errors:
1. Verify the client identity is correctly authenticated
2. Check that the MSP ID matches expected values
3. For ApproveByPatient, ensure client ID matches patientID
4. For ApproveByHospital, ensure invoker is the owning hospital

### Authorization Request Not Found

If authorization request doesn't exist:
1. Verify requestID is correct
2. Check that RequestAccess was successfully invoked
3. Verify the request was created on the same channel

### Expired Authorization

If authorization has expired:
1. Check the expiresAt timestamp
2. Create a new authorization request with appropriate duration
3. Ensure duration is set correctly when requesting access

### Patient Identity Mismatch

If patient approval fails:
1. Verify the client identity (X.509 CN) matches patientID in the medical record
2. Ensure patient is using the correct certificate
3. Check that patientID in medical record matches patientID in authorization request

## References

- [Hyperledger Fabric Chaincode Documentation](https://hyperledger-fabric.readthedocs.io/en/latest/chaincode4ade.html)
- [Fabric Contract API Go](https://github.com/hyperledger/fabric-contract-api-go)
- [Client Identity Library](https://github.com/hyperledger/fabric-chaincode-go)
