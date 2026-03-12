#!/bin/bash
#
# Fix missing orderer endpoints in channel config for medical-main-channel.
# Runs the full config update flow inside regulator peer container.
#

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [ "$(pwd)" != "${PROJECT_ROOT}" ]; then
    echo "Please run this script from project root:"
    echo "  ${PROJECT_ROOT}"
    exit 1
fi

REGULATOR_CONTAINER="peer0.regulator.medical-network.com"
CHANNEL_NAME="medical-main-channel"
ORDERER_ADDRESS="orderer0.medical-network.com:7050"
ORDERER_TLS_HOSTNAME="orderer0.medical-network.com"
ORDERER_CA="/etc/hyperledger/orderer-tls/ca.crt"
ORDERER_ENDPOINT="orderer0.medical-network.com:7050"
WORKDIR="/tmp/fix-orderer-endpoint"

echo "=========================================="
echo "Fixing orderer endpoint for ${CHANNEL_NAME}"
echo "Container: ${REGULATOR_CONTAINER}"
echo "=========================================="

if ! docker ps --format '{{.Names}}' | grep -qx "${REGULATOR_CONTAINER}"; then
    echo "Required container is not running: ${REGULATOR_CONTAINER}"
    exit 1
fi

docker exec -i "${REGULATOR_CONTAINER}" bash -s -- \
    "${CHANNEL_NAME}" \
    "${ORDERER_ADDRESS}" \
    "${ORDERER_TLS_HOSTNAME}" \
    "${ORDERER_CA}" \
    "${ORDERER_ENDPOINT}" \
    "${WORKDIR}" <<'EOF'
set -euo pipefail

CHANNEL_NAME="$1"
ORDERER_ADDRESS="$2"
ORDERER_TLS_HOSTNAME="$3"
ORDERER_CA="$4"
ORDERER_ENDPOINT="$5"
WORKDIR="$6"

export CORE_PEER_LOCALMSPID=RegulatorMSP
export CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/users/Admin@regulator.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt
export FABRIC_CFG_PATH=/etc/hyperledger/fabric

echo "Preparing workspace: ${WORKDIR}"
rm -rf "${WORKDIR}"
mkdir -p "${WORKDIR}"
cd "${WORKDIR}"

echo "[precheck] Verifying required tools inside container"
for bin in peer jq configtxlator; do
    if ! command -v "${bin}" >/dev/null 2>&1; then
        echo "Missing required binary in container: ${bin}"
        echo "Install ${bin} in ${HOSTNAME} and rerun."
        exit 1
    fi
done

echo "STEP 1: Fetch current channel config block"
peer channel fetch config "${WORKDIR}/config_block.pb" \
    -o "${ORDERER_ADDRESS}" \
    --ordererTLSHostnameOverride "${ORDERER_TLS_HOSTNAME}" \
    -c "${CHANNEL_NAME}" \
    --tls \
    --cafile "${ORDERER_CA}"

echo "STEP 2: Decode config block to JSON"
configtxlator proto_decode \
    --input "${WORKDIR}/config_block.pb" \
    --type common.Block \
    --output "${WORKDIR}/config_block.json"

jq .data.data[0].payload.data.config "${WORKDIR}/config_block.json" > "${WORKDIR}/config.json"

echo "STEP 3: Add orderer discovery endpoints"
ORDERER_ORG_KEY="$(jq -r '.channel_group.groups.Orderer.groups | keys[0]' "${WORKDIR}/config.json")"
if [ -z "${ORDERER_ORG_KEY}" ] || [ "${ORDERER_ORG_KEY}" = "null" ]; then
    echo "Could not detect orderer org key under channel_group.groups.Orderer.groups"
    exit 1
fi
echo "Detected orderer org key: ${ORDERER_ORG_KEY}"

jq --arg endpoint "${ORDERER_ENDPOINT}" --arg org "${ORDERER_ORG_KEY}" '
   .channel_group.values.OrdererAddresses =
     ((.channel_group.values.OrdererAddresses // {"mod_policy":"Admins","value":{"addresses":[]},"version":"0"})
      | .value.addresses = [$endpoint]) |
   .channel_group.groups.Orderer.groups[$org].values.Endpoints =
     ((.channel_group.groups.Orderer.groups[$org].values.Endpoints // {"mod_policy":"Admins","value":{"addresses":[]},"version":"0"})
      | .value.addresses = [$endpoint])
  ' "${WORKDIR}/config.json" > "${WORKDIR}/modified_config.json"

echo "STEP 4: Compute config update"
configtxlator proto_encode \
    --input "${WORKDIR}/config.json" \
    --type common.Config \
    --output "${WORKDIR}/config.pb"

configtxlator proto_encode \
    --input "${WORKDIR}/modified_config.json" \
    --type common.Config \
    --output "${WORKDIR}/modified_config.pb"

configtxlator compute_update \
    --channel_id "${CHANNEL_NAME}" \
    --original "${WORKDIR}/config.pb" \
    --updated "${WORKDIR}/modified_config.pb" \
    --output "${WORKDIR}/config_update.pb"

echo "STEP 5: Wrap config update in envelope"
configtxlator proto_decode \
    --input "${WORKDIR}/config_update.pb" \
    --type common.ConfigUpdate \
    --output "${WORKDIR}/config_update.json"

jq -n \
   --arg channel "${CHANNEL_NAME}" \
   --slurpfile cu "${WORKDIR}/config_update.json" \
   '{payload:{header:{channel_header:{channel_id:$channel,type:2}},data:{config_update:$cu[0]}}}' \
   > "${WORKDIR}/envelope.json"

echo "STEP 6: Encode envelope"
configtxlator proto_encode \
    --input "${WORKDIR}/envelope.json" \
    --type common.Envelope \
    --output "${WORKDIR}/config_update_envelope.pb"

echo "STEP 7: Submit channel config update"
peer channel update \
    -f "${WORKDIR}/config_update_envelope.pb" \
    -c "${CHANNEL_NAME}" \
    -o "${ORDERER_ADDRESS}" \
    --ordererTLSHostnameOverride "${ORDERER_TLS_HOSTNAME}" \
    --tls \
    --cafile "${ORDERER_CA}"

echo "Orderer endpoint update submitted successfully."
echo "Artifacts retained in: ${WORKDIR}"
EOF

echo "=========================================="
echo "Done. Check peer logs for:"
echo "  Connecting to ordering service"
echo "=========================================="
