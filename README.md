# Hyperledger Fabric Medical Information Sharing Network

## Network Architecture

This is a production-grade Hyperledger Fabric v2.x consortium network designed for academic medical information sharing with regulatory oversight.

### Organizations (MSPs)

1. **RegulatorMSP** - Regulatory Authority
   - Acts as the oversight body for medical data sharing
   - Required for medical record creation endorsements
   - Ensures compliance with healthcare regulations

2. **HospitalMSP1** - Hospital Organization 1
   - Primary healthcare provider
   - Can create and access medical records
   - Required for both record creation and authorization approvals

3. **HospitalMSP2** - Hospital Organization 2
   - Secondary healthcare provider
   - Can create and access medical records
   - Required for both record creation and authorization approvals

4. **ResearchOrgMSP** - Research Organization
   - Third-party research institutions
   - Can access anonymized/authorized medical data
   - Participates in the network for research purposes

### Network Topology

- **Ordering Service**: Raft-based (3 orderer nodes for high availability)
- **Channel**: `medical-main-channel` (single main channel)
- **Identity Management**: Fabric CA (one CA per organization)
- **Consensus**: Raft (crash fault-tolerant)

### Endorsement Policies

#### 1. Medical Record Creation Policy
```
AND(HospitalMSP.member, RegulatorMSP.member)
```
**Rationale**: Ensures that medical records are created only with both hospital and regulatory authority approval, maintaining compliance and data integrity. This aligns with the Regulatory Authority model where the regulator must validate all medical data entries.

#### 2. Authorization Approval Policy
```
AND(PatientIdentity, HospitalMSP.member)
```
**Rationale**: Authorization approvals require both the patient's explicit consent (via client certificate) and hospital validation. This ensures patient autonomy while maintaining medical professional oversight.

### Folder Structure

```
.
├── network/
│   ├── docker/
│   │   ├── docker-compose-ca.yaml      # CA services
│   │   ├── docker-compose-orderer.yaml # Orderer nodes
│   │   └── docker-compose-peer.yaml    # Peer nodes
│   ├── configtx/
│   │   └── configtx.yaml               # Channel configuration
│   ├── crypto-config/
│   │   └── crypto-config.yaml          # Crypto material structure
│   ├── channel-artifacts/              # Generated channel artifacts (gitignored)
│   ├── scripts/
│   │   ├── generate-crypto.sh          # Generate crypto material
│   │   ├── generate-channel-artifacts.sh # Generate channel artifacts
│   │   ├── create-channel.sh           # Create and join channel
│   │   └── network.sh                  # Network management
│   ├── ARCHITECTURE.md                 # Detailed architecture documentation
│   └── ENDORSEMENT_POLICIES.md         # Endorsement policy details
├── README.md                           # This file
├── SETUP.md                            # Detailed setup guide
└── .gitignore                          # Git ignore rules
```

### Prerequisites

- Docker and Docker Compose
- Hyperledger Fabric binaries (v2.5.x recommended)
- OpenSSL
- jq (for JSON parsing)

### Quick Start

1. **Generate crypto material**:
   ```bash
   ./network/scripts/generate-crypto.sh
   ```

2. **Generate channel artifacts**:
   ```bash
   ./network/scripts/generate-channel-artifacts.sh
   ```

3. **Start the network**:
   ```bash
   ./network/scripts/network.sh up
   ```

4. **Create the channel**:
   ```bash
   ./network/scripts/create-channel.sh
   ```

### Network Management

- Start: `./network/scripts/network.sh up`
- Stop: `./network/scripts/network.sh down`
- Restart: `./network/scripts/network.sh restart`
- Clean: `./network/scripts/network.sh clean` (removes all data)
- Logs: `./network/scripts/network.sh logs`

### Documentation

- **SETUP.md**: Complete setup and installation guide
- **network/ARCHITECTURE.md**: Detailed network architecture documentation
- **network/ENDORSEMENT_POLICIES.md**: Comprehensive endorsement policy explanations

### Future Extensions

This network is designed to support:
- Private data collections for sensitive patient information
- Encrypted payloads for additional security
- Chaincode implementation for medical record management
- Patient identity management with client certificates

### Regulatory Authority Model Alignment

The network implements a regulatory authority model where:

1. **RegulatorMSP** validates all medical record entries, ensuring:
   - Compliance with healthcare regulations
   - Data privacy law adherence
   - Medical standards compliance
   - Complete audit trails

2. **Hospital Organizations** provide medical validation:
   - Medical accuracy verification
   - Clinical protocol adherence
   - Professional oversight

3. **Patient Identity** ensures patient autonomy:
   - Explicit consent for data sharing
   - Patient-controlled authorizations
   - Privacy-first approach

This model ensures that all medical data on the blockchain is both medically accurate and regulatorily compliant, providing a foundation for trustworthy medical information sharing.
