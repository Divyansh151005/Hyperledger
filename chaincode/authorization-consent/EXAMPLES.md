# Authorization and Consent Management Chaincode - Usage Examples

This document provides practical examples for using the authorization-consent chaincode.

## Prerequisites

- Network is running and channel is created
- `medical-records` chaincode is deployed (authorization-consent queries it)
- `authorization-consent` chaincode is installed and committed
- Client certificates are available for each organization
- Patient client identities are created (X.509 certificates with CN matching patientID)

## Example 1: Complete Authorization Flow

### Step 1: Create Medical Record (Prerequisite)

First, a medical record must exist. Create it using the medical-records chaincode:

```bash
# As HospitalMSP1
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Create medical record
peer chaincode invoke \
  -C medical-main-channel \
  -n medical-records \
  -c '{
    "function":"CreateMedicalRecord",
    "Args":[
      "REC-2024-001",
      "x509::CN=patient123,OU=client,O=hospital1::CN=patient123,OU=client,O=hospital1",
      "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"
    ]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

### Step 2: Research Organization Requests Access

```bash
# As ResearchOrgMSP
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/research.medical-network.com/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:10051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt

# Request access for 7 days (168 hours)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"RequestAccess",
    "Args":["REC-2024-001","168"]
  }' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
2024-01-15 10:30:00.123 UTC [chaincodeCmd] chaincodeInvokeOrQuery -> INFO 001 Chaincode invoke successful. result: status:200
```

**Note**: The function returns a requestID. In a real application, you would capture this from the transaction response. For this example, assume the requestID is `AUTH-REC-2024-001-1705315800123456789`.

### Step 3: Hospital Approves

```bash
# As HospitalMSP1 (owner of REC-2024-001)
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Approve authorization request
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"ApproveByHospital",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
2024-01-15 10:35:00.456 UTC [chaincodeCmd] chaincodeInvokeOrQuery -> INFO 001 Chaincode invoke successful. result: status:200
```

### Step 4: Patient Approves

```bash
# As Patient (using patient client identity)
# The patient must use their X.509 certificate with CN matching patientID
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/patient123@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Approve authorization request
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"ApproveByPatient",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
2024-01-15 10:40:00.789 UTC [chaincodeCmd] chaincodeInvokeOrQuery -> INFO 001 Chaincode invoke successful. result: status:200
```

**Status**: Authorization is now `GRANTED` (both parties approved).

### Step 5: Query Authorization Status

```bash
# As RegulatorMSP (can query any authorization)
export CORE_PEER_LOCALMSPID=RegulatorMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/users/Admin@regulator.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt

# Query authorization request
peer chaincode query \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"GetAuthorization",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt
```

**Expected Output**:
```json
{
  "requestID": "AUTH-REC-2024-001-1705315800123456789",
  "recordID": "REC-2024-001",
  "requesterOrg": "ResearchOrgMSP",
  "hospitalID": "HospitalMSP1",
  "patientID": "x509::CN=patient123,OU=client,O=hospital1::CN=patient123,OU=client,O=hospital1",
  "status": "GRANTED",
  "createdAt": "2024-01-15T10:30:00Z",
  "expiresAt": "2024-01-22T10:30:00Z"
}
```

## Example 2: Alternative Flow - Patient Approves First

### Step 1: Research Organization Requests Access

(Same as Example 1, Step 2)

### Step 2: Patient Approves First

```bash
# As Patient
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/patient123@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"ApproveByPatient",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Status**: Changes to `PATIENT_APPROVED`.

### Step 3: Hospital Approves

```bash
# As HospitalMSP1
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"ApproveByHospital",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Status**: Changes to `GRANTED` (both parties approved).

## Example 3: Revocation Flow

### Step 1-4: Complete Authorization (from Example 1)

Authorization is now `GRANTED`.

### Step 5: Patient Revokes Authorization

```bash
# As Patient
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/patient123@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Revoke authorization
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"RevokeAuthorization",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Status**: Changes to `REVOKED`.

