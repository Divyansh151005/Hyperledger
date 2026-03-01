# Medical Records Chaincode - Usage Examples

This document provides practical examples for using the medical-records chaincode.

## Prerequisites

- Network is running and channel is created
- Chaincode is installed and committed
- Client certificates are available for each organization

## Example 1: Hospital Creates a Medical Record

### Step 1: Compute Hash of Off-Chain Medical Record

First, compute the SHA-256 hash of the medical record data (stored off-chain):

```bash
# Example: Hash a medical record file
sha256sum medical_record.json
# Output: a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890
```

Or using a simple script:
```python
import hashlib
import json

# Load medical record data
with open('medical_record.json', 'r') as f:
    record_data = json.load(f)

# Compute hash
record_json = json.dumps(record_data, sort_keys=True)
hash_value = hashlib.sha256(record_json.encode()).hexdigest()
print(f"Record Hash: {hash_value}")
```

### Step 2: Invoke CreateMedicalRecord

```bash
# Set environment variables for HospitalMSP1
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Invoke CreateMedicalRecord
peer chaincode invoke \
  -C medical-main-channel \
  -n medical-records \
  -c '{
    "function":"CreateMedicalRecord",
    "Args":[
      "REC-2024-001",
      "PAT-12345",
      "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890"
    ]
  }' \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
2024-01-15 10:30:00.123 UTC [chaincodeCmd] chaincodeInvokeOrQuery -> INFO 001 Chaincode invoke successful. result: status:200
```

## Example 2: Regulator Queries Medical Record

```bash
# Set environment variables for RegulatorMSP
export CORE_PEER_LOCALMSPID=RegulatorMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/users/Admin@regulator.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt

# Query GetMedicalRecord
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{
    "function":"GetMedicalRecord",
    "Args":["REC-2024-001"]
  }' \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt
```

**Expected Output**:
```json
{
  "recordID": "REC-2024-001",
  "patientID": "PAT-12345",
  "hospitalID": "HospitalMSP1",
  "recordHash": "a1b2c3d4e5f6789012345678901234567890123456789012345678901234567890",
  "timestamp": "2024-01-15T10:30:00Z",
  "status": "ACTIVE"
}
```

## Example 3: Hospital Verifies Record Integrity

```bash
# Set environment variables for HospitalMSP1 (owner of the record)
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Compute hash of current medical record data
CURRENT_HASH=$(sha256sum medical_record.json | cut -d' ' -f1)

# Query VerifyMedicalRecordHash
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c "{
    \"function\":\"VerifyMedicalRecordHash\",
    \"Args\":[\"REC-2024-001\",\"$CURRENT_HASH\"]
  }" \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
true
```

## Example 4: Unauthorized Access Attempt

### Attempt by ResearchOrgMSP to Create Record

```bash
# Set environment variables for ResearchOrgMSP
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/research.medical-network.com/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:10051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt

# Attempt to invoke CreateMedicalRecord (will fail)
peer chaincode invoke \
  -C medical-main-channel \
  -n medical-records \
  -c '{
    "function":"CreateMedicalRecord",
    "Args":[
      "REC-2024-002",
      "PAT-12346",
      "b2c3d4e5f6789012345678901234567890123456789012345678901234567890ab"
    ]
  }' \
  --peerAddresses peer0.research.medical-network.com:10051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/research.medical-network.com/peers/peer0.research.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
Error: access denied: CreateMedicalRecord can only be invoked by HospitalMSP organizations, got ResearchOrgMSP
```

### Attempt by HospitalMSP2 to Access HospitalMSP1's Record

```bash
# Set environment variables for HospitalMSP2
export CORE_PEER_LOCALMSPID=HospitalMSP2
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital2.medical-network.com/users/Admin@hospital2.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital2.medical-network.com:9051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital2.medical-network.com/peers/peer0.hospital2.medical-network.com/tls/ca.crt

# Attempt to query HospitalMSP1's record (will fail)
peer chaincode query \
  -C medical-main-channel \
  -n medical-records \
  -c '{
    "function":"GetMedicalRecord",
    "Args":["REC-2024-001"]
  }' \
  --peerAddresses peer0.hospital2.medical-network.com:9051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital2.medical-network.com/peers/peer0.hospital2.medical-network.com/tls/ca.crt
```

