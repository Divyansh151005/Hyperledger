#!/bin/bash

export FABRIC_CFG_PATH=$PWD/network/config

################################################################################
# Create Medical Main Channel
# This script creates the medical-main-channel and joins all peers to it
################################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NETWORK_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
CHANNEL_ARTIFACTS_DIR="${NETWORK_DIR}/channel-artifacts"
CRYPTO_CONFIG_DIR="${NETWORK_DIR}/crypto-config"

# Channel configuration
CHANNEL_NAME="medical-main-channel"
ORDERER_ADDRESS="orderer0.medical-network.com:7050"
ORDERER_TLS_CA="${CRYPTO_CONFIG_DIR}/ordererOrganizations/medical-network.com/orderers/orderer0.medical-network.com/tls/ca.crt"

# Organization configurations
ORGS=(
    "RegulatorMSP:regulator.medical-network.com:peer0.regulator.medical-network.com:7051"
    "HospitalMSP1:hospital1.medical-network.com:peer0.hospital1.medical-network.com:8051"
    "HospitalMSP2:hospital2.medical-network.com:peer0.hospital2.medical-network.com:9051"
    "ResearchOrgMSP:research.medical-network.com:peer0.research.medical-network.com:10051"
)

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Creating Channel: ${CHANNEL_NAME}${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if channel transaction exists
CHANNEL_TX="${CHANNEL_ARTIFACTS_DIR}/${CHANNEL_NAME}.tx"
if [ ! -f "${CHANNEL_TX}" ]; then
    echo -e "${RED}Error: Channel transaction file not found at ${CHANNEL_TX}${NC}"
    echo -e "${YELLOW}Please run generate-channel-artifacts.sh first${NC}"
    exit 1
fi

# Function to create channel from an organization
create_channel() {
    local MSP_ID=$1
    local ORG_DOMAIN=$2
    local PEER_NAME=$3
    local PEER_PORT=$4
    
    echo -e "\n${YELLOW}Creating channel from ${MSP_ID}...${NC}"

    docker exec peer0.regulator.medical-network.com bash -c "
        export CORE_PEER_LOCALMSPID=\"RegulatorMSP\"
        export CORE_PEER_TLS_ENABLED=true
        export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
        export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp
        export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
        sleep 10
        peer channel create \
        -o orderer0.medical-network.com:7050 \
        -c medical-main-channel \
        -f /etc/hyperledger/configtx/medical-main-channel.tx \
        --outputBlock /etc/hyperledger/configtx/medical-main-channel.block \
        --tls \
        --cafile /etc/hyperledger/orderer-tls/ca.crt
    "
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Channel created successfully!${NC}"
        return 0
    else
        echo -e "${YELLOW}Channel may already exist, continuing...${NC}"
        return 0
    fi
}

# Function to join peer to channel
join_channel() {
    local MSP_ID=$1
    local ORG_DOMAIN=$2
    local PEER_NAME=$3
    local PEER_PORT=$4
    
    echo -e "\n${YELLOW}Joining ${PEER_NAME} to channel...${NC}"

    docker exec "${PEER_NAME}" bash -c "
        export CORE_PEER_LOCALMSPID=\"${MSP_ID}\"
        export CORE_PEER_TLS_ENABLED=true
        export CORE_PEER_ADDRESS=${PEER_NAME}:7051
        export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
        export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@${ORG_DOMAIN}/msp
        peer channel join \
        -b /etc/hyperledger/configtx/${CHANNEL_NAME}.block \
        --tls \
        --cafile /etc/hyperledger/orderer-tls/ca.crt
    "
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}${PEER_NAME} joined channel successfully!${NC}"
    else
        echo -e "${RED}Failed to join ${PEER_NAME} to channel${NC}"
        return 1
    fi
}

# Function to update anchor peers
update_anchor_peers() {
    local MSP_ID=$1
    local ORG_DOMAIN=$2
    local PEER_NAME=$3
    local PEER_PORT=$4
    local ANCHOR_TX=$5
    
    echo -e "\n${YELLOW}Updating anchor peers for ${MSP_ID}...${NC}"
    
    export CORE_PEER_LOCALMSPID="${MSP_ID}"
    export CORE_PEER_TLS_ENABLED=true
    export CORE_PEER_MSPCONFIGPATH="${CRYPTO_CONFIG_DIR}/peerOrganizations/${ORG_DOMAIN}/users/Admin@${ORG_DOMAIN}/msp"
    export CORE_PEER_ADDRESS="${PEER_NAME}:${PEER_PORT}"
    export CORE_PEER_TLS_ROOTCERT_FILE="${CRYPTO_CONFIG_DIR}/peerOrganizations/${ORG_DOMAIN}/peers/${PEER_NAME}/tls/ca.crt"
    
    peer channel update -o "${ORDERER_ADDRESS}" \
        -c "${CHANNEL_NAME}" \
        -f "${ANCHOR_TX}" \
        --tls \
        --cafile "${ORDERER_TLS_CA}"
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Anchor peers updated for ${MSP_ID}!${NC}"
    else
        echo -e "${YELLOW}Anchor peer update may have failed, but continuing...${NC}"
    fi
}

# Create channel (only need to do this once)
FIRST_ORG="${ORGS[0]}"
IFS=':' read -r MSP_ID ORG_DOMAIN PEER_NAME PEER_PORT <<< "${FIRST_ORG}"
create_channel "${MSP_ID}" "${ORG_DOMAIN}" "${PEER_NAME}" "${PEER_PORT}"

# Join all peers to channel
for org in "${ORGS[@]}"; do
    IFS=':' read -r MSP_ID ORG_DOMAIN PEER_NAME PEER_PORT <<< "${org}"
    join_channel "${MSP_ID}" "${ORG_DOMAIN}" "${PEER_NAME}" "${PEER_PORT}"
done

# Update anchor peers
update_anchor_peers "RegulatorMSP" "regulator.medical-network.com" "peer0.regulator.medical-network.com" "7051" "${CHANNEL_ARTIFACTS_DIR}/RegulatorMSPanchors.tx"
update_anchor_peers "HospitalMSP1" "hospital1.medical-network.com" "peer0.hospital1.medical-network.com" "8051" "${CHANNEL_ARTIFACTS_DIR}/HospitalMSP1anchors.tx"
update_anchor_peers "HospitalMSP2" "hospital2.medical-network.com" "peer0.hospital2.medical-network.com" "9051" "${CHANNEL_ARTIFACTS_DIR}/HospitalMSP2anchors.tx"
update_anchor_peers "ResearchOrgMSP" "research.medical-network.com" "peer0.research.medical-network.com" "10051" "${CHANNEL_ARTIFACTS_DIR}/ResearchOrgMSPanchors.tx"

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Channel setup completed!${NC}"
echo -e "${GREEN}Channel Name: ${CHANNEL_NAME}${NC}"
echo -e "${GREEN}========================================${NC}"