### Alternative: Hospital Revokes

```bash
# As HospitalMSP1
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"RevokeAuthorization",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

## Example 4: Unauthorized Access Attempts

### Attempt 1: HospitalMSP2 Tries to Approve HospitalMSP1's Record

```bash
# As HospitalMSP2 (not the owner)
export CORE_PEER_LOCALMSPID=HospitalMSP2
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital2.medical-network.com/users/Admin@hospital2.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital2.medical-network.com:9051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital2.medical-network.com/peers/peer0.hospital2.medical-network.com/tls/ca.crt

# Attempt to approve (will fail)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"ApproveByHospital",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital2.medical-network.com/peers/peer0.hospital2.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
Error: access denied: only the owning hospital (HospitalMSP1) can approve this request, invoker is HospitalMSP2
```

### Attempt 2: ResearchOrgMSP Tries to Request with Invalid Duration

```bash
# As ResearchOrgMSP
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/research.medical-network.com/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:10051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt

# Attempt to request with invalid duration (will fail)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"RequestAccess",
    "Args":["REC-2024-001","0"]
  }' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
Error: duration must be greater than 0 hours
```

### Attempt 3: Wrong Patient Tries to Approve

```bash
# As Different Patient (patientID mismatch)
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/patient456@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Attempt to approve (will fail)
peer chaincode invoke \
  -C medical-main-channel \
  -n authorization-consent \
  -c '{
    "function":"ApproveByPatient",
    "Args":["AUTH-REC-2024-001-1705315800123456789"]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
Error: access denied: only the patient (patientID: x509::CN=patient123...) can approve this request, invoker client ID is x509::CN=patient456...
```

## Example 5: Complete Workflow Script

```bash
#!/bin/bash

# Configuration
CHANNEL="medical-main-channel"
CHAINCODE="authorization-consent"
MEDICAL_CHAINCODE="medical-records"
RECORD_ID="REC-2024-003"
PATIENT_ID="x509::CN=patient789,OU=client,O=hospital1::CN=patient789,OU=client,O=hospital1"
DURATION="168"  # 7 days in hours

echo "=== Authorization and Consent Management Workflow ==="

# Step 1: Create medical record (as HospitalMSP1)
echo "Step 1: Creating medical record..."
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode invoke \
  -C $CHANNEL \
  -n $MEDICAL_CHAINCODE \
  -c "{
    \"function\":\"CreateMedicalRecord\",
    \"Args\":[\"$RECORD_ID\",\"$PATIENT_ID\",\"a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890\"]
  }" \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Step 2: Research org requests access
echo "Step 2: Research organization requesting access..."
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/research.medical-network.com/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:10051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt

# Note: In a real application, capture the requestID from the response
peer chaincode invoke \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"RequestAccess\",
    \"Args\":[\"$RECORD_ID\",\"$DURATION\"]
  }" \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt

# For this example, assume requestID (in practice, parse from response)
REQUEST_ID="AUTH-$RECORD_ID-$(date +%s)000000000"

# Step 3: Hospital approves
echo "Step 3: Hospital approving authorization..."
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode invoke \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"ApproveByHospital\",
    \"Args\":[\"$REQUEST_ID\"]
  }" \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Step 4: Patient approves
echo "Step 4: Patient approving authorization..."
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/patient789@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode invoke \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"ApproveByPatient\",
    \"Args\":[\"$REQUEST_ID\"]
  }" \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Step 5: Query authorization (as regulator)
echo "Step 5: Querying authorization status (as regulator)..."
export CORE_PEER_LOCALMSPID=RegulatorMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/users/Admin@regulator.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt

peer chaincode query \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"GetAuthorization\",
    \"Args\":[\"$REQUEST_ID\"]
  }" \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt

echo "Workflow completed!"
```

## Listening to Events

### Using Fabric SDK (Node.js)

```javascript
const { Gateway, Wallets } = require('fabric-network');
const fs = require('fs');
const path = require('path');

