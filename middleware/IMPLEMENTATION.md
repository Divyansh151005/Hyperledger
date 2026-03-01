# Middleware Implementation Summary

## Overview

This document provides a summary of the middleware implementation for the Hyperledger Fabric-based medical information sharing platform.

## Architecture Components

### 1. API Layer (`internal/api/`)

- **Router**: REST API endpoints using Gin framework
- **Services**: Business logic services for records, authorization, and sharing
- **Authentication**: X.509 certificate-based authentication middleware

**Key Files:**
- `router.go`: HTTP route definitions and handlers
- `record_service.go`: Medical record creation and retrieval
- `authorization_service.go`: Authorization request management
- `sharing_service.go`: Record sharing operations

### 2. Fabric Integration (`internal/fabric/`)

- **Gateway**: Fabric Gateway client connection management
- **Chaincode Services**: Wrappers for chaincode invocations
- **Identity Service**: Identity mapping and role validation

**Key Files:**
- `gateway.go`: Fabric Gateway connection and management
- `chaincode_services.go`: Chaincode function wrappers
- `identity.go`: Identity mapping and role management

### 3. Storage Layer (`internal/storage/`)

- **Database**: PostgreSQL for metadata and audit logs
- **Object Storage**: S3-compatible storage for medical records
- **Encryption**: AES-256-GCM encryption with per-authorization keys

**Key Files:**
- `database.go`: PostgreSQL connection and schema
- `object_storage.go`: S3 and in-memory storage implementations
- `encryption.go`: Encryption/decryption service

### 4. Audit Service (`internal/audit/`)

- **Event Listener**: Listens to Fabric blockchain events
- **Audit Service**: Persists events to database
- **Query Interface**: Provides audit log queries

**Key Files:**
- `audit.go`: Event listening and audit logging

## Data Flow

### Create Medical Record

1. Client sends POST request with medical data
2. Middleware validates identity (must be Hospital)
3. Middleware encrypts medical data
4. Middleware stores encrypted data off-chain
5. Middleware computes hash of encrypted data
6. Middleware invokes `CreateMedicalRecord` chaincode
7. Chaincode stores metadata on blockchain
8. Event listener captures event and logs to audit database

### Request Access

1. Research org sends POST request
2. Middleware validates identity (must be Research Org)
3. Middleware invokes `RequestAccess` chaincode
4. Chaincode creates authorization request
5. Event listener captures and logs event

### Approve Authorization

1. Hospital/Patient sends POST request
2. Middleware validates identity and ownership
3. Middleware invokes `ApproveByHospital`/`ApproveByPatient` chaincode
4. Chaincode updates authorization status
5. If status becomes GRANTED, middleware generates encryption key
6. Event listener captures and logs event

### Share Record

1. Hospital sends POST request
2. Middleware validates authorization is GRANTED
3. Middleware retrieves encrypted medical record
4. Middleware encrypts pointer using authorization-specific key
5. Middleware invokes `ShareRecord` chaincode
6. Chaincode stores encrypted pointer in Private Data Collection
7. Event listener captures and logs event

### Retrieve Shared Record

1. Research org sends GET request
2. Middleware validates identity
3. Middleware invokes `GetSharedRecord` chaincode
4. Chaincode returns encrypted pointer from Private Data Collection
5. Middleware decrypts pointer using authorization key
6. Middleware retrieves encrypted medical record from off-chain storage
7. Middleware decrypts medical record
8. Middleware returns decrypted data to client

## Encryption Flow

### Key Management

- **Key Generation**: Unique AES-256 key per `authorizationRequestID`
- **Key Storage**: Keys encrypted at rest using master key
- **Key Retrieval**: Keys retrieved and decrypted on-demand
- **Key Isolation**: Each authorization has its own key

### Encryption Process

1. **Medical Record Encryption**:
   - Data encrypted using hospital-specific key
   - Encrypted data stored off-chain
   - Hash computed and stored on-chain

2. **Pointer Encryption**:
   - Pointer encrypted using authorization-specific key
   - Encrypted pointer stored in Private Data Collection
   - Hash commitment stored on public ledger

3. **Decryption Process**:
   - Retrieve encrypted pointer from blockchain
   - Decrypt pointer using authorization key
   - Retrieve encrypted medical record using pointer
   - Decrypt medical record using hospital key

## Identity Mapping

### Identity Types

- **Patient**: X.509 CN = patientID
- **Hospital**: MSP ID = HospitalMSP1 or HospitalMSP2
- **Research Organization**: MSP ID = ResearchOrgMSP
- **Regulator**: MSP ID = RegulatorMSP

### Identity Validation Flow

