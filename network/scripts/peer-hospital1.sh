#!/bin/bash
#
# Run peer CLI commands with Hospital1 Admin identity inside peer0.hospital1.medical-network.com
# Usage: ./peer-hospital1.sh <peer command and args>
#
# Example:
#   ./peer-hospital1.sh peer lifecycle chaincode querycommitted -C medical-main-channel
#   ./peer-hospital1.sh peer channel list
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER="peer0.hospital1.medical-network.com"

# Load Hospital1 identity (paths are container paths)
ENV_EXPORTS='export CORE_PEER_LOCALMSPID="HospitalMSP1"
export CORE_PEER_MSPCONFIGPATH="/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp"
export CORE_PEER_ADDRESS="peer0.hospital1.medical-network.com:7051"
export CORE_PEER_TLS_ROOTCERT_FILE="/etc/hyperledger/fabric/tls/ca.crt"
export CORE_PEER_TLS_ENABLED="true"
export ORDERER_CA="/etc/hyperledger/orderer-tls/ca.crt"'

docker exec "${CONTAINER}" bash -c "${ENV_EXPORTS} && $*"
