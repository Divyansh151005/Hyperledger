# Medical Information Sharing Middleware

A production-grade middleware layer for Hyperledger Fabric-based medical information sharing platform.

## Overview

This middleware provides a REST API layer that abstracts the complexity of Hyperledger Fabric SDK operations, manages encryption/decryption, handles off-chain storage, maps identities, and provides audit capabilities.

## Features

- **Encryption & Key Management**: Per-authorization-request encryption keys
- **Off-Chain Storage**: Database and object storage for medical records
- **REST API**: Complete API for all operations
- **Identity Mapping**: X.509 certificate to role mapping
- **Audit Integration**: Event listening and audit logging
- **Access Control**: Role-based access control before blockchain invocation

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## API Documentation

See [API.md](./API.md) for complete API documentation.

## Examples

See [EXAMPLES.md](./EXAMPLES.md) for end-to-end request lifecycle examples.

## Prerequisites

- Go 1.21 or later
- PostgreSQL 12 or later
- Hyperledger Fabric v2.x network deployed
- Chaincodes deployed:
  - medical-records
  - authorization-consent
  - anonymous-sharing

## Installation

1. **Clone the repository:**
   ```bash
   cd middleware
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set up environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Initialize database:**
   ```bash
   createdb medical_middleware
   ```

5. **Run the middleware:**
   ```bash
   go run main.go
   ```

## Configuration

Environment variables (see `.env.example`):

```bash
# API Server
API_HOST=0.0.0.0
API_PORT=8080
API_LOG_LEVEL=info

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/medical_middleware?sslmode=disable

# Object Storage (S3)
S3_ENDPOINT=https://s3.amazonaws.com
S3_BUCKET=medical-records
S3_ACCESS_KEY=xxx
S3_SECRET_KEY=xxx
S3_REGION=us-east-1

# Fabric Network
FABRIC_NETWORK_CONFIG_PATH=./network/connection-profile.json
FABRIC_CHANNEL_NAME=medical-main-channel
FABRIC_CHAINCODE_MEDICAL_RECORDS=medical-records
FABRIC_CHAINCODE_AUTHORIZATION=authorization-consent
FABRIC_CHAINCODE_ANONYMOUS_SHARING=anonymous-sharing
FABRIC_PEER_ENDPOINT=localhost:7051
FABRIC_PEER_TLS_CERT=./crypto-config/.../tls/ca.crt
FABRIC_CERT_PATH=./crypto-config/.../signcerts/...cert.pem
FABRIC_KEY_PATH=./crypto-config/.../keystore

# Encryption
ENCRYPTION_KEY_STORE_PATH=./keystore
ENCRYPTION_ALGORITHM=AES-256-GCM
ENCRYPTION_KEY_SIZE=32
```

## Usage

### Start the Middleware

```bash
go run main.go
```

The API server will start on `http://localhost:8080`.

### Health Check

```bash
curl http://localhost:8080/health
```

### Create Medical Record

```bash
curl -X POST http://localhost:8080/api/v1/records \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: hospital-cert-123" \
  -d '{
    "patient_id": "patient-123",
    "data": {
      "diagnosis": "Hypertension",
      "medications": ["Lisinopril 10mg"]
    }
  }'
```

## Project Structure

```
middleware/
├── main.go                    # Application entry point
├── go.mod                     # Go module definition
├── internal/
│   ├── api/                  # API layer
│   │   ├── router.go        # HTTP router
│   │   ├── services.go      # Business logic services
│   │   ├── record_service.go
│   │   ├── authorization_service.go
│   │   └── sharing_service.go
│   ├── config/              # Configuration
│   │   └── config.go
│   ├── fabric/              # Fabric SDK integration
│   │   ├── gateway.go       # Gateway connection
│   │   ├── chaincode_services.go
│   │   └── identity.go      # Identity mapping
│   ├── storage/             # Storage layer
│   │   ├── database.go      # PostgreSQL
│   │   ├── object_storage.go # S3/IPFS
│   │   └── encryption.go    # Encryption service
│   └── audit/               # Audit service
│       └── audit.go
├── ARCHITECTURE.md          # Architecture documentation
├── API.md                   # API documentation
└── EXAMPLES.md              # Usage examples
```

## Key Components

### Encryption Service

- Generates unique encryption keys per `authorizationRequestID`
- Uses AES-256-GCM for symmetric encryption
- Keys stored encrypted at rest
- Supports key rotation

### Identity Service

- Maps X.509 certificates to user roles
- Validates identity before operations
- Enforces role-based access control

### Audit Service

- Listens to Fabric events
- Persists events to database
- Provides query interface for regulators

### Storage Service

- PostgreSQL for metadata and audit logs
- S3-compatible storage for medical records
- In-memory storage for development/testing

## Security Considerations

1. **Encryption**: All medical data encrypted before storage
2. **Access Control**: Multi-layer access control (middleware + chaincode)
3. **Audit Trail**: Complete audit trail for compliance
4. **Key Management**: Secure key storage and management
5. **Identity Validation**: X.509 certificate validation

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o middleware main.go
```

### Docker

```bash
docker build -t medical-middleware .
docker run -p 8080:8080 --env-file .env medical-middleware
```

## License

[Your License Here]

## Contributing

[Contributing Guidelines]

## Support

[Support Information]
