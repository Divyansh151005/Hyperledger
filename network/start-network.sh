#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
NETWORK_DIR="${PROJECT_ROOT}/network"
DOCKER_COMPOSE_FILE="${NETWORK_DIR}/docker/docker-compose-network.yaml"
CHANNEL_ARTIFACTS_DIR="${NETWORK_DIR}/channel-artifacts"
CRYPTO_DIR="${NETWORK_DIR}/crypto-config"
CONFIGTX_DIR="${NETWORK_DIR}/configtx"
CHANNEL_NAME="medical-main-channel"
SYSTEM_CHANNEL="system-channel"
ORDERER_ADDRESS="orderer0.medical-network.com:7050"
ORDERER_TLS_CA="/etc/hyperledger/orderer-tls/ca.crt"

if command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD="docker-compose"
else
  COMPOSE_CMD="docker compose"
fi

require_command() {
  local cmd="$1"
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "Missing required command: ${cmd}"
    exit 1
  fi
}

wait_for_container() {
  local container="$1"
  local tries=30
  local i
  for ((i=1; i<=tries; i++)); do
    if docker ps --format '{{.Names}}' | grep -qx "${container}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "Container did not start: ${container}"
  return 1
}

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

create_channel() {
  local attempt
  for attempt in {1..20}; do
    if peer_exec "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" \
      "peer channel create \
        -o ${ORDERER_ADDRESS} \
        --ordererTLSHostnameOverride orderer0.medical-network.com \
        -c ${CHANNEL_NAME} \
        -f /etc/hyperledger/configtx/${CHANNEL_NAME}.tx \
        --outputBlock /etc/hyperledger/configtx/${CHANNEL_NAME}.block \
        --tls --cafile ${ORDERER_TLS_CA}" >/dev/null 2>&1; then
      echo "Channel ${CHANNEL_NAME} created"
      return 0
    fi
    sleep 2
  done
  echo "Failed to create channel ${CHANNEL_NAME} after retries"
  return 1
}

join_channel() {
  local container="$1"
  local msp="$2"
  local admin_msp="$3"

  if peer_exec "${container}" "${msp}" "${admin_msp}" "peer channel list | grep -w ${CHANNEL_NAME}" >/dev/null 2>&1; then
    echo "${container} already joined ${CHANNEL_NAME}"
    return 0
  fi

  peer_exec "${container}" "${msp}" "${admin_msp}" "peer channel join -b /etc/hyperledger/configtx/${CHANNEL_NAME}.block"
}

update_anchor() {
  local container="$1"
  local msp="$2"
  local admin_msp="$3"
  local anchor_tx="$4"

  peer_exec "${container}" "${msp}" "${admin_msp}" \
    "peer channel update \
      -o ${ORDERER_ADDRESS} \
      --ordererTLSHostnameOverride orderer0.medical-network.com \
      -c ${CHANNEL_NAME} \
      -f /etc/hyperledger/configtx/${anchor_tx} \
      --tls --cafile ${ORDERER_TLS_CA}"
}

echo "Starting Hyperledger Fabric medical network..."
cd "${PROJECT_ROOT}"

require_command docker
require_command cryptogen
require_command configtxgen

if ! docker info >/dev/null 2>&1; then
  echo "Docker daemon is not running"
  exit 1
fi

echo "Stopping old containers and cleaning artifacts..."
${COMPOSE_CMD} -f "${DOCKER_COMPOSE_FILE}" down -v --remove-orphans || true
rm -rf "${PROJECT_ROOT}/data"
mkdir -p "${PROJECT_ROOT}/data"
mkdir -p "${CHANNEL_ARTIFACTS_DIR}"
rm -f "${CHANNEL_ARTIFACTS_DIR}"/*.block "${CHANNEL_ARTIFACTS_DIR}"/*.tx
rm -rf "${CRYPTO_DIR}/ordererOrganizations" "${CRYPTO_DIR}/peerOrganizations"

echo "Generating crypto material..."
cryptogen generate --config="${CRYPTO_DIR}/crypto-config.yaml" --output="${CRYPTO_DIR}"

echo "Generating channel artifacts..."
export FABRIC_CFG_PATH="${CONFIGTX_DIR}"
configtxgen -profile MedicalNetworkGenesis -channelID "${SYSTEM_CHANNEL}" -outputBlock "${CHANNEL_ARTIFACTS_DIR}/genesis.block"
configtxgen -profile MedicalMainChannel -channelID "${CHANNEL_NAME}" -outputCreateChannelTx "${CHANNEL_ARTIFACTS_DIR}/${CHANNEL_NAME}.tx"
configtxgen -profile MedicalMainChannel -channelID "${CHANNEL_NAME}" -asOrg HospitalMSP1 -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/HospitalMSP1anchors.tx"
configtxgen -profile MedicalMainChannel -channelID "${CHANNEL_NAME}" -asOrg HospitalMSP2 -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/HospitalMSP2anchors.tx"
configtxgen -profile MedicalMainChannel -channelID "${CHANNEL_NAME}" -asOrg ResearchOrgMSP -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/ResearchOrgMSPanchors.tx"
configtxgen -profile MedicalMainChannel -channelID "${CHANNEL_NAME}" -asOrg RegulatorMSP -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/RegulatorMSPanchors.tx"

echo "Starting orderer and peers..."
${COMPOSE_CMD} -f "${DOCKER_COMPOSE_FILE}" up -d

wait_for_container "orderer0.medical-network.com"
wait_for_container "peer0.hospital1.medical-network.com"
wait_for_container "peer0.hospital2.medical-network.com"
wait_for_container "peer0.research.medical-network.com"
wait_for_container "peer0.regulator.medical-network.com"
sleep 5

echo "Creating channel ${CHANNEL_NAME}..."
create_channel

echo "Joining peers to ${CHANNEL_NAME}..."
join_channel "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp"
join_channel "peer0.hospital2.medical-network.com" "HospitalMSP2" "/etc/hyperledger/fabric/users/Admin@hospital2.medical-network.com/msp"
join_channel "peer0.research.medical-network.com" "ResearchOrgMSP" "/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp"
join_channel "peer0.regulator.medical-network.com" "RegulatorMSP" "/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp"

echo "Updating anchor peers..."
update_anchor "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" "HospitalMSP1anchors.tx"
update_anchor "peer0.hospital2.medical-network.com" "HospitalMSP2" "/etc/hyperledger/fabric/users/Admin@hospital2.medical-network.com/msp" "HospitalMSP2anchors.tx"
update_anchor "peer0.research.medical-network.com" "ResearchOrgMSP" "/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp" "ResearchOrgMSPanchors.tx"
update_anchor "peer0.regulator.medical-network.com" "RegulatorMSP" "/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp" "RegulatorMSPanchors.tx"

echo "Deploying chaincode lifecycle for medicalcc..."
"${NETWORK_DIR}/scripts/deploy-chaincode.sh"

echo
echo "Network is up."
echo "Expected containers:"
docker ps --format 'table {{.Names}}\t{{.Status}}' | grep -E 'orderer0.medical-network.com|peer0.hospital1.medical-network.com|peer0.hospital2.medical-network.com|peer0.research.medical-network.com|peer0.regulator.medical-network.com'
echo
echo "Channel membership check:"
peer_exec "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" "peer channel list"
