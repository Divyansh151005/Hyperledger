#!/bin/bash

################################################################################
# Generate Crypto Material for Medical Network
# This script generates all necessary cryptographic material for the network
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
CRYPTO_CONFIG_DIR="${NETWORK_DIR}/crypto-config"
CRYPTO_CONFIG_FILE="${NETWORK_DIR}/crypto-config/crypto-config.yaml"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Generating Crypto Material${NC}"
echo -e "${GREEN}========================================${NC}"

# Check if cryptogen exists
if ! command -v cryptogen &> /dev/null; then
    echo -e "${RED}Error: cryptogen not found. Please install Hyperledger Fabric binaries.${NC}"
    echo -e "${YELLOW}Download from: https://github.com/hyperledger/fabric/releases${NC}"
    exit 1
fi

# Check if crypto-config.yaml exists
if [ ! -f "${CRYPTO_CONFIG_FILE}" ]; then
    echo -e "${RED}Error: crypto-config.yaml not found at ${CRYPTO_CONFIG_FILE}${NC}"
    exit 1
fi

# Remove existing crypto material if it exists
if [ -d "${CRYPTO_CONFIG_DIR}" ] && [ "$(ls -A ${CRYPTO_CONFIG_DIR} 2>/dev/null)" ]; then
    echo -e "${YELLOW}Removing existing crypto material...${NC}"
    rm -rf "${CRYPTO_CONFIG_DIR}"/ordererOrganizations
    rm -rf "${CRYPTO_CONFIG_DIR}"/peerOrganizations
fi

# Generate crypto material
echo -e "${GREEN}Generating orderer organization crypto material...${NC}"
cryptogen generate --config="${CRYPTO_CONFIG_FILE}" --output="${CRYPTO_CONFIG_DIR}"

if [ $? -ne 0 ]; then
    echo -e "${RED}Failed to generate crypto material${NC}"
    exit 1
fi

# Set proper permissions
echo -e "${GREEN}Setting proper permissions...${NC}"
find "${CRYPTO_CONFIG_DIR}" -type f -name "*.pem" -exec chmod 644 {} \;
find "${CRYPTO_CONFIG_DIR}" -type f -name "*.key" -exec chmod 600 {} \;

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Crypto material generated successfully!${NC}"
echo -e "${GREEN}Location: ${CRYPTO_CONFIG_DIR}${NC}"
echo -e "${GREEN}========================================${NC}"

# Display structure
echo -e "\n${YELLOW}Generated structure:${NC}"
tree -L 3 "${CRYPTO_CONFIG_DIR}" 2>/dev/null || find "${CRYPTO_CONFIG_DIR}" -type d -maxdepth 3 | head -20
