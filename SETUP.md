# Setup Guide - Medical Information Sharing Network

## Prerequisites

Before setting up the network, ensure you have the following installed:

### Required Software

1. **Docker** (version 20.10 or later)
   ```bash
   docker --version
   docker-compose --version
   ```

2. **Hyperledger Fabric Binaries** (v2.5.3 recommended)
   - Download from: https://github.com/hyperledger/fabric/releases
   - Required binaries:
     - `configtxgen`
     - `cryptogen`
     - `peer`
   - Add to your PATH

3. **OpenSSL** (usually pre-installed)
   ```bash
   openssl version
   ```

4. **jq** (for JSON parsing in scripts)
   ```bash
   # macOS
   brew install jq
   
   # Linux
   sudo apt-get install jq
   ```

### System Requirements

- **RAM**: Minimum 8GB, recommended 16GB
- **Disk Space**: At least 10GB free
- **CPU**: Multi-core processor recommended

## Installation Steps

### Step 1: Clone or Navigate to Project Directory

```bash
cd /Users/divyanshbarodiya/Desktop/Hyperledger
```

### Step 2: Generate Cryptographic Material

Generate all necessary certificates and keys:

```bash
./network/scripts/generate-crypto.sh
```

This will create:
- Orderer organization certificates
- Peer organization certificates for all 4 organizations
- TLS certificates for secure communication

**Expected Output:**
```
Generating Crypto Material
========================================
Generating orderer organization crypto material...
Crypto material generated successfully!
```

### Step 3: Generate Channel Artifacts

Generate the genesis block and channel transaction files:

```bash
./network/scripts/generate-channel-artifacts.sh
```

This will create:
- `genesis.block` - Genesis block for the system channel
- `medical-main-channel.tx` - Channel creation transaction
- Anchor peer update transactions for each organization

**Expected Output:**
```
Generating Channel Artifacts
========================================
Generating genesis block for system channel...
Generating channel creation transaction for medical-main-channel...
Channel artifacts generated successfully!
```

### Step 4: Start the Network

Start all network components (CAs, Orderers, Peers):

```bash
./network/scripts/network.sh up
```

This will:
1. Start all 5 Certificate Authorities
2. Start 3 Orderer nodes (Raft cluster)
3. Start 4 Peer nodes (one per organization)

**Expected Output:**
```
Starting Medical Network
========================================
Starting Certificate Authorities...
Starting Orderer nodes...
Starting Peer nodes...
Network started successfully!
```

Wait for all containers to be healthy (check with `docker ps`).

### Step 5: Create and Join Channel

Create the `medical-main-channel` and join all peers:

```bash
./network/scripts/create-channel.sh
```

This will:
1. Create the channel
2. Join all 4 peers to the channel
3. Update anchor peers for each organization

**Expected Output:**
```
Creating Channel: medical-main-channel
========================================
Channel created successfully!
All peers joined channel successfully!
Channel setup completed!
```

### Step 6: Verify Network Status

Check that all containers are running:

```bash
docker ps
```

You should see:
- 5 CA containers
- 3 Orderer containers
- 4 Peer containers

Check network connectivity:

```bash
docker network inspect medical-network
```

## Verification

### Verify Channel Creation

Query channel information from any peer:

```bash
# Set environment for Regulator peer
export CORE_PEER_LOCALMSPID=RegulatorMSP
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_MSPCONFIGPATH=./network/crypto-config/peerOrganizations/regulator.medical-network.com/users/Admin@regulator.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.regulator.medical-network.com:7051
export CORE_PEER_TLS_ROOTCERT_FILE=./network/crypto-config/peerOrganizations/regulator.medical-network.com/peers/peer0.regulator.medical-network.com/tls/ca.crt

# Query channel list
peer channel list
```

You should see `medical-main-channel` in the list.

### Verify Peer Channel Membership

Check that all peers have joined the channel:

```bash
# From Regulator peer
peer channel getinfo -c medical-main-channel

# From Hospital1 peer
export CORE_PEER_LOCALMSPID=HospitalMSP1
export CORE_PEER_MSPCONFIGPATH=./network/crypto-config/peerOrganizations/hospital1.medical-network.com/users/Admin@hospital1.medical-network.com/msp
export CORE_PEER_ADDRESS=peer0.hospital1.medical-network.com:8051
export CORE_PEER_TLS_ROOTCERT_FILE=./network/crypto-config/peerOrganizations/hospital1.medical-network.com/peers/peer0.hospital1.medical-network.com/tls/ca.crt

peer channel getinfo -c medical-main-channel
```

## Network Management

### View Logs

View logs from all containers:

```bash
./network/scripts/network.sh logs
```

View logs from specific service:

```bash
docker logs peer0.regulator.medical-network.com -f
docker logs orderer0.medical-network.com -f
```

### Stop Network

Stop all running containers:

```bash
./network/scripts/network.sh down
```

### Restart Network

Restart the network:

```bash
./network/scripts/network.sh restart
```

### Clean Network

**Warning**: This will remove all containers, volumes, and networks:

```bash
./network/scripts/network.sh clean
```

## Troubleshooting

### Issue: Containers fail to start

**Solution:**
1. Check Docker is running: `docker info`
2. Check port conflicts: `netstat -an | grep -E '7050|7051|7054'`
3. Check logs: `docker logs <container-name>`

### Issue: Channel creation fails

**Solution:**
1. Ensure orderers are running: `docker ps | grep orderer`
2. Check orderer logs: `docker logs orderer0.medical-network.com`
3. Verify genesis block exists: `ls -la network/channel-artifacts/genesis.block`

### Issue: Peer join fails

**Solution:**
1. Ensure channel was created successfully
2. Check peer logs: `docker logs peer0.regulator.medical-network.com`
3. Verify channel block exists: `ls -la network/channel-artifacts/medical-main-channel.block`

### Issue: Crypto material errors

**Solution:**
1. Remove old crypto material: `rm -rf network/crypto-config/ordererOrganizations network/crypto-config/peerOrganizations`
2. Regenerate: `./network/scripts/generate-crypto.sh`

## Next Steps

After successful network setup:

1. **Review Architecture**: See `network/ARCHITECTURE.md`
2. **Understand Policies**: See `network/ENDORSEMENT_POLICIES.md`
3. **Implement Chaincode**: Develop smart contracts for medical records
4. **Set Up Patient Identities**: Register patient identities with Fabric CA
5. **Configure Private Data**: Set up private data collections (future)

## Production Considerations

For production deployment:

1. **Replace cryptogen**: Use Fabric CA for all identity generation
2. **Secure CA Keys**: Use hardware security modules (HSM) for CA keys
3. **Network Security**: Implement network policies and firewalls
4. **Monitoring**: Set up Prometheus/Grafana for monitoring
5. **Backup**: Implement backup strategies for ledger data
6. **Disaster Recovery**: Plan for disaster recovery scenarios
7. **Kubernetes**: Deploy on Kubernetes for better orchestration

## Support

For issues or questions:
- Review Hyperledger Fabric documentation
- Check container logs for errors
- Verify all prerequisites are installed correctly
