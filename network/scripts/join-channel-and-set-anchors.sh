#!/bin/bash

export FABRIC_CFG_PATH=$PWD/network/config

set -e
set -o pipefail

CHANNEL_NAME="medical-main-channel"
ORDERER_ENDPOINT="orderer0.medical-network.com:7050"
ORDERER_TLS_CA="/etc/hyperledger/orderer-tls/ca.crt"
PEER_TLS_ROOTCERT="/etc/hyperledger/fabric/tls/ca.crt"
CHANNEL_BLOCK_PATH="/etc/hyperledger/configtx/medical-main-channel.block"

setGlobals() {
    local container=$1
    local msp=$2
    local admin_msp_path=$3

    GLOBALS_EXPORTS="export CORE_PEER_LOCALMSPID=${msp}
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_ADDRESS=${container}:7051
export CORE_PEER_TLS_ROOTCERT_FILE=${PEER_TLS_ROOTCERT}
export CORE_PEER_MSPCONFIGPATH=${admin_msp_path}"
}

isPeerInChannel() {
    local container=$1
    local msp=$2
    local admin_msp_path=$3

    setGlobals "${container}" "${msp}" "${admin_msp_path}"
    docker exec "${container}" bash -c "
${GLOBALS_EXPORTS}
peer channel list 2>/dev/null
" | grep "${CHANNEL_NAME}" || true
}

joinChannel() {
    local container=$1
    local msp=$2
    local admin_msp_path=$3
    local peer_name=$4

    setGlobals "${container}" "${msp}" "${admin_msp_path}"

    if [[ -n "$(isPeerInChannel "${container}" "${msp}" "${admin_msp_path}")" ]]; then
        echo "✔ ${peer_name} already joined ${CHANNEL_NAME}"
        return 0
    fi

    docker exec "${container}" bash -c "
${GLOBALS_EXPORTS}
peer channel join -b ${CHANNEL_BLOCK_PATH}
"

    echo "✔ ${peer_name} joined channel"
}

updateAnchor() {
    local container=$1
    local msp=$2
    local admin_msp_path=$3
    local anchor_tx=$4

    setGlobals "${container}" "${msp}" "${admin_msp_path}"

    if docker exec "${container}" bash -c "
${GLOBALS_EXPORTS}
peer channel update \
  -o localhost:7050 \
  --ordererTLSHostnameOverride orderer0.medical-network.com \
  -c ${CHANNEL_NAME} \
  -f ${anchor_tx} \
  --tls \
  --cafile ${ORDERER_TLS_CA}
"
    then
        echo "✔ Anchor updated"
    else
        echo "✔ Anchor already configured (skipping)"
    fi
}

echo "================================="
echo "Joining peers to ${CHANNEL_NAME}"
echo "================================="

if [ ! -f "${CHANNEL_BLOCK_PATH}" ]; then
    echo "Channel block not found: ${CHANNEL_BLOCK_PATH}"
    exit 1
fi

echo "Using existing channel block: ${CHANNEL_BLOCK_PATH}"

echo "Joining channel for Regulator"
joinChannel "peer0.regulator.medical-network.com" "RegulatorMSP" "/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp" "Regulator"

echo "Joining channel for Hospital1"
joinChannel "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" "Hospital1"

echo "Joining channel for Hospital2"
joinChannel "peer0.hospital2.medical-network.com" "HospitalMSP2" "/etc/hyperledger/fabric/users/Admin@hospital2.medical-network.com/msp" "Hospital2"

echo "Joining channel for Research"
joinChannel "peer0.research.medical-network.com" "ResearchOrgMSP" "/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp" "Research"

echo
echo "================================="
echo "Updating anchor peers"
echo "================================="

echo "Updating anchor peer for Regulator"
updateAnchor "peer0.regulator.medical-network.com" "RegulatorMSP" "/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp" "/etc/hyperledger/configtx/RegulatorMSPanchors.tx"

echo "Updating anchor peer for Hospital1"
updateAnchor "peer0.hospital1.medical-network.com" "HospitalMSP1" "/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp" "/etc/hyperledger/configtx/HospitalMSP1anchors.tx"

echo "Updating anchor peer for Hospital2"
updateAnchor "peer0.hospital2.medical-network.com" "HospitalMSP2" "/etc/hyperledger/fabric/users/Admin@hospital2.medical-network.com/msp" "/etc/hyperledger/configtx/HospitalMSP2anchors.tx"

echo "Updating anchor peer for Research"
updateAnchor "peer0.research.medical-network.com" "ResearchOrgMSP" "/etc/hyperledger/fabric/users/Admin@research.medical-network.com/msp" "/etc/hyperledger/configtx/ResearchOrgMSPanchors.tx"
