/**
 * Researcher routes - request access and retrieve records
 */
const express = require('express');
const crypto = require('crypto');
const router = express.Router();
const { invoke, query } = require('../fabricClient');
const minio = require('../minioClient');

/**
 * POST /researcher/request-access
 * Input: recordID
 */
router.post('/request-access', async (req, res) => {
    try {
        const { recordID } = req.body;

        if (!recordID) {
            return res.status(400).json({ error: 'recordID is required' });
        }

        const result = await invoke('research', 'requestAccess', recordID);

        console.log(`[Researcher] Access requested for record ${recordID}`);
        res.json({
            success: true,
            researcherID: result.researcherID,
            request: result,
            message: 'Access request submitted. Save researcherID for patient approval and retrieval.'
        });
    } catch (err) {
        console.error('[Researcher] Request access error:', err);
        res.status(500).json({ error: err.message });
    }
});

/**
 * GET /researcher/retrieve-record
 * Query: recordID
 * Steps: verifyAccess via Fabric -> fetch from MinIO -> verify SHA256 -> return file
 */
router.get('/retrieve-record', async (req, res) => {
    try {
        const { recordID, researcherID } = req.query;

        if (!recordID || !researcherID) {
            return res.status(400).json({ error: 'recordID and researcherID are required' });
        }

        // 1. verifyAccess via Fabric (throws if no active consent)
        const metadata = await query('research', 'verifyAccess', recordID, researcherID);
        if (!metadata) {
            return res.status(403).json({ error: 'Access denied or consent expired' });
        }

        const meta = typeof metadata === 'string' ? JSON.parse(metadata) : (metadata || {});

        // 2. Fetch file from MinIO
        const fileBuffer = await minio.downloadFile(meta.minioPath);

        // 3. Recompute SHA256
        const computedHash = crypto.createHash('sha256').update(fileBuffer).digest('hex');

        // 4. Compare with blockchain hash
        if (computedHash !== meta.fileHash) {
            console.error(`[Researcher] Integrity check FAILED for record ${recordID}`);
            return res.status(500).json({ error: 'File integrity verification failed - hash mismatch' });
        }

        console.log(`[Researcher] Integrity verified for record ${recordID}`);

        // 5. Return file
        const filename = meta.minioPath.split('/').pop() || 'record.pdf';
        const ext = filename.split('.').pop() || 'pdf';
        const contentType = ext === 'pdf' ? 'application/pdf' : 'application/octet-stream';
        res.setHeader('Content-Type', contentType);
        res.setHeader('Content-Disposition', `attachment; filename="${filename}"`);
        res.send(fileBuffer);
    } catch (err) {
        console.error('[Researcher] Retrieve error:', err);
        res.status(500).json({ error: err.message });
    }
});

module.exports = router;