**Expected Output**:
```
Error: access denied: GetMedicalRecord can only be invoked by RegulatorMSP or the owning HospitalMSP (record owned by HospitalMSP1, invoker is HospitalMSP2)
```

## Example 5: Complete Workflow

### Workflow: Create, Query, and Verify

```bash
#!/bin/bash

# Configuration
CHANNEL="medical-main-channel"
CHAINCODE="medical-records"
RECORD_ID="REC-2024-003"
PATIENT_ID="PAT-12347"
MEDICAL_RECORD_FILE="medical_record_003.json"

# Step 1: Compute hash
echo "Step 1: Computing hash of medical record..."
RECORD_HASH=$(sha256sum $MEDICAL_RECORD_FILE | cut -d' ' -f1)
echo "Record Hash: $RECORD_HASH"

# Step 2: Create medical record (as HospitalMSP1)
echo "Step 2: Creating medical record..."
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode invoke \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"CreateMedicalRecord\",
    \"Args\":[\"$RECORD_ID\",\"$PATIENT_ID\",\"$RECORD_HASH\"]
  }" \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Step 3: Query record (as HospitalMSP1 - owner)
echo "Step 3: Querying medical record (as owner)..."
peer chaincode query \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"GetMedicalRecord\",
    \"Args\":[\"$RECORD_ID\"]
  }" \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

# Step 4: Query record (as RegulatorMSP - oversight)
echo "Step 4: Querying medical record (as regulator)..."
export CORE_PEER_LOCALMSPID=RegulatorMSP
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/users/Admin@regulator.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt

peer chaincode query \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"GetMedicalRecord\",
    \"Args\":[\"$RECORD_ID\"]
  }" \
  --peerAddresses peer0.regulator.medical-network.com:7051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt

# Step 5: Verify hash integrity
echo "Step 5: Verifying record hash integrity..."
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=/path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer chaincode query \
  -C $CHANNEL \
  -n $CHAINCODE \
  -c "{
    \"function\":\"VerifyMedicalRecordHash\",
    \"Args\":[\"$RECORD_ID\",\"$RECORD_HASH\"]
  }" \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

echo "Workflow completed!"
```

## Listening to Events

### Using Fabric SDK (Node.js)

```javascript
const { Gateway, Wallets } = require('fabric-network');
const fs = require('fs');
const path = require('path');

async function listenToEvents() {
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
  const contract = network.getContract('medical-records');

  // Listen to events
  const listener = async (event) => {
    const eventData = JSON.parse(event.payload.toString());
    console.log('Event received:', eventData);
    console.log('Event Name:', event.eventName);
    console.log('Record ID:', eventData.recordID);
    console.log('Action:', eventData.action);
    console.log('Invoker MSP:', eventData.invokerMSP);
    console.log('Timestamp:', eventData.timestamp);
  };

  // Register listener
  await contract.addContractListener(listener);

  console.log('Listening for events...');
}
```

### Using Peer Event Service

```bash
# Start peer event service
peer channel fetch 0 block.pb -c medical-main-channel \
  --peerAddresses peer0.hospital1.medical-network.com:8051 \
  --tlsRootCertFiles /path/to/tls/ca.crt

# Listen to events (requires event service setup)
# This is typically done through SDK or event service client
```

## Error Handling

### Common Errors and Solutions

1. **"access denied"**
   - **Cause**: Invoker MSP doesn't have permission
   - **Solution**: Use correct MSP identity (HospitalMSP for create, RegulatorMSP or owner for read)

2. **"medical record already exists"**
   - **Cause**: RecordID is already in use
   - **Solution**: Use a unique recordID

3. **"invalid recordHash format"**
   - **Cause**: Hash is not 64-character hex string
   - **Solution**: Ensure hash is SHA-256, hex-encoded, 64 characters

4. **"medical record does not exist"**
   - **Cause**: RecordID not found
   - **Solution**: Verify recordID is correct and record was created

## Best Practices

1. **Hash Computation**: Always compute hashes consistently (same algorithm, same encoding)
2. **Record IDs**: Use a consistent naming scheme (e.g., REC-YYYY-NNN)
3. **Patient IDs**: Use pseudonymized identifiers for privacy
4. **Error Handling**: Always check transaction status and handle errors appropriately
5. **Event Monitoring**: Set up event listeners for audit and compliance tracking
6. **Access Control**: Always use the correct MSP identity for each operation
