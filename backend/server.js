/**
 * Medical Records Backend - Express server with Fabric + MinIO
 */
const express = require('express');
const cors = require('cors');
const path = require('path');
const hospitalRoutes = require('./routes/hospitalRoutes');
const patientRoutes = require('./routes/patientRoutes');
const researcherRoutes = require('./routes/researcherRoutes');
const regulatorRoutes = require('./routes/regulatorRoutes');
const { listenForEvents } = require('./fabricClient');
const minio = require('./minioClient');

const PORT = process.env.PORT || 3000;

const app = express();
app.use(cors());
app.use(express.json());

// Routes
app.use('/hospital', hospitalRoutes);
app.use('/patient', patientRoutes);
app.use('/researcher', researcherRoutes);
app.use('/regulator', regulatorRoutes);

// Health check
app.get('/health', (req, res) => {
    res.json({ status: 'ok', service: 'medical-records-backend' });
});

// Demo info
app.get('/', (req, res) => {
    res.json({
        service: 'Medical Records API',
        version: '1.0',
        endpoints: {
            hospital: {
                'POST /hospital/upload-record': 'Upload medical record (patientID, file)'
            },
            researcher: {
                'POST /researcher/request-access': 'Request access (recordID)',
                'GET /researcher/retrieve-record': 'Retrieve record (recordID, researcherID)'
            },
            patient: {
                'POST /patient/approve-request': 'Approve (requestID, expiryTimestamp)',
                'POST /patient/reject-request': 'Reject (requestID)'
            },
            regulator: {
                'GET /regulator/audit-records': 'List all records',
                'GET /regulator/audit-consents': 'List all consents',
                'POST /regulator/revoke-consent': 'Revoke (consentID)'
            }
        }
    });
});

// Error handler
app.use((err, req, res, next) => {
    console.error(err);
    res.status(500).json({ error: err.message });
});

async function start() {
    try {
        await minio.ensureBucket();
    } catch (err) {
        console.warn('[MinIO] Could not connect - ensure MinIO is running. File operations will fail.');
    }

    app.listen(PORT, () => {
        console.log(`[Server] Medical Records API running on http://localhost:${PORT}`);
    });

    // Event listener (non-blocking)
    try {
        await listenForEvents((eventName, payload) => {
            console.log(`[Event] ${eventName}:`, JSON.stringify(payload));
        });
        console.log('[Server] Chaincode event listener started');
    } catch (err) {
        console.warn('[Server] Event listener could not start:', err.message);
    }
}

start().catch(err => {
    console.error('Failed to start server:', err);
    process.exit(1);
});
