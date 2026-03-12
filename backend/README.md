# Medical Records Backend

Express API for medical data sharing with Hyperledger Fabric and MinIO.

## Setup

```bash
npm install
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 3000 | Server port |
| PEER_HOST | localhost | Peer host (use localhost when peers are port-mapped) |
| PEER_PORT | 11051 | Regulator peer port (7051 mapped to 11051) |
| CRYPTO_PATH | ../network/crypto-config | Path to Fabric crypto material |
| MINIO_ENDPOINT | localhost | MinIO host |
| MINIO_PORT | 9000 | MinIO port |
| MINIO_ACCESS_KEY | minioadmin | MinIO access key |
| MINIO_SECRET_KEY | minioadmin | MinIO secret key |

## Run

```bash
# Start MinIO first
docker-compose -f ../docker-compose.minio.yaml up -d

# Start backend
npm start
```

## API Endpoints

See `../DEMO.md` for full request examples.
