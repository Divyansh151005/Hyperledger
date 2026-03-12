#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}"
cd "${PROJECT_ROOT}"

set_platform_for_apple_silicon() {
    if [[ "$(uname -s)" == "Darwin" && "$(uname -m)" == "arm64" ]]; then
        export DOCKER_DEFAULT_PLATFORM=linux/amd64
        echo "Using DOCKER_DEFAULT_PLATFORM=${DOCKER_DEFAULT_PLATFORM} for Apple Silicon"
    fi
}

validate_peer_mspconfigpath() {
    local peers=(
        "peer0.regulator.medical-network.com"
        "peer0.hospital1.medical-network.com"
        "peer0.hospital2.medical-network.com"
        "peer0.research.medical-network.com"
    )

    for peer in "${peers[@]}"; do
        local msp_config_path
        msp_config_path="$(docker exec "${peer}" sh -c 'echo "${CORE_PEER_MSPCONFIGPATH}"')"
        if [[ "${msp_config_path}" != "/etc/hyperledger/fabric/msp" ]]; then
            echo "Invalid CORE_PEER_MSPCONFIGPATH in ${peer}: ${msp_config_path}"
            exit 1
        fi

        docker exec "${peer}" test -d /etc/hyperledger/fabric/msp
        docker exec "${peer}" test -d /etc/hyperledger/fabric/tls
    done
}

set_platform_for_apple_silicon

echo "================================="
echo "Starting Medical Blockchain Network"
echo "================================="

# Ensure each run starts from a clean ledger state.
echo "Cleaning previous ledger data..."
rm -rf data
rm -rf system-genesis-block
rm -rf network/channel-artifacts/*.block

mkdir -p data
mkdir -p system-genesis-block
mkdir -p network/channel-artifacts

./network/scripts/generate-channel-artifacts.sh
docker compose \
  -f network/docker/docker-compose-ca.yaml \
  -f network/docker/docker-compose-orderer.yaml \
  -f network/docker/docker-compose-peer.yaml \
  up -d --force-recreate --remove-orphans

echo "Waiting for Orderer0 to start..."

until docker logs orderer0.medical-network.com 2>&1 | grep -q "Beginning to serve requests"; do
  echo "Orderer not ready yet..."
  sleep 5
done

echo "Orderer is ready!"

echo "Waiting for orderer gRPC port..."
until docker exec orderer0.medical-network.com bash -c "</dev/tcp/localhost/7050" 2>/dev/null; do
  echo "Orderer gRPC not ready yet..."
  sleep 2
done

echo "Orderer gRPC is ready!"

validate_peer_mspconfigpath

echo
echo "================================="
echo "Creating Channel"
echo "================================="

echo
echo "================================="
echo "Checking if channel exists"
echo "================================="

CHANNEL_BLOCK="network/channel-artifacts/medical-main-channel.block"
if [ -f "$CHANNEL_BLOCK" ]; then
    echo "Debug: channel block exists at $CHANNEL_BLOCK: yes"
else
    echo "Debug: channel block exists at $CHANNEL_BLOCK: no"
fi

if docker exec peer0.regulator.medical-network.com peer channel list 2>/dev/null | grep -q "medical-main-channel" && [ -f "$CHANNEL_BLOCK" ]; then
    echo "✔ Channel medical-main-channel already exists"
    echo "Skipping channel creation"
else
    echo "Channel missing or block missing"
    echo "Recreating channel..."
    ./network/scripts/create-channel.sh
fi

echo
echo "================================="
echo "Deploying Smart Contract"
echo "================================="
echo "Starting chaincode in dev mode..."
docker compose -f network/docker/docker-compose-peer.yaml up -d medicalcc

sleep 5
echo "Chaincode started."

echo
echo "================================="
echo "Network startup completed"
echo "================================="