async function listenToAuthorizationEvents() {
  // Load connection profile
  const ccpPath = path.resolve(__dirname, 'connection-profile.json');
  const ccp = JSON.parse(fs.readFileSync(ccpPath, 'utf8'));

  // Create wallet
  const walletPath = path.join(process.cwd(), 'wallet');
  const wallet = await Wallets.newFileSystemWallet(walletPath);

  // Check if user exists
  const identity = await wallet.get('appUser');
  if (!identity) {
    console.log('User does not exist in wallet');
    return;
  }

  // Create gateway
  const gateway = new Gateway();
  await gateway.connect(ccp, {
    wallet,
    identity: 'appUser',
    discovery: { enabled: true, asLocalhost: true }
  });

  // Get network and contract
  const network = await gateway.getNetwork('medical-main-channel');
  const contract = network.getContract('authorization-consent');

  // Listen to events
  const listener = async (event) => {
    const eventData = JSON.parse(event.payload.toString());
    console.log('=== Authorization Event Received ===');
    console.log('Event Name:', event.eventName);
    console.log('Request ID:', eventData.requestID);
    console.log('Record ID:', eventData.recordID);
    console.log('Action:', eventData.action);
    console.log('Status:', eventData.status);
    console.log('Invoker MSP:', eventData.invokerMSP);
    console.log('Client ID:', eventData.clientID);
    console.log('Timestamp:', eventData.timestamp);
    console.log('Patient ID:', eventData.patientID);
    console.log('Hospital ID:', eventData.hospitalID);
    console.log('Requester Org:', eventData.requesterOrg);
    console.log('===================================');
  };

  // Register listener for all contract events
  await contract.addContractListener(listener);

  console.log('Listening for authorization events...');
  
  // Keep the process running
  process.on('SIGINT', async () => {
    await gateway.disconnect();
    process.exit(0);
  });
}

listenToAuthorizationEvents().catch(console.error);
```

## Error Handling

### Common Errors and Solutions

1. **"access denied: RequestAccess can only be invoked by ResearchOrgMSP"**
   - **Cause**: Invoker is not ResearchOrgMSP
   - **Solution**: Use ResearchOrgMSP identity

2. **"access denied: only the owning hospital can approve"**
   - **Cause**: Invoker hospital is not the owner of the medical record
   - **Solution**: Use the correct hospital MSP that owns the record

3. **"access denied: only the patient can approve"**
   - **Cause**: Client ID doesn't match patientID in authorization request
   - **Solution**: Use the patient's client identity (X.509 certificate with matching CN)

4. **"medical record does not exist"**
   - **Cause**: RecordID doesn't exist in medical-records chaincode
   - **Solution**: Create the medical record first using medical-records chaincode

5. **"duration must be greater than 0 hours"**
   - **Cause**: Duration parameter is 0 or negative
   - **Solution**: Provide a positive duration value

6. **"authorization request has expired"**
   - **Cause**: Authorization request's expiresAt timestamp has passed
   - **Solution**: Create a new authorization request with appropriate duration

7. **"invalid state transition"**
   - **Cause**: Trying to perform an invalid state transition (e.g., approving already granted authorization)
   - **Solution**: Check current status and follow valid state transitions

## Best Practices

1. **Request ID Management**: Always capture and store the requestID returned by RequestAccess
2. **Patient Identity**: Ensure patient client identities are created with CN matching patientID in medical records
3. **Duration Planning**: Set appropriate duration based on research needs (consider privacy and compliance)
4. **Error Handling**: Always check transaction status and handle errors appropriately
5. **Event Monitoring**: Set up event listeners for audit and compliance tracking
6. **Access Control**: Always use the correct MSP identity for each operation
7. **State Tracking**: Track authorization status to know when both parties have approved
8. **Revocation**: Implement revocation workflows for when patients or hospitals need to revoke access
