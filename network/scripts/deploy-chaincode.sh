#!/usr/bin/env bash

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${PROJECT_ROOT}"

CHANNEL_NAME="medical-main-channel"
CC_NAME="medicalcc"
CC_LANG="${CC_LANG:-golang}"
CC_VERSION="1.0"
CC_SEQUENCE="1"
CC_LABEL="medicalcc_1.0"
CC_ENDORSEMENT_POLICY="OR('HospitalMSP1.peer')"
CC_PACKAGE_HOST="${PROJECT_ROOT}/network/channel-artifacts/medicalcc.tar.gz"
CC_PACKAGE_IN_PEER="/etc/hyperledger/configtx/medicalcc.tar.gz"
CC_PATH_TOOLS="/workspace/chaincode/medical-records"
CC_PATH_HOST="${PROJECT_ROOT}/chaincode/medical-records"

ORDERER_ADDRESS="orderer0.medical-network.com:7050"
ORDERER_TLS_CA="/etc/hyperledger/orderer-tls/ca.crt"

PEERS=(
  "peer0.hospital1.medical-network.com|HospitalMSP1|/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp"
  "peer0.hospital2.medical-network.com|HospitalMSP2|/etc/hyperledger/fabric/users/Admin@hospital2.medical-network.com/msp"
  "peer0.research.medical-network.com|ResearchOrgMSP|/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp"
  "peer0.regulator.medical-network.com|RegulatorMSP|/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp"
)

peer_exec() {
  local container="$1"
  local msp="$2"
  local admin_msp="$3"
  local cmd="$4"

  docker exec \
    -e CORE_PEER_LOCALMSPID="${msp}" \
    -e CORE_PEER_TLS_ENABLED=true \
    -e CORE_PEER_ADDRESS="${container}:7051" \
    -e CORE_PEER_MSPCONFIGPATH="${admin_msp}" \
    -e CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt \
    -e FABRIC_CFG_PATH=/etc/hyperledger/peercfg \
    "${container}" bash -c "${cmd}"
}

for row in "${PEERS[@]}"; do
  IFS='|' read -r container _ _ <<< "${row}"
  if ! docker ps --format '{{.Names}}' | grep -qx "${container}" >/dev/null 2>&1; then
    echo "Required peer is not running: ${container}"
    exit 1
  fi
done

# Keep user-requested default (golang), but gracefully fall back to node if the
# current chaincode path has no Go sources. This preserves existing project code.
if [[ "${CC_LANG}" == "golang" ]]; then
  shopt -s nullglob
  go_files=("${CC_PATH_HOST}"/*.go)
  shopt -u nullglob
  if [[ "${#go_files[@]}" -eq 0 ]]; then
    echo "No .go files found under ${CC_PATH_HOST}; using node packaging for existing chaincode."
    CC_LANG="node"
  fi
fi

echo "1) Packaging chaincode (${CC_LANG})..."
rm -f "${CC_PACKAGE_HOST}"
docker run --rm \
  -v "${PROJECT_ROOT}:/workspace" \
  -w /workspace \
  hyperledger/fabric-tools:2.5 \
  peer lifecycle chaincode package "network/channel-artifacts/medicalcc.tar.gz" \
    --path "${CC_PATH_TOOLS}" \
    --lang "${CC_LANG}" \
    --label "${CC_LABEL}"

echo "2) Installing chaincode on all peers..."
for row in "${PEERS[@]}"; do
  IFS='|' read -r container msp admin_msp <<< "${row}"
  if peer_exec "${container}" "${msp}" "${admin_msp}" "peer lifecycle chaincode queryinstalled | grep '${CC_LABEL}'" >/dev/null 2>&1; then
    echo " - ${container}: already installed"
  else
    echo " - ${container}: installing"
    peer_exec "${container}" "${msp}" "${admin_msp}" "peer lifecycle chaincode install ${CC_PACKAGE_IN_PEER}"
  fi
done

echo "3) Resolving package ID..."
PACKAGE_ID="$(peer_exec "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" \
  "peer lifecycle chaincode queryinstalled | sed -n 's/^Package ID: \\([^,]*\\), Label: ${CC_LABEL}$/\\1/p' | head -n 1")"

if [[ -z "${PACKAGE_ID}" ]]; then
  echo "Failed to resolve package ID for ${CC_LABEL}"
  exit 1
fi
echo " - PACKAGE_ID=${PACKAGE_ID}"

echo "4) Approving chaincode definition for all organizations..."
for row in "${PEERS[@]}"; do
  IFS='|' read -r container msp admin_msp <<< "${row}"
  echo " - approving for ${msp}"
  peer_exec "${container}" "${msp}" "${admin_msp}" \
    "peer lifecycle chaincode approveformyorg \
      -o ${ORDERER_ADDRESS} \
      --ordererTLSHostnameOverride orderer0.medical-network.com \
      --channelID ${CHANNEL_NAME} \
      --name ${CC_NAME} \
      --version ${CC_VERSION} \
      --package-id ${PACKAGE_ID} \
      --sequence ${CC_SEQUENCE} \
      --signature-policy \"${CC_ENDORSEMENT_POLICY}\" \
      --tls --cafile ${ORDERER_TLS_CA} \
      --waitForEvent=false"
done

echo "4b) Waiting for commit readiness..."
readiness_cmd="peer lifecycle chaincode checkcommitreadiness \
  --channelID ${CHANNEL_NAME} \
  --name ${CC_NAME} \
  --version ${CC_VERSION} \
  --sequence ${CC_SEQUENCE} \
  --signature-policy \"${CC_ENDORSEMENT_POLICY}\" \
  --output json"

for attempt in {1..15}; do
  readiness="$(peer_exec "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" "${readiness_cmd}")"
  if echo "${readiness}" | grep -Eq '"HospitalMSP1":[[:space:]]*true' \
    && echo "${readiness}" | grep -Eq '"HospitalMSP2":[[:space:]]*true' \
    && echo "${readiness}" | grep -Eq '"ResearchOrgMSP":[[:space:]]*true' \
    && echo "${readiness}" | grep -Eq '"RegulatorMSP":[[:space:]]*true'; then
    echo " - all organizations are ready to commit"
    break
  fi
  if [[ "${attempt}" -eq 15 ]]; then
    echo "Commit readiness did not converge:"
    echo "${readiness}"
    exit 1
  fi
  sleep 2
done

echo "5) Committing chaincode definition..."
peer_exec "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" \
  "peer lifecycle chaincode commit \
    -o ${ORDERER_ADDRESS} \
    --ordererTLSHostnameOverride orderer0.medical-network.com \
    --channelID ${CHANNEL_NAME} \
    --name ${CC_NAME} \
    --version ${CC_VERSION} \
    --sequence ${CC_SEQUENCE} \
    --signature-policy \"OR('HospitalMSP1.peer')\" \
    --tls --cafile ${ORDERER_TLS_CA} \
    --peerAddresses peer0.hospital1.medical-network.com:7051 \
    --tlsRootCertFiles /etc/hyperledger/peer-tls/hospital1/ca.crt \
    --waitForEvent=false"

echo "6) Verifying committed chaincode..."
peer_exec "peer0.regulator.medical-network.com" "RegulatorMSP" "/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp" \
  "peer lifecycle chaincode querycommitted -C ${CHANNEL_NAME}"

echo "Chaincode lifecycle completed for ${CC_NAME} on ${CHANNEL_NAME}."
