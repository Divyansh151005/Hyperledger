# Endorsement Policies for Medical Information Sharing Network

## Overview

This document defines the endorsement policies for the Hyperledger Fabric medical information sharing network. These policies ensure that transactions are validated according to the regulatory and security requirements of the healthcare domain.

## Policy Definitions

### 1. Medical Record Creation Policy

**Policy Expression:**
```
AND('RegulatorMSP.peer', 'HospitalMSP.peer')
```

**Description:**
This policy requires endorsement from both:
- A peer from the Regulatory Authority (RegulatorMSP)
- A peer from a Hospital organization (HospitalMSP1 or HospitalMSP2)

**Rationale:**
This policy aligns with the Regulatory Authority model where:
- **Regulatory Oversight**: The regulator must validate all medical record entries to ensure compliance with healthcare regulations, data privacy laws, and medical standards.
- **Medical Authority**: A hospital must validate the medical accuracy and authenticity of the record.
- **Dual Validation**: Both entities must agree before a medical record is committed to the ledger, preventing unauthorized or non-compliant data entry.

**Implementation Notes:**
- When implementing chaincode, this policy should be applied to functions that create new medical records.
- The policy ensures that no medical record can be created without regulatory approval.
- This creates an audit trail showing both medical and regulatory validation.

**Example Chaincode Policy (for future implementation):**
```go
// Policy for CreateMedicalRecord function
policy := "AND('RegulatorMSP.peer', OR('HospitalMSP1.peer', 'HospitalMSP2.peer'))"
```

### 2. Authorization Approval Policy

**Policy Expression:**
```
AND('PatientIdentity', 'HospitalMSP.peer')
```

**Description:**
This policy requires endorsement from:
- The patient's identity (via client certificate)
- A peer from a Hospital organization

**Rationale:**
This policy ensures:
- **Patient Consent**: The patient must explicitly authorize data sharing through their client certificate, ensuring patient autonomy and consent.
- **Medical Validation**: A hospital peer validates that the authorization request is legitimate and medically appropriate.
- **Privacy Protection**: Patient data cannot be shared without explicit patient consent, even if a hospital approves.

**Implementation Notes:**
- Patient identities will be registered with the Fabric CA and issued client certificates.
- The patient's client certificate must be used when submitting authorization transactions.
- This policy enforces the principle of patient-controlled data sharing.

**Example Chaincode Policy (for future implementation):**
```go
// Policy for ApproveAuthorization function
// PatientIdentity refers to a specific patient's client certificate
policy := "AND('PatientIdentity', OR('HospitalMSP1.peer', 'HospitalMSP2.peer'))"
```

## Policy Architecture Alignment

### Regulatory Authority Model

The endorsement policies are designed to support a regulatory authority model where:

1. **RegulatorMSP** acts as the oversight body:
   - Validates all medical record entries
   - Ensures compliance with healthcare regulations
   - Maintains audit trails for regulatory purposes

2. **Hospital Organizations** provide medical expertise:
   - Validate medical accuracy
   - Ensure proper medical protocols are followed
   - Maintain clinical integrity

3. **Patient Identity** ensures patient autonomy:
   - Patients control their data sharing
   - Explicit consent required for all authorizations
   - Privacy-first approach

### Security Considerations

1. **Multi-Organization Validation**: Policies require multiple organizations to agree, preventing single points of failure or corruption.

2. **Patient Control**: Authorization policies ensure patients maintain control over their data.

3. **Regulatory Compliance**: Medical record creation requires regulatory approval, ensuring all data meets compliance standards.

4. **Audit Trail**: All endorsements are recorded, providing a complete audit trail for compliance and investigation purposes.

## Future Extensions

### Private Data Collections

When implementing private data collections:
- Medical records can be stored in private collections visible only to authorized parties
- Endorsement policies will still apply to the private data transactions
- Regulatory authority will have access to audit private data transactions

### Additional Policies

Future policies may include:
- **Research Data Access**: `AND('ResearchOrgMSP.peer', 'PatientIdentity', 'RegulatorMSP.peer')`
- **Emergency Access**: `OR('HospitalMSP.peer', 'RegulatorMSP.peer')` (for emergency situations)
- **Data Deletion**: `AND('PatientIdentity', 'RegulatorMSP.peer')` (for right to be forgotten)

## Policy Testing

When chaincode is implemented, policies should be tested with:
1. Valid endorsements from required organizations
2. Invalid endorsements (missing required organizations)
3. Patient identity validation
4. Multi-organization scenarios

## References

- Hyperledger Fabric Documentation: [Endorsement Policies](https://hyperledger-fabric.readthedocs.io/en/latest/endorsement-policies.html)
- Fabric CA Documentation: [Client Certificates](https://hyperledger-fabric-ca.readthedocs.io/en/latest/users-guide.html)