1. Client sends request with `X-Certificate-ID` header
2. Middleware queries `identity_mappings` table
3. Middleware validates role and MSP
4. Middleware enforces access control
5. If valid, request proceeds; otherwise, returns 403

## Audit Integration

### Event Listening

- Event listener subscribes to Fabric events
- Events captured from all chaincodes
- Events persisted to `audit_logs` table

### Audit Log Schema

```sql
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    chaincode_name VARCHAR(100) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL,
    block_number BIGINT NOT NULL,
    invoker_msp VARCHAR(100),
    client_id VARCHAR(255),
    payload JSONB,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Query Interface

Regulators can query audit logs via:
```
GET /api/v1/audit/logs?event_type=MedicalRecordCreated&limit=100
```

## Security Features

1. **Multi-Layer Access Control**:
   - Middleware validates identity and role
   - Chaincode enforces access control
   - Both must pass for operation to succeed

2. **Encryption**:
   - All medical data encrypted
   - Keys managed securely
   - Keys isolated per authorization

3. **Audit Trail**:
   - All operations logged
   - Immutable audit log
   - Queryable by regulators

4. **Identity Validation**:
   - X.509 certificate validation
   - Role-based access control
   - MSP-based validation

## Database Schema

### identity_mappings

Stores X.509 certificate to role mappings.

```sql
CREATE TABLE identity_mappings (
    certificate_id VARCHAR(255) PRIMARY KEY,
    msp_id VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL,
    user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### audit_logs

Stores audit log entries from blockchain events.

```sql
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    chaincode_name VARCHAR(100) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL,
    block_number BIGINT NOT NULL,
    invoker_msp VARCHAR(100),
    client_id VARCHAR(255),
    payload JSONB,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### encryption_keys

Stores encryption keys encrypted at rest.

```sql
CREATE TABLE encryption_keys (
    authorization_request_id VARCHAR(255) PRIMARY KEY,
    encrypted_key BYTEA NOT NULL,
    key_metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);
```

### medical_records_metadata

Stores metadata for quick lookups (optional, can be derived from blockchain).

```sql
CREATE TABLE medical_records_metadata (
    record_id VARCHAR(255) PRIMARY KEY,
    patient_id VARCHAR(255) NOT NULL,
    hospital_id VARCHAR(100) NOT NULL,
    record_hash VARCHAR(64) NOT NULL,
    storage_location TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## API Endpoints Summary

### Medical Records
- `POST /api/v1/records` - Create medical record
- `GET /api/v1/records/:recordID` - Get medical record

### Authorization
- `POST /api/v1/authorizations/request` - Request access
- `POST /api/v1/authorizations/:requestID/approve/hospital` - Approve by hospital
- `POST /api/v1/authorizations/:requestID/approve/patient` - Approve by patient
- `POST /api/v1/authorizations/:requestID/revoke` - Revoke authorization
- `GET /api/v1/authorizations/:requestID` - Get authorization

### Sharing
- `POST /api/v1/sharing/share` - Share record
- `GET /api/v1/sharing/:recordID/:authRequestID` - Get shared record

### Audit
- `GET /api/v1/audit/logs` - Get audit logs

## Configuration

All configuration is done via environment variables. See `.env.example` for all available options.

Key configuration areas:
- API server settings
- Database connection
- Object storage (S3) settings
- Fabric network connection
- Encryption settings

## Deployment Considerations

1. **Database**: PostgreSQL with proper backup and replication
2. **Object Storage**: S3-compatible storage with encryption at rest
3. **Key Management**: Secure key management service for master key
4. **TLS**: All communications over TLS
5. **Monitoring**: Logging and monitoring for production
6. **Scaling**: Horizontal scaling of API servers

## Testing

### Unit Tests

Test individual components:
```bash
go test ./internal/storage/...
go test ./internal/fabric/...
go test ./internal/api/...
```

### Integration Tests

Test end-to-end flows:
```bash
go test ./tests/integration/...
```

## Future Enhancements

1. **gRPC API**: Add gRPC API alongside REST
2. **Key Rotation**: Implement key rotation without re-encryption
3. **Caching**: Add Redis caching for frequently accessed data
4. **Rate Limiting**: Implement rate limiting per identity
5. **Metrics**: Add Prometheus metrics
6. **Tracing**: Add distributed tracing
7. **Multi-Channel Support**: Support multiple Fabric channels
8. **IPFS Integration**: Add IPFS as alternative object storage

## Conclusion

The middleware provides a complete abstraction layer for the Hyperledger Fabric medical information sharing platform. It handles encryption, storage, identity mapping, and audit logging while maintaining security and compliance requirements.
