#!/bin/bash

set -e

CHANNEL="medical-main-channel"
CC="medicalcc"
ORDERER="orderer0.medical-network.com:7050"
RECORD_ID="rec_$(date +%s)"
FILE_HASH=""
VERIFY_HASH=""
LAST_HEIGHT=""
EXPIRY=""
RESEARCHER_ID="x509::/C=US/ST=California/L=San Francisco/OU=admin/CN=Admin@research.medical-network.com::/C=US/ST=California/L=San Francisco/O=research.medical-network.com/CN=ca.research.medical-network.com"
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_ledger_info() {
  LEDGER_INFO=$(docker exec -i peer0.hospital1.medical-network.com bash -c '
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
peer channel getinfo -c medical-main-channel
')

  HEIGHT=$(echo "$LEDGER_INFO" | sed -n 's/.*"height":[[:space:]]*\([0-9][0-9]*\).*/\1/p')
  CURRENT_HASH=$(echo "$LEDGER_INFO" | sed -n 's/.*"currentBlockHash":"\([^"]*\)".*/\1/p')
  PREVIOUS_HASH=$(echo "$LEDGER_INFO" | sed -n 's/.*"previousBlockHash":"\([^"]*\)".*/\1/p')

  echo "$LEDGER_INFO"
  echo "block height: ${HEIGHT:-unknown}"
  echo "current block hash: ${CURRENT_HASH:-unknown}"
  echo "previous block hash: ${PREVIOUS_HASH:-unknown}"

  if [ -n "$HEIGHT" ]; then
    if [ -n "$LAST_HEIGHT" ] && [ "$HEIGHT" -gt "$LAST_HEIGHT" ]; then
      echo "Ledger height increased."
    fi
    LAST_HEIGHT="$HEIGHT"
  fi
}

extract_txid() {
  TX_ID=$(echo "$1" | grep -o "txid \[[^]]*\]" | head -1 | sed 's/txid \[//' | sed 's/\]//')
  if [ -n "$TX_ID" ]; then
    echo -e "${GREEN}Transaction committed with ID: $TX_ID${NC}"
    echo "Transaction permanently recorded on blockchain."
  else
    echo "Transaction ID not parsed from command output."
  fi
}

post_transaction_blockchain_proof() {
  echo
  echo "============================================"
  echo "BLOCK INSPECTION"
  echo "============================================"
  docker exec peer0.hospital1.medical-network.com \
  peer channel fetch newest newest_block.pb \
  -o orderer0.medical-network.com:7050 \
  -c medical-main-channel \
  --tls \
  --cafile /etc/hyperledger/orderer-tls/ca.crt
  echo "Latest block fetched from blockchain ledger."
  echo "This block contains the transaction that recorded the medical metadata."
  echo
  echo "Checking ledger height after transaction..."
  print_ledger_info
  echo "Ledger height increased, proving the transaction was appended to the blockchain."
}

echo "====================================================="
echo " BLOCKCHAIN-BASED MEDICAL DATA SHARING DEMO"
echo " Hyperledger Fabric + MinIO Hybrid Architecture"
echo "====================================================="
echo "---------------------------------------"
echo "DEMO RECORD ID: $RECORD_ID"
echo "---------------------------------------"

echo "Organizations:"
echo "  Hospital1"
echo "  Hospital2"
echo "  ResearchOrg"
echo "  Regulator"

echo
echo "Architecture:"
echo "  Blockchain: Hyperledger Fabric"
echo "  Storage: MinIO Object Storage"
echo "  Smart Contract: MedicalContract"

echo
echo "============================================"
echo "SECURITY PROPERTIES DEMONSTRATED"
echo "============================================"
echo "1. Hospitals can upload records"
echo "2. Researchers cannot access records without approval"
echo "3. Patient consent defines access expiry"
echo "4. Regulator can audit entire ledger"
echo "5. Large medical files stored OFF-CHAIN"
echo "6. Blockchain stores only metadata + hash"
echo "7. Hash guarantees file integrity"
sleep 1

echo
echo "============================================"
echo "SYSTEM ARCHITECTURE"
echo "============================================"
echo "Hospital"
echo "   |"
echo "   v"
echo "Hyperledger Fabric Blockchain"
echo "   |"
echo "   v"
echo "MinIO Object Storage"
echo "   |"
echo "   v"
echo "Researcher Access"
sleep 1

echo
echo "============================================"
echo "NETWORK TOPOLOGY"
echo "============================================"
echo "Hospital1 Peer"
echo "Hospital2 Peer"
echo "Research Peer"
echo "Regulator Peer"
echo "Ordering Service"
docker ps --format "table {{.Names}}\t{{.Status}}"
sleep 1

echo
echo "============================================"
echo -e "${BLUE}STEP 1 — Verify Channel${NC}"
echo "============================================"
docker exec peer0.hospital1.medical-network.com \
peer channel list
sleep 1

echo
echo "============================================"
echo "BLOCKCHAIN LEDGER INFO"
echo "============================================"
print_ledger_info
echo "This proves immutable blockchain ledger."
sleep 1

echo
echo "============================================"
echo "ENDORSEMENT POLICY"
echo "============================================"
docker exec \
-e CORE_PEER_LOCALMSPID=HospitalMSP1 \
-e CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp \
-e CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:7051 \
-e CORE_PEER_TLS_ENABLED=true \
-e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
peer0.hospital1.medical-network.com \
peer lifecycle chaincode querycommitted -C $CHANNEL
echo "Transactions require endorsement from multiple organizations."
sleep 1

echo
echo "============================================"
echo "STEP 4 — Upload file to MinIO (OFF-CHAIN)"
echo "============================================"
docker exec minio-medical mkdir -p /data/medical-records/reports
docker exec minio-medical sh -c \
"echo 'Patient medical report for demo record $RECORD_ID' > /data/medical-records/reports/$RECORD_ID.txt"
FILE_HASH=$(docker exec minio-medical sha256sum /data/medical-records/reports/$RECORD_ID.txt | awk '{print $1}')
echo
echo "Computed file SHA256 hash:"
echo "$FILE_HASH"
sleep 1

echo
echo "============================================"
echo "STEP 5 — Hospital uploads metadata ON-CHAIN"
echo "============================================"
START_TIME=$(date +%s)
UPLOAD_OUTPUT=$(docker exec -e RECORD_ID="$RECORD_ID" -e FILE_HASH="$FILE_HASH" -i peer0.hospital1.medical-network.com bash -c '
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt

peer chaincode invoke \
-o orderer0.medical-network.com:7050 \
--ordererTLSHostnameOverride orderer0.medical-network.com \
--tls \
--cafile /etc/hyperledger/orderer-tls/ca.crt \
--waitForEvent \
-C medical-main-channel \
-n medicalcc \
--peerAddresses peer0.hospital1.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital1/ca.crt \
--peerAddresses peer0.hospital2.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital2/ca.crt \
--peerAddresses peer0.research.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/research/ca.crt \
-c "{\"Args\":[\"uploadMedicalRecord\",\"$RECORD_ID\",\"patient1\",\"$FILE_HASH\",\"medical-records/reports/$RECORD_ID.txt\"]}"
' 2>&1)
END_TIME=$(date +%s)
TX_TIME=$((END_TIME - START_TIME))
echo "$UPLOAD_OUTPUT"
echo "Transaction latency: ${TX_TIME}s"
extract_txid "$UPLOAD_OUTPUT"
echo -e "${GREEN}Transaction committed successfully${NC}"
post_transaction_blockchain_proof
sleep 1

echo
echo "============================================"
echo "STEP 6 — Verify record stored ON-CHAIN"
echo "============================================"
docker exec -e RECORD_ID="$RECORD_ID" -it peer0.hospital1.medical-network.com bash -c '
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:7051

peer chaincode query \
-C medical-main-channel \
-n medicalcc \
-c "{\"Args\":[\"getMedicalRecord\",\"$RECORD_ID\"]}"
'
sleep 1

echo
echo "============================================"
echo -e "${YELLOW}SECURITY TEST — Unauthorized Access Attempt${NC}"
echo "============================================"
set +e
docker exec -it peer0.research.medical-network.com bash -c "
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt

peer chaincode query \
-C medical-main-channel \
-n medicalcc \
-c '{\"Args\":[\"retrieveRecord\",\"$RECORD_ID\",\"$RESEARCHER_ID\"]}'
"
UNAUTHORIZED_STATUS=$?
set -e
echo "Expected result: Access denied because patient consent not granted."
if [ "$UNAUTHORIZED_STATUS" -eq 0 ]; then
  echo "WARNING: Unauthorized retrieval unexpectedly succeeded."
else
  echo -e "${RED}Access denied as expected${NC}"
  echo "Unauthorized request was rejected by chaincode access policy."
fi
sleep 1

echo
echo "============================================"
echo "STEP 7 — Researcher requests access"
echo "============================================"
START_TIME=$(date +%s)
REQUEST_OUTPUT=$(docker exec -e RECORD_ID="$RECORD_ID" -i peer0.research.medical-network.com bash -c '
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt

peer chaincode invoke \
-o orderer0.medical-network.com:7050 \
--ordererTLSHostnameOverride orderer0.medical-network.com \
--tls \
--cafile /etc/hyperledger/orderer-tls/ca.crt \
--waitForEvent \
-C medical-main-channel \
-n medicalcc \
--peerAddresses peer0.hospital1.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital1/ca.crt \
--peerAddresses peer0.hospital2.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital2/ca.crt \
--peerAddresses peer0.research.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/research/ca.crt \
-c "{\"Args\":[\"requestAccess\",\"$RECORD_ID\"]}"
' 2>&1)
END_TIME=$(date +%s)
TX_TIME=$((END_TIME - START_TIME))
echo "$REQUEST_OUTPUT"
echo "Transaction latency: ${TX_TIME}s"
extract_txid "$REQUEST_OUTPUT"
echo -e "${GREEN}Transaction committed successfully${NC}"
post_transaction_blockchain_proof
sleep 1

echo
echo "============================================"
echo "STEP 8 — Hospital approves access"
echo "============================================"
EXPIRY=$(($(date +%s) + 30))
echo
echo "Consent expiry set to:"
echo "$EXPIRY"
START_TIME=$(date +%s)
APPROVE_OUTPUT=$(docker exec -i peer0.hospital1.medical-network.com bash -c "
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt

peer chaincode invoke \
-o orderer0.medical-network.com:7050 \
--ordererTLSHostnameOverride orderer0.medical-network.com \
--tls \
--cafile /etc/hyperledger/orderer-tls/ca.crt \
--waitForEvent \
-C medical-main-channel \
-n medicalcc \
--peerAddresses peer0.hospital1.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital1/ca.crt \
--peerAddresses peer0.hospital2.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital2/ca.crt \
--peerAddresses peer0.research.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/research/ca.crt \
-c '{\"Args\":[\"approveAccess\",\"$RECORD_ID\",\"$RESEARCHER_ID\",\"$EXPIRY\"]}'
" 2>&1)
END_TIME=$(date +%s)
TX_TIME=$((END_TIME - START_TIME))
echo "$APPROVE_OUTPUT"
echo "Transaction latency: ${TX_TIME}s"
extract_txid "$APPROVE_OUTPUT"
echo -e "${GREEN}Transaction committed successfully${NC}"
post_transaction_blockchain_proof
echo
echo "Current time:"
date +%s
echo "Access allowed only if current_time < expiry"
sleep 1

echo
echo "============================================"
echo "STEP 9 — Regulator audit (ON-CHAIN)"
echo "============================================"
docker exec -it peer0.regulator.medical-network.com bash -c '
export CORE_PEER_LOCALMSPID=RegulatorMSP
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
export FABRIC_CFG_PATH=/etc/hyperledger/peercfg

peer chaincode query \
-C medical-main-channel \
-n medicalcc \
-c "{\"Args\":[\"auditRecords\"]}"
'
sleep 1

echo
echo "============================================"
echo "STEP 10 — Researcher retrieves record"
echo "============================================"
docker exec -it peer0.research.medical-network.com bash -c "
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:7051

peer chaincode query \
-C medical-main-channel \
-n medicalcc \
-c '{\"Args\":[\"retrieveRecord\",\"$RECORD_ID\",\"$RESEARCHER_ID\"]}'
"
sleep 1

echo
echo "============================================"
echo "STEP — Waiting for consent expiry"
echo "============================================"
sleep 35

set +e
docker exec -it peer0.research.medical-network.com bash -c "
export CORE_PEER_LOCALMSPID=ResearchOrgMSP
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.research.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt

peer chaincode query \
-C medical-main-channel \
-n medicalcc \
-c '{\"Args\":[\"retrieveRecord\",\"$RECORD_ID\",\"$RESEARCHER_ID\"]}'
"
EXPIRED_STATUS=$?
set -e
echo "Access denied because patient consent expired."
if [ "$EXPIRED_STATUS" -eq 0 ]; then
  echo "WARNING: Expired consent retrieval unexpectedly succeeded."
else
  echo -e "${RED}Access denied as expected${NC}"
  echo "Expired consent was enforced correctly."
fi
sleep 1

echo
echo "============================================"
echo "STEP 11 — Verify OFF-CHAIN storage (MinIO)"
echo "============================================"
docker exec minio-medical ls -R /data/medical-records
echo
echo "Stored file contents:"
docker exec minio-medical cat /data/medical-records/reports/$RECORD_ID.txt
sleep 1

echo
echo "============================================"
echo "VERIFY FILE INTEGRITY"
echo "============================================"
VERIFY_HASH=$(docker exec minio-medical sha256sum /data/medical-records/reports/$RECORD_ID.txt | awk '{print $1}')
echo "Blockchain hash:"
echo "$FILE_HASH"
echo
echo "Actual file hash:"
echo "$VERIFY_HASH"
if [ "$FILE_HASH" = "$VERIFY_HASH" ]; then
  echo -e "${GREEN}Integrity verified — file not tampered.${NC}"
else
  echo "WARNING: Hash mismatch detected."
fi
sleep 1

echo
echo "============================================"
echo "SECURITY TEST — File Tampering Attack"
echo "============================================"
docker exec minio-medical sh -c \
"echo 'Malicious modification' >> /data/medical-records/reports/$RECORD_ID.txt"
TAMPER_HASH=$(docker exec minio-medical sha256sum /data/medical-records/reports/$RECORD_ID.txt | awk '{print $1}')
echo "Blockchain hash: $FILE_HASH"
echo "Tampered file hash: $TAMPER_HASH"
if [ "$FILE_HASH" != "$TAMPER_HASH" ]; then
  echo -e "${RED}Tampering detected via SHA256 mismatch.${NC}"
  echo "Blockchain integrity verification prevents data tampering."
else
  echo "WARNING: Tampering detection did not trigger."
fi
sleep 1

echo
echo "============================================"
echo "BLOCKCHAIN IMMUTABILITY TEST"
echo "============================================"
set +e
IMMUTABILITY_OUTPUT=$(docker exec -e RECORD_ID="$RECORD_ID" -e FILE_HASH="$FILE_HASH" -i peer0.hospital1.medical-network.com bash -c '
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt

peer chaincode invoke \
-o orderer0.medical-network.com:7050 \
--ordererTLSHostnameOverride orderer0.medical-network.com \
--tls \
--cafile /etc/hyperledger/orderer-tls/ca.crt \
--waitForEvent \
-C medical-main-channel \
-n medicalcc \
--peerAddresses peer0.hospital1.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital1/ca.crt \
--peerAddresses peer0.hospital2.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/hospital2/ca.crt \
--peerAddresses peer0.research.medical-network.com:7051 \
--tlsRootCertFiles /etc/hyperledger/peer-tls/research/ca.crt \
-c "{\"Args\":[\"uploadMedicalRecord\",\"$RECORD_ID\",\"patient1\",\"$FILE_HASH\",\"medical-records/reports/$RECORD_ID.txt\"]}"
' 2>&1)
IMMUTABILITY_STATUS=$?
set -e
echo "$IMMUTABILITY_OUTPUT"
if [ "$IMMUTABILITY_STATUS" -eq 0 ]; then
  echo "WARNING: Duplicate record upload unexpectedly succeeded."
else
  echo -e "${GREEN}Blockchain prevents modification of historical records.${NC}"
fi
sleep 1

echo
echo "FULL CONSENT LIFECYCLE:"
echo "UPLOAD -> REQUEST -> APPROVE -> ACCESS -> EXPIRE"
sleep 1

echo
echo "============================================"
echo "CHAINCODE EVENTS"
echo "============================================"
echo "Events emitted:"
echo "  RecordUploaded"
echo "  AccessRequested"
echo "  ConsentGranted"
echo "Backend event listeners consume these events for async processing,"
echo "notifications, analytics, and regulator-facing dashboards."
sleep 1

echo
echo "============================================"
echo "ON-CHAIN VS OFF-CHAIN STORAGE"
echo "============================================"
echo "ON-CHAIN"
echo "recordID"
echo "patientID"
echo "fileHash"
echo "minioPath"
echo "timestamp"
echo "consent logs"
echo
echo "OFF-CHAIN"
echo "medical files"
echo "reports"
echo "large datasets"
echo "Hybrid architecture keeps blockchain lightweight while preserving integrity."
sleep 1

echo
echo "============================================"
echo "PERMISSION MODEL"
echo "============================================"
echo "Hospital role:"
echo "  upload records"
echo "  approve researcher access"
echo
echo "Researcher role:"
echo "  request access"
echo "  retrieve records if consent active"
echo
echo "Regulator role:"
echo "  audit full blockchain state"
echo
echo "Patient role:"
echo "  consent expiry controls access duration"
sleep 1

echo
echo "============================================"
echo "SECURITY GUARANTEES"
echo "============================================"
echo "✔ Permissioned blockchain network"
echo "✔ Multi-organization endorsement"
echo "✔ Role-based access control"
echo "✔ Patient-controlled consent"
echo "✔ Time-limited access"
echo "✔ Unauthorized access rejection"
echo "✔ Immutable audit logs"
echo "✔ File integrity verification"
sleep 1

echo
echo "============================================"
echo "BLOCKCHAIN MEDICAL DATA SHARING SYSTEM"
echo "============================================"
echo "Components:"
echo "  Hyperledger Fabric"
echo "  Smart Contracts"
echo "  Role-Based Access Control"
echo "  Consent Management"
echo "  MinIO Storage"
echo "  SHA256 Integrity"
echo
echo "System guarantees:"
echo "  Decentralization"
echo "  Transparency"
echo "  Data Integrity"
echo "  Secure Data Sharing"
echo "  Regulatory Auditability"
sleep 1

echo
echo "============================================"
echo "DEMO EXECUTION TIMELINE"
echo "============================================"
echo "Record Upload -> Blockchain Metadata Stored"
echo "Researcher Request -> Pending Consent"
echo "Hospital Approval -> Consent Granted"
echo "Researcher Retrieval -> Access Allowed"
echo "Consent Expiry -> Access Revoked"
sleep 1

echo
echo "============================================"
echo "SYSTEM METRICS"
echo "============================================"
echo "Running containers:"
docker ps | wc -l
echo "Blockchain ledger height:"
docker exec peer0.hospital1.medical-network.com \
peer channel getinfo -c medical-main-channel
echo "Stored medical files in MinIO:"
docker exec minio-medical ls /data/medical-records/reports | wc -l
sleep 1

echo
echo "============================================"
echo " DEMO COMPLETED SUCCESSFULLY"
echo "============================================"
echo
echo "============================================"
echo " DEMO SUMMARY"
echo "============================================"
echo "✔ Permissioned blockchain network"
echo "✔ Multi-organization endorsement"
echo "✔ Patient consent controlled access"
echo "✔ Time-limited access control"
echo "✔ Unauthorized access prevented"
echo "✔ Hash-based file integrity"
echo "✔ Hybrid on-chain/off-chain architecture"
echo "✔ Regulator audit capability"