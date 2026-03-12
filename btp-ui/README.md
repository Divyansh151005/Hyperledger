# BTP UI - Hyperledger Fabric + MinIO

Professional Web3 healthcare dashboard for medical data sharing.

## Structure

```
btp-ui/
  frontend/
  backend/
```

## Run Backend

```bash
cd backend
npm install
cp .env.example .env
node server.js
```

## Run Frontend

```bash
cd frontend
npm install
cp .env.local.example .env.local
npm run dev
```

## Notes

- MetaMask wallet is required for login.
- Backend uses simple Fabric CLI wrappers through `fabricService.js`.
- Set `FABRIC_USE_MOCK=false` and point command env vars to real scripts for live Fabric execution.
