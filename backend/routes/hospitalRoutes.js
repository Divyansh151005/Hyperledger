/**
 * Hospital routes - upload medical records
 */
const express = require('express');
const multer = require('multer');
const crypto = require('crypto');
const router = express.Router();
const { invoke } = require('../fabricClient');
const minio = require('../minioClient');

const upload = multer({ storage: multer.memoryStorage() });

/**
 * POST /hospital/upload-record
 * Inputs: patientID, file (multipart)
 */
router.post('/upload-record', upload.single('file'), async (req, res) => {
    try {
        const { patientID } = req.body;
        const file = req.file;

        if (!patientID || !file) {
            return res.status(400).json({ error: 'patientID and file are required' });
        }

        // 1. Compute SHA256 hash
        const fileHash = crypto.createHash('sha256').update(file.buffer).digest('hex');

        // 2. Generate record ID
        const recordID = `REC_${Date.now()}`;

        // 3. Upload to MinIO
        const ext = (file.originalname.split('.').pop() || 'pdf').toLowerCase();
        const minioPath = await minio.uploadFile(patientID, recordID, file.buffer, ext);

        // 4. Call Fabric chaincode (use hospital1 or hospital2 based on header/param)
        const org = req.headers['x-org'] || 'hospital1';
        await invoke(org, 'uploadMedicalRecord', recordID, patientID, fileHash, minioPath);

        console.log(`[Hospital] Uploaded record ${recordID} for patient ${patientID}`);
        res.json({
            success: true,
            recordID,
            patientID,
            fileHash,
            minioPath,
            message: 'Record uploaded successfully'
        });
    } catch (err) {
        console.error('[Hospital] Upload error:', err);
        res.status(500).json({ error: err.message });
    }
});

module.exports = router;
