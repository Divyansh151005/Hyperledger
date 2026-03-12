const crypto = require("crypto");
const path = require("path");
const Minio = require("minio");

const MINIO_ENDPOINT = process.env.MINIO_ENDPOINT || "127.0.0.1";
const MINIO_PORT = Number(process.env.MINIO_PORT || "9000");
const MINIO_USE_SSL = process.env.MINIO_USE_SSL === "true";
const MINIO_ACCESS_KEY = process.env.MINIO_ACCESS_KEY || "minioadmin";
const MINIO_SECRET_KEY = process.env.MINIO_SECRET_KEY || "minioadmin";
const MINIO_BUCKET = process.env.MINIO_BUCKET || "medical-records";

const client = new Minio.Client({
  endPoint: MINIO_ENDPOINT,
  port: MINIO_PORT,
  useSSL: MINIO_USE_SSL,
  accessKey: MINIO_ACCESS_KEY,
  secretKey: MINIO_SECRET_KEY
});

async function ensureBucket() {
  try {
    const exists = await client.bucketExists(MINIO_BUCKET);
    if (!exists) {
      await client.makeBucket(MINIO_BUCKET, "us-east-1");
    }
  } catch (err) {
    console.warn("[MinIO] Bucket check/create failed:", err.message);
    throw err;
  }
}

function computeHash(buffer) {
  return crypto.createHash("sha256").update(buffer).digest("hex");
}

function buildObjectName(recordId, originalName) {
  const ext = path.extname(originalName || "") || ".pdf";
  return `reports/${recordId}${ext}`;
}

async function uploadMedicalRecord({ patientId, recordId, fileBuffer, originalName }) {
  const objectName = buildObjectName(recordId, originalName);
  const ext = path.extname(originalName || "").toLowerCase() || ".pdf";
  const contentType = ext === ".pdf" ? "application/pdf" : "application/octet-stream";
  await ensureBucket();
  await client.putObject(MINIO_BUCKET, objectName, fileBuffer, fileBuffer.length, {
    "Content-Type": contentType
  });
  return { objectName, hash: computeHash(fileBuffer), extension: ext, contentType };
}

async function downloadMedicalRecord(objectName) {
  const stream = await client.getObject(MINIO_BUCKET, objectName);
  const chunks = [];
  for await (const chunk of stream) chunks.push(chunk);
  return Buffer.concat(chunks);
}

async function listMedicalReports() {
  await ensureBucket();
  const stream = client.listObjects(MINIO_BUCKET, "reports", true);
  const results = [];
  for await (const obj of stream) {
    results.push({
      name: obj.name,
      size: obj.size,
      lastModified: obj.lastModified
    });
  }
  return results;
}

module.exports = {
  MINIO_BUCKET,
  uploadMedicalRecord,
  downloadMedicalRecord,
  computeHash,
  listMedicalReports
};
