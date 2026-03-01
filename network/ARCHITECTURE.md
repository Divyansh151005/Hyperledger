# Medical Information Sharing Network Architecture

## Executive Summary

This document describes the architecture of a production-grade Hyperledger Fabric v2.x consortium network designed for academic medical information sharing with regulatory oversight. The network implements a multi-organization blockchain that ensures data integrity, regulatory compliance, and patient privacy.

## Network Topology

### Organizations (MSPs)

The network consists of four organizations, each with distinct roles:

#### 1. RegulatorMSP (Regulatory Authority)
- **Role**: Regulatory oversight and compliance validation
- **Domain**: `regulator.medical-network.com`
- **Components**:
  - 1 Certificate Authority (CA)
  - 1 Peer node (peer0)
  - Admin and client identities
- **Responsibilities**:
  - Validate medical record creation
  - Ensure regulatory compliance
  - Maintain audit trails
  - Oversee data sharing policies

#### 2. HospitalMSP1 (Hospital Organization 1)
- **Role**: Primary healthcare provider
- **Domain**: `hospital1.medical-network.com`
- **Components**:
  - 1 Certificate Authority (CA)
  - 1 Peer node (peer0)
  - Admin and client identities
- **Responsibilities**:
  - Create and manage medical records
  - Validate medical data accuracy
  - Participate in authorization approvals

#### 3. HospitalMSP2 (Hospital Organization 2)
- **Role**: Secondary healthcare provider
- **Domain**: `hospital2.medical-network.com`
- **Components**:
  - 1 Certificate Authority (CA)
  - 1 Peer node (peer0)
  - Admin and client identities
- **Responsibilities**:
  - Create and manage medical records
  - Validate medical data accuracy
  - Participate in authorization approvals

#### 4. ResearchOrgMSP (Research Organization)
- **Role**: Third-party research institutions
- **Domain**: `research.medical-network.com`
- **Components**:
  - 1 Certificate Authority (CA)
  - 1 Peer node (peer0)
  - Admin and client identities
- **Responsibilities**:
  - Access authorized research data
  - Participate in research data sharing
  - Maintain research integrity

### Ordering Service

- **Type**: Raft (crash fault-tolerant consensus)
- **Nodes**: 3 orderer nodes for high availability
- **Domain**: `medical-network.com`
- **Orderer Nodes**:
  - orderer0.medical-network.com
  - orderer1.medical-network.com
  - orderer2.medical-network.com

### Channel Configuration

- **Channel Name**: `medical-main-channel`
- **Type**: Application channel
- **Consortium**: MedicalConsortium
- **Organizations**: All four peer organizations participate

## Network Components

### Certificate Authorities (Fabric CA)

Each organization operates its own Fabric CA for identity management:

1. **ca.regulator.medical-network.com** (Port 7054)
2. **ca.hospital1.medical-network.com** (Port 8054)
3. **ca.hospital2.medical-network.com** (Port 9054)
4. **ca.research.medical-network.com** (Port 10054)
5. **ca.orderer.medical-network.com** (Port 6054)

### Peer Nodes

Each organization operates one peer node:

1. **peer0.regulator.medical-network.com** (Port 7051)
2. **peer0.hospital1.medical-network.com** (Port 8051)
3. **peer0.hospital2.medical-network.com** (Port 9051)
4. **peer0.research.medical-network.com** (Port 10051)

### Network Communication

- **Network Name**: `medical-network` (Docker bridge network)
- **TLS**: Enabled for all communications
- **Gossip**: Enabled for peer-to-peer communication
- **Service Discovery**: Enabled for dynamic peer discovery

## Security Architecture

### Identity Management

- **Fabric CA**: Each organization manages its own identities
- **MSP Structure**: Node OUs enabled for role-based access
- **Client Certificates**: Patients and users issued client certificates
- **TLS**: All communications encrypted with TLS

### Access Control

