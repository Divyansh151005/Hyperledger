/**
 * MinIO client for medical record file storage
 */
const Minio = require('minio');
const path = require('path');

const MINIO_ENDPOINT = process.env.MINIO_ENDPOINT || 'localhost';
const MINIO_PORT = parseInt(process.env.MINIO_PORT || '9000', 10);
const MINIO_ACCESS_KEY = process.env.MINIO_ACCESS_KEY || 'minioadmin';
const MINIO_SECRET_KEY = process.env.MINIO_SECRET_KEY || 'minioadmin';
const MINIO_BUCKET = process.env.MINIO_BUCKET || 'medical-records';
const MINIO_USE_SSL = process.env.MINIO_USE_SSL === 'true';

const client = new Minio.Client({
    endPoint: MINIO_ENDPOINT,
    port: MINIO_PORT,
    useSSL: MINIO_USE_SSL,
    accessKey: MINIO_ACCESS_KEY,
    secretKey: MINIO_SECRET_KEY
});

/**
 * Ensure bucket exists
 */
async function ensureBucket() {
    try {
        const exists = await client.bucketExists(MINIO_BUCKET);
        if (!exists) {
            await client.makeBucket(MINIO_BUCKET, 'us-east-1');
            console.log(`[MinIO] Created bucket: ${MINIO_BUCKET}`);
        }
    } catch (err) {
        console.error('[MinIO] Bucket setup error:', err.message);
        throw err;
    }
}

/**
 * Upload file to MinIO
 * Path: records/<patientID>/<recordID>.pdf
 */
async function uploadFile(patientID, recordID, fileBuffer, extension = 'pdf') {
    await ensureBucket();
    const objectName = `records/${patientID}/${recordID}.${extension}`;
    const contentType = extension === 'pdf' ? 'application/pdf' : 'application/octet-stream';
    await client.putObject(MINIO_BUCKET, objectName, fileBuffer, fileBuffer.length, {
        'Content-Type': contentType
    });
    return objectName;
}

/**
 * Download file from MinIO
 */
async function downloadFile(minioPath) {
    const stream = await client.getObject(MINIO_BUCKET, minioPath);
    const chunks = [];
    for await (const chunk of stream) {
        chunks.push(chunk);
    }
    return Buffer.concat(chunks);
}

/**
 * Get object path for record
 */
function getObjectPath(patientID, recordID, extension = 'pdf') {
    return `records/${patientID}/${recordID}.${extension}`;
}

module.exports = {
    client,
    ensureBucket,
    uploadFile,
    downloadFile,
    getObjectPath,
    MINIO_BUCKET
};
