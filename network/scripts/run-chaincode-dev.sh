#!/bin/bash

echo "--------------------------------------"
echo "Starting Medical Chaincode (Dev Mode)"
echo "--------------------------------------"

echo "Running chaincode inside peer container..."

docker exec -it peer0.regulator.medical-network.com bash -c "
cd /opt/gopath/src/github.com/chaincode/medical-records 2>/dev/null || true

if [ ! -d /tmp/medicalcc ]; then
mkdir -p /tmp/medicalcc
fi

cp -r /etc/hyperledger/* /tmp/medicalcc 2>/dev/null || true

echo 'Chaincode container environment ready'
"

echo ""
echo "NOTE:"
echo "Chaincode logic will be executed using peer chaincode invoke commands."
echo "You can now run transactions to generate logs."
