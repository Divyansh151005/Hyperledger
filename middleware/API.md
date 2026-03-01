# Middleware API Documentation

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

All API requests require authentication via X.509 certificate. The certificate ID should be provided in the `X-Certificate-ID` header.

```
X-Certificate-ID: <certificate_id>
```

## Endpoints

### Medical Records

#### Create Medical Record

Creates a new medical record. Only hospitals can create records.

**Endpoint:** `POST /records`

**Headers:**
- `X-Certificate-ID`: Hospital certificate ID

**Request Body:**
```json
{
  "patient_id": "patient-123",
  "data": {
    "diagnosis": "Hypertension",
    "medications": ["Lisinopril 10mg"],
    "notes": "Patient shows improvement"
  }
}
```

**Response:** `201 Created`
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "record_hash": "a1b2c3d4e5f6...",
  "status": "ACTIVE"
}
```

#### Get Medical Record

Retrieves a medical record. Only the owning hospital or regulator can access.

**Endpoint:** `GET /records/:recordID`

**Headers:**
- `X-Certificate-ID`: Hospital or Regulator certificate ID

**Response:** `200 OK`
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "patient_id": "patient-123",
  "hospital_id": "HospitalMSP1",
  "data": {
    "diagnosis": "Hypertension",
    "medications": ["Lisinopril 10mg"],
    "notes": "Patient shows improvement"
  },
  "timestamp": "2024-01-01T00:00:00Z",
  "status": "ACTIVE"
}
```

### Authorization

#### Request Access

Creates a new authorization request. Only research organizations can request access.

**Endpoint:** `POST /authorizations/request`

**Headers:**
- `X-Certificate-ID`: Research organization certificate ID

**Request Body:**
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "duration": 24
}
```

**Response:** `201 Created`
```json
{
  "request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1234567890",
  "status": "REQUESTED"
}
```

#### Approve by Hospital

Approves an authorization request by the hospital. Only the owning hospital can approve.

**Endpoint:** `POST /authorizations/:requestID/approve/hospital`

**Headers:**
- `X-Certificate-ID`: Hospital certificate ID

**Response:** `200 OK`
```json
{
  "status": "approved"
}
```

#### Approve by Patient

Approves an authorization request by the patient. Only the patient can approve.

**Endpoint:** `POST /authorizations/:requestID/approve/patient`

**Headers:**
- `X-Certificate-ID`: Patient certificate ID

**Response:** `200 OK`
```json
{
  "status": "approved"
}
```

#### Revoke Authorization

Revokes an authorization request. Only the patient or owning hospital can revoke.

**Endpoint:** `POST /authorizations/:requestID/revoke`

**Headers:**
- `X-Certificate-ID`: Patient or Hospital certificate ID

**Response:** `200 OK`
```json
{
  "status": "revoked"
}
```

#### Get Authorization

Retrieves an authorization request.

**Endpoint:** `GET /authorizations/:requestID`

**Headers:**
- `X-Certificate-ID`: Certificate ID (must have access)

**Response:** `200 OK`
```json
{
  "request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1234567890",
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "requester_org": "ResearchOrgMSP",
  "hospital_id": "HospitalMSP1",
  "patient_id": "patient-123",
  "status": "GRANTED",
  "created_at": "2024-01-01T00:00:00Z",
  "expires_at": "2024-01-02T00:00:00Z"
}
```

### Sharing

#### Share Record

Shares a medical record with an authorized research organization. Only the owning hospital can share.

**Endpoint:** `POST /sharing/share`

**Headers:**
- `X-Certificate-ID`: Hospital certificate ID

**Request Body:**
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "authorization_request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1234567890"
}
```

**Response:** `201 Created`
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "authorization_request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1234567890",
  "status": "SHARED"
}
```

#### Get Shared Record

Retrieves a shared medical record. Only the requesting research organization or owning hospital can retrieve.

**Endpoint:** `GET /sharing/:recordID/:authRequestID`

**Headers:**
- `X-Certificate-ID`: Research organization or Hospital certificate ID

**Response:** `200 OK`
```json
{
  "record_id": "550e8400-e29b-41d4-a716-446655440000",
  "authorization_request_id": "AUTH-550e8400-e29b-41d4-a716-446655440000-1234567890",
  "data": {
    "diagnosis": "Hypertension",
    "medications": ["Lisinopril 10mg"],
    "notes": "Patient shows improvement"
  },
  "shared_at": "2024-01-01T12:00:00Z"
}
```

### Audit

#### Get Audit Logs

Retrieves audit logs with optional filters.

**Endpoint:** `GET /audit/logs`

**Query Parameters:**
- `event_type` (optional): Filter by event type
- `chaincode_name` (optional): Filter by chaincode name
- `limit` (optional): Maximum number of results (default: 100)
- `offset` (optional): Offset for pagination (default: 0)

**Headers:**
- `X-Certificate-ID`: Certificate ID

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "event_type": "MedicalRecordCreated",
    "chaincode_name": "medical-records",
    "transaction_id": "tx-123",
    "block_number": 100,
    "invoker_msp": "HospitalMSP1",
    "client_id": "User1@hospital1.medical-network.com",
    "payload": {
      "recordID": "550e8400-e29b-41d4-a716-446655440000",
      "patientID": "patient-123"
    },
    "timestamp": "2024-01-01T00:00:00Z",
    "created_at": "2024-01-01T00:00:01Z"
  }
]
```

## Error Responses

All errors follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

**Status Codes:**
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Missing or invalid certificate ID
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Example cURL Requests

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

### Request Access
```bash
curl -X POST http://localhost:8080/api/v1/authorizations/request \
  -H "Content-Type: application/json" \
  -H "X-Certificate-ID: research-cert-123" \
  -d '{
    "record_id": "550e8400-e29b-41d4-a716-446655440000",
    "duration": 24
  }'
```

### Get Shared Record
```bash
curl -X GET http://localhost:8080/api/v1/sharing/550e8400-e29b-41d4-a716-446655440000/AUTH-550e8400-e29b-41d4-a716-446655440000-1234567890 \
  -H "X-Certificate-ID: research-cert-123"
```
