# Authorization and Consent Management - State Transition Diagram

## State Machine Overview

The authorization-consent chaincode implements a state machine that enforces valid authorization flows. The state machine ensures that both hospital and patient must approve before access is granted, and supports revocation at any stage.

## States

1. **REQUESTED**: Initial state when research organization requests access
2. **HOSPITAL_APPROVED**: Hospital has approved, waiting for patient approval
3. **PATIENT_APPROVED**: Patient has approved, waiting for hospital approval
4. **GRANTED**: Both parties have approved, access is granted
5. **REVOKED**: Authorization has been revoked (terminal state)

## State Transition Diagram

```
                    REQUESTED
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        │              │              │
        ▼              ▼              ▼
HOSPITAL_APPROVED  PATIENT_APPROVED  REVOKED
        │              │              │
        │              │              │
        ├──────────────┘              │
        │                             │
        ▼                             │
     GRANTED                          │
        │                             │
        └─────────────────────────────┘
```

## Valid State Transitions

### From REQUESTED
- **REQUESTED → HOSPITAL_APPROVED**: When hospital approves first
  - Function: `ApproveByHospital`
  - Access: Owning HospitalMSP

- **REQUESTED → PATIENT_APPROVED**: When patient approves first
  - Function: `ApproveByPatient`
  - Access: Patient (client ID must match patientID)

- **REQUESTED → REVOKED**: When authorization is revoked before any approval
  - Function: `RevokeAuthorization`
  - Access: Patient OR Owning HospitalMSP

### From HOSPITAL_APPROVED
- **HOSPITAL_APPROVED → GRANTED**: When patient approves after hospital
  - Function: `ApproveByPatient`
  - Access: Patient (client ID must match patientID)

- **HOSPITAL_APPROVED → REVOKED**: When authorization is revoked after hospital approval
  - Function: `RevokeAuthorization`
  - Access: Patient OR Owning HospitalMSP

### From PATIENT_APPROVED
- **PATIENT_APPROVED → GRANTED**: When hospital approves after patient
  - Function: `ApproveByHospital`
  - Access: Owning HospitalMSP

- **PATIENT_APPROVED → REVOKED**: When authorization is revoked after patient approval
  - Function: `RevokeAuthorization`
  - Access: Patient OR Owning HospitalMSP

### From GRANTED
- **GRANTED → REVOKED**: When authorization is revoked after access is granted
  - Function: `RevokeAuthorization`
  - Access: Patient OR Owning HospitalMSP

### Terminal State
- **REVOKED**: Terminal state - no transitions allowed from REVOKED

## Invalid State Transitions

The following transitions are **NOT allowed** and will result in errors:

- ❌ **REQUESTED → GRANTED**: Cannot skip intermediate states
- ❌ **HOSPITAL_APPROVED → PATIENT_APPROVED**: Invalid transition
- ❌ **PATIENT_APPROVED → HOSPITAL_APPROVED**: Invalid transition
- ❌ **GRANTED → REQUESTED**: Cannot reset to initial state
- ❌ **GRANTED → HOSPITAL_APPROVED**: Cannot go back to intermediate state
- ❌ **GRANTED → PATIENT_APPROVED**: Cannot go back to intermediate state
- ❌ **REVOKED → Any state**: REVOKED is terminal

## State Transition Examples

### Example 1: Hospital Approves First

```
1. ResearchOrg requests access
   → Status: REQUESTED

2. Hospital approves
   → Status: HOSPITAL_APPROVED

3. Patient approves
   → Status: GRANTED
```

### Example 2: Patient Approves First

```
1. ResearchOrg requests access
   → Status: REQUESTED

2. Patient approves
   → Status: PATIENT_APPROVED

3. Hospital approves
   → Status: GRANTED
```

### Example 3: Revocation Before Approval

```
1. ResearchOrg requests access
   → Status: REQUESTED

2. Patient revokes
   → Status: REVOKED (terminal)
```

### Example 4: Revocation After Grant

```
1. ResearchOrg requests access
   → Status: REQUESTED

2. Hospital approves
   → Status: HOSPITAL_APPROVED

3. Patient approves
   → Status: GRANTED

4. Patient revokes
   → Status: REVOKED (terminal)
```

## State Validation Logic

### ApproveByHospital Validation

```go
// Hospital can approve if status is:
// - REQUESTED → HOSPITAL_APPROVED
// - PATIENT_APPROVED → GRANTED
if authRequest.Status != StatusRequested && authRequest.Status != StatusPatientApproved {
    return error("invalid state transition")
}
```

### ApproveByPatient Validation

```go
// Patient can approve if status is:
// - REQUESTED → PATIENT_APPROVED
// - HOSPITAL_APPROVED → GRANTED
if authRequest.Status != StatusRequested && authRequest.Status != StatusHospitalApproved {
    return error("invalid state transition")
}
```

### RevokeAuthorization Validation

```go
// Can revoke from any state except REVOKED
if authRequest.Status == StatusRevoked {
    return error("already revoked")
}
// Any status → REVOKED
```

## Expiration Handling

All state transitions (except REVOKED) check if the authorization has expired:

```go
if isExpired(authRequest.ExpiresAt) {
    return error("authorization request has expired")
}
```

**Note**: Expired authorizations cannot be approved, but can still be revoked.

## Access Control Matrix by State

| State | RequestAccess | ApproveByHospital | ApproveByPatient | RevokeAuthorization | GetAuthorization |
|-------|---------------|-------------------|------------------|---------------------|------------------|
| **REQUESTED** | ❌ (already exists) | ✅ (own hospital) | ✅ (patient) | ✅ (patient/hospital) | ✅ (authorized parties) |
| **HOSPITAL_APPROVED** | ❌ | ❌ (invalid transition) | ✅ (patient) | ✅ (patient/hospital) | ✅ (authorized parties) |
| **PATIENT_APPROVED** | ❌ | ✅ (own hospital) | ❌ (invalid transition) | ✅ (patient/hospital) | ✅ (authorized parties) |
| **GRANTED** | ❌ | ❌ (invalid transition) | ❌ (invalid transition) | ✅ (patient/hospital) | ✅ (authorized parties) |
| **REVOKED** | ❌ | ❌ (terminal state) | ❌ (terminal state) | ❌ (already revoked) | ✅ (authorized parties) |

**Legend**:
- ✅ = Allowed operation
- ❌ = Not allowed (invalid transition or terminal state)

## State Machine Properties

### Safety Properties

1. **No Skipping States**: Cannot go directly from REQUESTED to GRANTED
2. **Dual Consent Required**: Both hospital and patient must approve before GRANTED
3. **Terminal Revocation**: REVOKED is terminal - cannot transition from REVOKED
4. **Expiration Check**: Cannot approve expired authorizations

### Liveness Properties

1. **Flexible Order**: Hospital and patient can approve in any order
2. **Revocation Always Possible**: Can revoke from any non-terminal state
3. **Query Always Possible**: Can query authorization from any state (with proper access)

## Implementation Notes

- State transitions are enforced in chaincode logic, not just in documentation
- All state transitions emit events for auditability
- State validation occurs before any state change
- Expiration checks prevent operations on expired authorizations
- Access control is enforced at each state transition
