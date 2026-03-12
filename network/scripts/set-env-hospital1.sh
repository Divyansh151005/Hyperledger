#!/bin/bash
#
# Hospital1 Admin identity for peer CLI commands
# Source this before running peer commands inside peer0.hospital1.medical-network.com
#
# Usage:
#   source network/scripts/set-env-hospital1.sh
#   docker exec peer0.hospital1.medical-network.com bash -c "source /tmp/set-env-hospital1.sh && peer lifecycle chaincode querycommitted -C medical-main-channel"
#
# Or from host when running peer in container:
#   . network/scripts/set-env-hospital1.sh
#   docker exec peer0.hospital1.medical-network.com bash -c "export CORE_PEER_LOCALMSPID=$CORE_PEER_LOCALMSPID && ..."

export CORE_PEER_LOCALMSPID="HospitalMSP1"
export CORE_PEER_MSPCONFIGPATH="/etc/hyperledger/fabric/users/Admin@hospital1.medical-network.com/msp"
export CORE_PEER_ADDRESS="peer0.hospital1.medical-network.com:7051"
export CORE_PEER_TLS_ROOTCERT_FILE="/etc/hyperledger/fabric/tls/ca.crt"
export CORE_PEER_TLS_ENABLED="true"
# Orderer TLS CA (for channel/chaincode lifecycle commands)
# Use orderer-tls for orderer connection; fabric/tls is peer's TLS
export ORDERER_CA="/etc/hyperledger/orderer-tls/ca.crt"
