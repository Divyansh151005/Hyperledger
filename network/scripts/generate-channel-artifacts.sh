#!/bin/bash

################################################################################
# Generate Channel Artifacts
# This script generates the genesis block and channel transaction files
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
CONFIGTX_DIR="${NETWORK_DIR}/configtx"
CHANNEL_ARTIFACTS_DIR="${NETWORK_DIR}/channel-artifacts"
CONFIGTX_FILE="${CONFIGTX_DIR}/configtx.yaml"

# Channel configuration
SYSTEM_CHANNEL="system-channel"
CHANNEL_NAME="medical-main-channel"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Generating Channel Artifacts${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if configtxgen exists
if ! command -v configtxgen &> /dev/null; then
    echo -e "${RED}Error: configtxgen not found. Please install Hyperledger Fabric binaries.${NC}"
    echo -e "${YELLOW}Download from: https://github.com/hyperledger/fabric/releases${NC}"
    exit 1
fi

# Check if configtx.yaml exists
if [ ! -f "${CONFIGTX_FILE}" ]; then
    echo -e "${RED}Error: configtx.yaml not found at ${CONFIGTX_FILE}${NC}"
    exit 1
fi

# Create channel-artifacts directory if it doesn't exist
mkdir -p "${CHANNEL_ARTIFACTS_DIR}"

# Safeguard: Remove genesis.block if it exists as a directory
if [ -d "${CHANNEL_ARTIFACTS_DIR}/genesis.block" ]; then
    echo -e "${YELLOW}Warning: genesis.block exists as a directory. Removing it...${NC}"
    rm -rf "${CHANNEL_ARTIFACTS_DIR}/genesis.block"
fi

# Safeguard: Remove genesis.block if it exists as a file (for idempotency)
if [ -f "${CHANNEL_ARTIFACTS_DIR}/genesis.block" ]; then
    echo -e "${YELLOW}Removing existing genesis.block file...${NC}"
    rm -f "${CHANNEL_ARTIFACTS_DIR}/genesis.block"
fi

# Safeguard: Remove channel transaction file if it exists (for idempotency)
if [ -f "${CHANNEL_ARTIFACTS_DIR}/${CHANNEL_NAME}.tx" ]; then
    echo -e "${YELLOW}Removing existing ${CHANNEL_NAME}.tx file...${NC}"
    rm -f "${CHANNEL_ARTIFACTS_DIR}/${CHANNEL_NAME}.tx"
fi

# Set FABRIC_CFG_PATH to the configtx directory
export FABRIC_CFG_PATH="${CONFIGTX_DIR}"

# Change to network directory so relative paths in configtx.yaml resolve correctly
cd "${NETWORK_DIR}"

# Generate genesis block for system channel
echo -e "${GREEN}Generating genesis block for system channel...${NC}"
configtxgen -profile MedicalNetworkGenesis -channelID "${SYSTEM_CHANNEL}" -outputBlock "${CHANNEL_ARTIFACTS_DIR}/genesis.block"

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to generate genesis block${NC}"
    exit 1
fi

# Generate channel creation transaction
echo -e "${GREEN}Generating channel creation transaction for ${CHANNEL_NAME}...${NC}"
configtxgen -profile MedicalMainChannel -outputCreateChannelTx "${CHANNEL_ARTIFACTS_DIR}/${CHANNEL_NAME}.tx" -channelID "${CHANNEL_NAME}"

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to generate channel creation transaction${NC}"
    exit 1
fi

# Generate anchor peer updates for each organization
echo -e "${GREEN}Generating anchor peer updates...${NC}"

# Remove existing anchor peer update files for idempotency
for org in RegulatorMSP HospitalMSP1 HospitalMSP2 ResearchOrgMSP; do
    anchor_file="${CHANNEL_ARTIFACTS_DIR}/${org}anchors.tx"
    if [ -f "${anchor_file}" ]; then
        rm -f "${anchor_file}"
    fi
done

configtxgen -profile MedicalMainChannel -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/RegulatorMSPanchors.tx" -channelID "${CHANNEL_NAME}" -asOrg RegulatorMSP
configtxgen -profile MedicalMainChannel -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/HospitalMSP1anchors.tx" -channelID "${CHANNEL_NAME}" -asOrg HospitalMSP1
configtxgen -profile MedicalMainChannel -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/HospitalMSP2anchors.tx" -channelID "${CHANNEL_NAME}" -asOrg HospitalMSP2
configtxgen -profile MedicalMainChannel -outputAnchorPeersUpdate "${CHANNEL_ARTIFACTS_DIR}/ResearchOrgMSPanchors.tx" -channelID "${CHANNEL_NAME}" -asOrg ResearchOrgMSP

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Channel artifacts generated successfully!${NC}"
echo -e "${GREEN}Location: ${CHANNEL_ARTIFACTS_DIR}${NC}"
echo -e "${GREEN}========================================${NC}"

# List generated files
echo -e "\n${YELLOW}Generated files:${NC}"
ls -lh "${CHANNEL_ARTIFACTS_DIR}"