- **MSP-based**: Access control through Membership Service Providers
- **Channel-based**: Organizations must be members of the channel
- **Policy-based**: Endorsement policies control transaction validation

### Cryptographic Material

- **Structure**: Organized by organization and node type
- **Storage**: Crypto material stored in `crypto-config/` directory
- **Generation**: Using `cryptogen` tool (for development) or Fabric CA (for production)

## Endorsement Policy Architecture

### Policy 1: Medical Record Creation

**Requirement**: `AND(HospitalMSP.peer, RegulatorMSP.peer)`

**Flow**:
1. Hospital creates a medical record transaction
2. Transaction requires endorsement from:
   - Hospital peer (validates medical accuracy)
   - Regulator peer (validates regulatory compliance)
3. Both endorsements required before transaction is committed

**Rationale**: Ensures regulatory oversight and medical accuracy for all medical records.

### Policy 2: Authorization Approval

**Requirement**: `AND(PatientIdentity, HospitalMSP.peer)`

**Flow**:
1. Patient (via client certificate) requests authorization
2. Transaction requires endorsement from:
   - Patient identity (client certificate)
   - Hospital peer (validates request legitimacy)
3. Both endorsements required before authorization is approved

**Rationale**: Ensures patient consent and medical validation for data sharing.

## Data Model (Future)

### Medical Records
- Patient ID
- Medical data (encrypted)
- Timestamp
- Hospital ID
- Regulatory approval status

### Authorizations
- Patient ID
- Authorized parties
- Access scope
- Expiration date
- Approval status

### Audit Trail
- Transaction ID
- Endorsers
- Timestamp
- Organization IDs

## Scalability Considerations

### Horizontal Scaling
- Additional peer nodes can be added per organization
- Additional orderer nodes can be added to the Raft cluster
- Additional organizations can join the consortium

### Performance Optimization
- Gossip protocol for efficient peer communication
- Service discovery for dynamic peer location
- Channel partitioning for data isolation (future)

## High Availability

### Orderer High Availability
- 3-node Raft cluster provides fault tolerance
- Can tolerate 1 node failure
- Automatic leader election

### Peer High Availability
- Multiple peers per organization (future)
- Gossip-based state synchronization
- Automatic failover capabilities

## Network Deployment

### Development Environment
- Docker Compose for local deployment
- Single host deployment
- Simplified configuration

### Production Environment
- Kubernetes deployment (recommended)
- Multi-host deployment
- Production-grade security
- Monitoring and logging

## Monitoring and Operations

### Health Checks
- All containers include health checks
- Docker health check endpoints
- Peer and orderer health endpoints

### Logging
- Structured logging (INFO level)
- Container logs accessible via Docker
- Operations endpoints for metrics

### Operations Ports
- CA Operations: 17054, 18054, 19054, 20054, 16054
- Peer Operations: 9443, 9444, 9445, 9446
- Orderer Operations: 8443, 8444, 8445

## Future Enhancements

### Private Data Collections
- Sensitive patient data in private collections
- Selective data sharing
- Enhanced privacy protection

### Chaincode Implementation
- Medical record management chaincode
- Authorization management chaincode
- Query and reporting chaincode

### Patient Identity Management
- Patient registration system
- Client certificate issuance
- Identity verification

### Encryption
- Field-level encryption
- Encrypted payloads
- Key management integration

## Compliance and Regulatory Alignment

### Healthcare Regulations
- HIPAA compliance considerations
- GDPR compliance (right to be forgotten)
- Medical data privacy laws

### Audit Requirements
- Complete transaction history
- Regulatory oversight validation
- Patient consent tracking

## References

- [Hyperledger Fabric Documentation](https://hyperledger-fabric.readthedocs.io/)
- [Fabric CA Documentation](https://hyperledger-fabric-ca.readthedocs.io/)
- [Fabric Samples](https://github.com/hyperledger/fabric-samples)
