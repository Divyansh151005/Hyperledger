const express = require("express");
const cors = require("cors");
const multer = require("multer");
const path = require("path");
const { EventEmitter } = require("events");
require("dotenv").config();
const {
  createTxId,
  getBlockchainInfoFromCli,
  getPeerLogs,
  listRecords,
  listAccessRequests,
  queryRecord,
  queryAccessRequest,
  revokeAccess,
  approveByHospital,
  approveUploadByPatient,
  submitAccessRequest,
  submitApproval,
  submitUpload
} = require("./fabricService");
const {
  computeHash,
  downloadMedicalRecord,
  listMedicalReports,
  uploadEncryptedRecord
} = require("./minioService");
const {
  decryptFile,
  decryptKeyWithPrivateKey,
  encryptFile,
  encryptKeyWithPublicKey,
  generateAESKey,
  getKey,
  readPrivateKey,
  readPublicKey,
  reEncryptKeyForUser,
  revokeKey,
  saveKey,
  validateKey
} = require("./modules/keyManager");

const app = express();
const upload = multer({ storage: multer.memoryStorage() });
const PORT = Number(process.env.PORT || "4000");
const FABRIC_USE_MOCK = process.env.FABRIC_USE_MOCK !== "false";
const stream = new EventEmitter();

app.use(cors());
app.use(express.json());

const STATUS = {
  PENDING: "PENDING",
  PENDING_HOSPITAL_APPROVAL: "PENDING_HOSPITAL_APPROVAL",
  PENDING_PATIENT_APPROVAL: "PENDING_PATIENT_APPROVAL",
  APPROVED: "APPROVED",
  ACTIVE: "ACTIVE",
  EXPIRED: "EXPIRED",
  REVOKED: "REVOKED",
  REJECTED: "REJECTED"
};

const db = {
  uploadRequests: [],
  records: [],
  accessRequests: [],
  consents: [],
  activity: [],
  latestTransactions: [],
  blockchain: {
    blockHeight: 0,
    currentBlockHash: "",
    previousBlockHash: ""
  }
};

function pushActivity(type, message, details = {}) {
  const entry = {
    id: createTxId(),
    type,
    message,
    details,
    timestamp: new Date().toISOString()
  };
  db.activity.unshift(entry);
  db.activity = db.activity.slice(0, 80);
  stream.emit("activity", { eventName: "ActivityUpdated", data: entry });
}

function createPseudoHash() {
  return createTxId().replace("tx_", "").padEnd(64, "0").slice(0, 64);
}

function commitBlockchainEvent({ txId, recordId, action }) {
  const nextHeight = db.blockchain.blockHeight + 1;
  const nextHash = createPseudoHash();
  db.blockchain = {
    blockHeight: nextHeight,
    currentBlockHash: nextHash,
    previousBlockHash: db.blockchain.currentBlockHash || createPseudoHash()
  };
  db.latestTransactions.unshift({
    block: nextHeight,
    txId,
    recordId: recordId || "-",
    action: action || "invoke",
    timestamp: new Date().toISOString()
  });
  db.latestTransactions = db.latestTransactions.slice(0, 60);
}

function bytesToMb(size) {
  return `${(Number(size || 0) / (1024 * 1024)).toFixed(2)} MB`;
}

function toExpiryMillis(expirySeconds, expiryTimestamp) {
  if (expiryTimestamp) {
    return Number(expiryTimestamp);
  }
  const seconds = Number(expirySeconds || 0);
  if (!Number.isFinite(seconds) || seconds <= 0) return null;
  return Date.now() + seconds * 1000;
}

function normalizeRecord(record) {
  const normalizedRecordId = record.recordId || record.recordID;
  const normalizedPatientId = record.patientId || record.patientID;
  const normalizedHospitalId = record.hospitalID || record.hospitalId;
  const normalizedFileHash = record.fileHash || record.recordHash;
  const normalizedMinioPath = record.minioPath || record.objectName;
  return {
    ...record,
    recordId: normalizedRecordId,
    patientId: normalizedPatientId,
    hospitalID: normalizedHospitalId,
    fileHash: normalizedFileHash,
    minioPath: normalizedMinioPath
  };
}

function findRecord(recordId) {
  return db.records.find((item) => item.recordId === String(recordId));
}

function isFabricUnavailableError(err) {
  const message = String(err?.message || "");
  return (
    message.includes("UNAVAILABLE") ||
    message.includes("ECONNREFUSED") ||
    message.includes("No connection established")
  );
}

function shouldUseLocalAccessFlow() {
  return FABRIC_USE_MOCK;
}

function accessCondition({ role, consentValid, keyValid, expired }) {
  const roleValid = role === "researcher";
  if (roleValid && consentValid && keyValid && !expired) return true;
  return false;
}

function evaluateResearcherAccess(recordId, researcherWallet) {
  const record = findRecord(recordId);
  if (!record) {
    return { allowed: false, statusCode: 404, message: "Record not found", status: "DENIED" };
  }

  const consent = db.consents.find(
    (item) =>
      item.recordId === String(recordId) &&
      item.researcherWallet === String(researcherWallet)
  );
  if (!consent || consent.status !== STATUS.ACTIVE) {
    return {
      allowed: false,
      statusCode: 403,
      message: "Consent not active",
      status: "DENIED"
    };
  }

  const isExpired = consent.expiry && Date.now() > Number(consent.expiry);
  if (isExpired) {
    consent.status = STATUS.EXPIRED;
    revokeKey(recordId, researcherWallet);
    pushActivity("access_expired", "Access expired due to consent/key expiry", {
      recordId,
      researcherWallet
    });
  }

  const keyResult = validateKey(recordId, researcherWallet);
  const allowed = accessCondition({
    role: "researcher",
    consentValid: consent.status === STATUS.ACTIVE,
    keyValid: keyResult.valid,
    expired: isExpired
  });
  if (!allowed) {
    return {
      allowed: false,
      statusCode: 403,
      message: isExpired ? "Access expired" : `Access denied: ${keyResult.reason}`,
      status: isExpired ? STATUS.EXPIRED : "DENIED",
      consent,
      keyResult,
      record
    };
  }
  return { allowed: true, consent, keyResult, record };
}

async function evaluateOnChainAccess(consent) {
  if (shouldUseLocalAccessFlow()) {
    return {
      ok: true,
      request: {
        hospitalApproved: true,
        patientApproved: true,
        status: STATUS.ACTIVE,
        expiry: Number(consent.expiry || 0)
      }
    };
  }
  const request = await queryAccessRequest({ requestId: consent.requestId });
  if (!request) {
    return { ok: false, statusCode: 403, message: "On-chain request not found" };
  }
  const expired = Number(request.expiry || 0) > 0 && Date.now() > Number(request.expiry || 0);
  if (!request.hospitalApproved || !request.patientApproved) {
    return { ok: false, statusCode: 403, message: "On-chain approval incomplete" };
  }
  if (request.status === STATUS.REVOKED) {
    return { ok: false, statusCode: 403, message: "Access revoked on-chain" };
  }
  if (expired || request.status !== STATUS.ACTIVE) {
    return { ok: false, statusCode: 403, message: "Access expired on-chain" };
  }
  return { ok: true, request };
}

app.get("/", (_req, res) => {
  res.json({
    service: "BTP UI Backend - Hybrid Encrypted Sharing",
    endpoints: [
      "POST /upload-request",
      "POST /patient/approve-upload",
      "POST /request-access",
      "POST /hospital/approve-access",
      "POST /patient/approve-access",
      "GET /preview/:recordId",
      "POST /patient/revoke-access"
    ]
  });
});

app.post("/upload-request", upload.single("file"), async (req, res) => {
  try {
    const patientId = String(req.body.patientId || "");
    const hospitalID = String(req.body.hospitalID || req.body.walletAddress || "");
    const file = req.file;
    if (!patientId || !hospitalID || !file) {
      return res.status(400).json({ error: "patientId, hospitalID/walletAddress, and file are required" });
    }

    const requestId = `uplreq_${Date.now()}`;
    const recordId = `rec_${Date.now()}`;
    const aesKey = generateAESKey();
    const { encrypted, iv } = encryptFile(file.buffer, aesKey);
    const minioUpload = await uploadEncryptedRecord({ recordId, encryptedBuffer: encrypted });
    const fabricResponse = await submitUpload({
      recordID: recordId,
      patientID: patientId,
      hospitalID,
      fileHash: minioUpload.hash,
      minioPath: minioUpload.objectName
    });
    const transactionId = fabricResponse.transactionId || createTxId();

    db.uploadRequests.unshift({
      requestId,
      recordId,
      patientId,
      hospitalID,
      fileName: file.originalname,
      fileHash: minioUpload.hash,
      minioPath: minioUpload.objectName,
      iv: iv.toString("base64"),
      aesKey: aesKey.toString("base64"),
      status: STATUS.PENDING_PATIENT_APPROVAL,
      createdAt: new Date().toISOString(),
      transactionId
    });
    pushActivity("upload_request_created", "Upload request created and pending patient approval", {
      requestId,
      recordId,
      patientId,
      hospitalID
    });
    return res.json({
      requestId,
      recordId,
      transactionId,
      fileHash: minioUpload.hash,
      minioPath: minioUpload.objectName,
      status: STATUS.PENDING_PATIENT_APPROVAL
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Upload request failed" });
  }
});

app.post("/patient/approve-upload", async (req, res) => {
  try {
    const { requestId, patientId } = req.body;
    if (!requestId || !patientId) {
      return res.status(400).json({ error: "requestId and patientId are required" });
    }
    const uploadRequest = db.uploadRequests.find(
      (item) =>
        item.requestId === String(requestId) &&
        item.patientId === String(patientId) &&
        item.status === STATUS.PENDING_PATIENT_APPROVAL
    );
    if (!uploadRequest) {
      return res.status(404).json({ error: "Pending upload request not found" });
    }

    const recordId = uploadRequest.recordId;
    const aesKey = Buffer.from(uploadRequest.aesKey, "base64");
    pushActivity("aes_key_generated", "AES key generated for approved upload", { recordId });
    const onChain = await approveUploadByPatient({ recordId });
    const transactionId = onChain.transactionId || createTxId();

    const patientPublicKey = readPublicKey(uploadRequest.patientId);
    const encryptedPatientKey = encryptKeyWithPublicKey(aesKey, patientPublicKey);
    saveKey({
      recordId,
      user: uploadRequest.patientId,
      encryptedKey: encryptedPatientKey.toString("base64"),
      expiry: null,
      valid: true
    });
    pushActivity("key_encrypted", "AES key encrypted and stored for patient", {
      recordId,
      user: uploadRequest.patientId
    });

    db.records.unshift({
      recordId,
      patientId: uploadRequest.patientId,
      hospitalID: uploadRequest.hospitalID,
      fileHash: uploadRequest.fileHash,
      minioPath: uploadRequest.minioPath,
      objectName: uploadRequest.minioPath,
      iv: uploadRequest.iv,
      originalName: uploadRequest.fileName,
      extension: path.extname(uploadRequest.fileName || ".pdf") || ".pdf",
      contentType: "application/pdf",
      createdAt: new Date().toISOString(),
      transactionId,
      status: STATUS.APPROVED
    });
    uploadRequest.status = STATUS.APPROVED;
    uploadRequest.aesKey = null;
    uploadRequest.approvedAt = new Date().toISOString();

    commitBlockchainEvent({ txId: transactionId, recordId, action: "upload_approved" });
    pushActivity("patient_approved_upload", "Patient approved upload and encrypted file stored", {
      requestId,
      recordId,
      minioPath: uploadRequest.minioPath
    });

    return res.json({
      requestId,
      recordId,
      transactionId,
      fileHash: uploadRequest.fileHash,
      minioPath: uploadRequest.minioPath,
      status: STATUS.APPROVED
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Patient upload approval failed" });
  }
});

app.post("/patient/reject-upload", (req, res) => {
  const { requestId, patientId } = req.body;
  if (!requestId || !patientId) {
    return res.status(400).json({ error: "requestId and patientId are required" });
  }
  const uploadRequest = db.uploadRequests.find(
    (item) =>
      item.requestId === String(requestId) &&
      item.patientId === String(patientId) &&
      item.status === STATUS.PENDING_PATIENT_APPROVAL
  );
  if (!uploadRequest) {
    return res.status(404).json({ error: "Pending upload request not found" });
  }
  uploadRequest.status = STATUS.REJECTED;
  uploadRequest.aesKey = null;
  pushActivity("upload_request_rejected", "Patient rejected upload request", { requestId });
  return res.json({ message: "Upload request rejected", status: STATUS.REJECTED });
});

app.get("/patient/upload-requests", (req, res) => {
  const patientId = String(req.query.patientId || "");
  if (!patientId) return res.status(400).json({ error: "patientId is required" });
  return res.json({
    requests: db.uploadRequests
      .filter((item) => item.patientId === patientId)
      .map((item) => ({
        requestId: item.requestId,
        patientId: item.patientId,
        hospitalID: item.hospitalID,
        fileName: item.fileName,
        status: item.status,
        createdAt: item.createdAt,
        recordId: item.recordId || null
      }))
  });
});

app.post("/request-access", async (req, res) => {
  try {
    const { recordId, researcherWallet } = req.body;
    if (!recordId || !researcherWallet) {
      return res.status(400).json({ error: "recordId and researcherWallet are required" });
    }
    let existingRecord = findRecord(recordId);
    if (!existingRecord) {
      const onChainRecord = await queryRecord({ recordId });
      if (!onChainRecord || !onChainRecord.recordID && !onChainRecord.recordId) {
        return res.status(404).json({ error: "Record not found" });
      }
      existingRecord = normalizeRecord(onChainRecord);
    }

    const tx = await submitAccessRequest({
      requestId: `req_${Date.now()}`,
      recordId,
      researcherID: researcherWallet
    });
    const requestId = tx.requestId || `req_${Date.now()}`;
    const transactionId = tx.transactionId || createTxId();

    db.accessRequests.unshift({
      requestId,
      recordId,
      patientId: existingRecord.patientId,
      hospitalID: existingRecord.hospitalID,
      researcherWallet,
      status: STATUS.PENDING,
      requestedAt: new Date().toISOString(),
      hospitalApprovedAt: null,
      patientApprovedAt: null
    });
    commitBlockchainEvent({ txId: transactionId, recordId, action: "request_access" });
    pushActivity("access_request_created", "Researcher requested access", {
      requestId,
      recordId,
      researcherWallet
    });
    return res.json({ message: "Access request submitted", requestId, transactionId, status: STATUS.PENDING });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Request failed" });
  }
});

app.post("/hospital/approve-access", async (req, res) => {
  try {
    const { requestId, hospitalWallet } = req.body;
    if (!requestId || !hospitalWallet) {
      return res.status(400).json({ error: "requestId and hospitalWallet are required" });
    }
    const pending = db.accessRequests.find(
      (item) => item.requestId === String(requestId) && item.status === STATUS.PENDING
    );
    if (!pending) {
      return res.status(404).json({ error: "Pending request not found" });
    }
    const tx = await approveByHospital({ requestId });
    const onChainRequest = shouldUseLocalAccessFlow()
      ? null
      : await queryAccessRequest({ requestId });
    pending.status =
      shouldUseLocalAccessFlow() || onChainRequest?.hospitalApproved
        ? STATUS.PENDING_PATIENT_APPROVAL
        : STATUS.PENDING;
    pending.hospitalApprovedAt = new Date().toISOString();
    pending.hospitalWallet = hospitalWallet;
    const transactionId = tx.transactionId || createTxId();
    commitBlockchainEvent({ txId: transactionId, recordId: pending.recordId, action: "hospital_approve" });
    pushActivity("hospital_approved_access", "Hospital approved access request", {
      requestId,
      recordId: pending.recordId
    });
    return res.json({
      message: "Request approved by hospital",
      status: pending.status,
      transactionId
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Hospital approval failed" });
  }
});

app.post("/patient/approve-access", async (req, res) => {
  try {
    const { requestId, patientId, expirySeconds, expiryTimestamp } = req.body;
    if (!requestId || !patientId) {
      return res.status(400).json({ error: "requestId and patientId are required" });
    }
    const pending = db.accessRequests.find(
      (item) =>
        item.requestId === String(requestId) &&
        item.patientId === String(patientId) &&
        item.status === STATUS.PENDING_PATIENT_APPROVAL
    );
    if (!pending) {
      return res.status(404).json({ error: "Patient-pending request not found" });
    }
    const onChainBefore = shouldUseLocalAccessFlow()
      ? { hospitalApproved: true }
      : await queryAccessRequest({ requestId: pending.requestId });
    if (!onChainBefore?.hospitalApproved) {
      return res.status(403).json({ error: "Hospital approval missing on-chain" });
    }

    const expiry = toExpiryMillis(expirySeconds, expiryTimestamp);
    if (!expiry) {
      return res.status(400).json({ error: "A valid expirySeconds or expiryTimestamp is required" });
    }
    const consentId = `cons_${Date.now()}`;

    const shared = reEncryptKeyForUser({
      recordId: pending.recordId,
      fromUser: patientId,
      toUser: pending.researcherWallet,
      expiry
    });
    pushActivity("key_shared", "Patient shared re-encrypted key with researcher", {
      recordId: pending.recordId,
      researcherWallet: pending.researcherWallet,
      expiry
    });

    pending.status = STATUS.APPROVED;
    pending.patientApprovedAt = new Date().toISOString();
    db.consents.unshift({
      consentId,
      requestId: pending.requestId,
      recordId: pending.recordId,
      patientId,
      researcherWallet: pending.researcherWallet,
      hospitalID: pending.hospitalID,
      expiry,
      status: STATUS.ACTIVE,
      keyRefUser: shared.user
    });

    const tx = await submitApproval({
      requestId: pending.requestId,
      expiry
    });
    const onChainAfter = shouldUseLocalAccessFlow()
      ? { patientApproved: true }
      : await queryAccessRequest({ requestId: pending.requestId });
    if (!onChainAfter?.patientApproved) {
      return res.status(500).json({ error: "Patient approval was not committed on-chain" });
    }
    const transactionId = tx.transactionId || createTxId();
    commitBlockchainEvent({ txId: transactionId, recordId: pending.recordId, action: "grant_access" });
    pushActivity("access_granted", "Access granted after patient approval and key validation", {
      requestId: pending.requestId,
      consentId
    });

    return res.json({
      message: "Access approved by patient",
      consentId,
      expiry,
      status: STATUS.ACTIVE,
      transactionId
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Patient approval failed" });
  }
});

app.post("/patient/reject-access", (req, res) => {
  const { requestId, patientId } = req.body;
  if (!requestId || !patientId) {
    return res.status(400).json({ error: "requestId and patientId are required" });
  }
  const pending = db.accessRequests.find(
    (item) =>
      item.requestId === String(requestId) &&
      item.patientId === String(patientId) &&
      item.status === STATUS.PENDING_PATIENT_APPROVAL
  );
  if (!pending) return res.status(404).json({ error: "Pending request not found" });
  pending.status = STATUS.REJECTED;
  pushActivity("access_request_rejected", "Patient rejected access request", { requestId });
  return res.json({ message: "Access request rejected", status: STATUS.REJECTED });
});

app.post("/patient/revoke-access", async (req, res) => {
  try {
    const { consentId, requestId } = req.body;
    const consent = consentId
      ? db.consents.find((item) => item.consentId === String(consentId))
      : db.consents.find((item) => item.requestId === String(requestId));
    if (!consent) return res.status(404).json({ error: "Consent not found" });

    const tx = await revokeAccess({ requestId: consent.requestId });
    consent.status = STATUS.REVOKED;
    revokeKey(consent.recordId, consent.researcherWallet);
    const transactionId = tx.transactionId || createTxId();
    commitBlockchainEvent({ txId: transactionId, recordId: consent.recordId, action: "revoke_access" });
    pushActivity("access_revoked", "Patient revoked access and key validity", {
      consentId: consent.consentId,
      recordId: consent.recordId
    });
    return res.json({
      message: "Access revoked",
      status: STATUS.REVOKED,
      transactionId
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Revoke failed" });
  }
});

app.post("/patient/revoke", (req, res) => {
  req.url = "/patient/revoke-access";
  app._router.handle(req, res);
});

app.get("/preview/:recordId", async (req, res) => {
  try {
    const recordId = String(req.params.recordId || "");
    const researcherWallet = String(req.query.researcherWallet || "");
    const role = String(req.query.role || "researcher");
    if (!recordId || !researcherWallet) {
      return res.status(400).json({ error: "recordId and researcherWallet are required" });
    }

    await queryRecord({ recordId, researcherWallet });
    const access = evaluateResearcherAccess(recordId, researcherWallet);
    if (!access.allowed) {
      return res.status(access.statusCode).json({
        status: access.status,
        message: access.message
      });
    }
    const chainAccess = await evaluateOnChainAccess(access.consent);
    if (!chainAccess.ok) {
      return res.status(chainAccess.statusCode).json({
        status: "DENIED",
        message: chainAccess.message
      });
    }

    if (role !== "researcher") {
      return res.status(403).json({ status: "DENIED", message: "Preview endpoint is researcher-only" });
    }

    const encryptedFile = await downloadMedicalRecord(access.record.objectName);
    const keyEntry = getKey(recordId, researcherWallet);
    const privateKey = readPrivateKey(researcherWallet);
    const decryptedAesKey = decryptKeyWithPrivateKey(
      Buffer.from(keyEntry.encryptedKey, "base64"),
      privateKey
    );
    const plainBuffer = decryptFile(
      encryptedFile,
      decryptedAesKey,
      Buffer.from(access.record.iv, "base64")
    );

    res.setHeader("Content-Type", access.record.contentType || "application/pdf");
    res.setHeader("Content-Disposition", "inline");
    pushActivity("access_granted", "Researcher previewed decrypted medical data", {
      recordId,
      researcherWallet
    });
    commitBlockchainEvent({ txId: createTxId(), recordId, action: "preview_access" });
    return res.send(plainBuffer);
  } catch (err) {
    return res.status(500).json({ error: err.message || "Preview failed" });
  }
});

app.get("/retrieveRecord", async (req, res) => {
  const role = String(req.query.role || "researcher");
  if (role === "researcher") {
    return res.status(403).json({ message: "Researchers are preview-only. Use /preview/:recordId" });
  }
  return res.status(403).json({ message: "Direct download is disabled in encrypted mode" });
});

app.get("/retrieveRecord/:recordId", async (_req, res) => {
  return res.status(403).json({ message: "Direct retrieval disabled. Use preview with approved key." });
});

app.post("/verifyIntegrity", async (req, res) => {
  try {
    const { recordId, patientId } = req.body;
    if (!recordId || !patientId) {
      return res.status(400).json({ error: "recordId and patientId are required" });
    }

    const record = findRecord(recordId);
    if (!record) return res.status(404).json({ error: "Record not found" });
    if (record.patientId !== patientId) {
      return res.status(400).json({ error: "Patient ID does not match this record" });
    }

    const fileBuffer = await downloadMedicalRecord(record.objectName);
    const computedHash = computeHash(fileBuffer);
    return res.json({
      blockchainHash: record.fileHash,
      computedHash,
      verified: record.fileHash === computedHash
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Integrity verification failed" });
  }
});

app.get("/verify-hash/:recordId", async (req, res) => {
  try {
    const recordId = req.params.recordId;
    const record = findRecord(recordId);
    if (!record) return res.status(404).json({ error: "Record not found" });
    const encryptedBuffer = await downloadMedicalRecord(record.objectName);
    const fileHash = computeHash(encryptedBuffer);
    return res.json({
      blockchainHash: record.fileHash,
      fileHash,
      verified: record.fileHash === fileHash
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Verification failed" });
  }
});

app.get("/records", async (req, res) => {
  try {
    const patientId = String(req.query.patientId || "");
    const chainRecords = await listRecords();
    const allRecords = chainRecords.records?.length ? chainRecords.records : db.records;
    const mapped = allRecords.map(normalizeRecord);
    const records = patientId
      ? mapped.filter((record) => String(record.patientId) === patientId)
      : mapped;
    return res.json({ records });
  } catch (err) {
    if (isFabricUnavailableError(err)) {
      const patientId = String(req.query.patientId || "");
      const mapped = db.records.map(normalizeRecord);
      const records = patientId
        ? mapped.filter((record) => String(record.patientId) === patientId)
        : mapped;
      return res.json({ records, source: "fallback-local" });
    }
    return res.status(500).json({ error: err.message || "Could not list records" });
  }
});

app.get("/activity", (_req, res) => {
  res.json({ activity: db.activity });
});

app.get("/timeline", (_req, res) => {
  res.json({ timeline: db.activity });
});

app.get("/pendingRequests", (_req, res) => {
  res.json({
    requests: db.accessRequests.filter((item) => item.status === STATUS.PENDING)
  });
});

app.get("/hospital/access-requests", async (_req, res) => {
  try {
    const chainRequests = await listAccessRequests();
    if (chainRequests.length) {
      const pending = chainRequests
        .filter((item) => item.status === STATUS.PENDING && !item.hospitalApproved)
        .map((item) => ({
          requestId: item.requestID,
          recordId: item.recordID,
          patientId: item.patientID,
          hospitalID: item.hospitalID,
          researcherWallet: item.researcherID,
          status: item.status,
          requestedAt: item.createdAt
        }));
      return res.json({ requests: pending });
    }
    return res.json({
      requests: db.accessRequests.filter((item) => item.status === STATUS.PENDING)
    });
  } catch (err) {
    if (isFabricUnavailableError(err)) {
      return res.json({
        requests: db.accessRequests.filter((item) => item.status === STATUS.PENDING),
        source: "fallback-local"
      });
    }
    return res.status(500).json({ error: err.message || "Failed to query hospital access requests" });
  }
});

app.get("/patient/access-requests", async (req, res) => {
  const patientId = String(req.query.patientId || "");
  if (!patientId) {
    return res.status(400).json({ error: "patientId is required" });
  }
  try {
    const chainRequests = await listAccessRequests();
    if (chainRequests.length) {
      const patientPending = chainRequests
        .filter(
          (item) =>
            item.patientID === patientId &&
            item.status === STATUS.PENDING &&
            item.hospitalApproved &&
            !item.patientApproved
        )
        .map((item) => ({
          requestId: item.requestID,
          recordId: item.recordID,
          patientId: item.patientID,
          hospitalID: item.hospitalID,
          researcherWallet: item.researcherID,
          status: STATUS.PENDING_PATIENT_APPROVAL,
          requestedAt: item.createdAt
        }));
      return res.json({ requests: patientPending });
    }
    return res.json({
      requests: db.accessRequests.filter(
        (item) =>
          item.patientId === patientId && item.status === STATUS.PENDING_PATIENT_APPROVAL
      )
    });
  } catch (err) {
    if (isFabricUnavailableError(err)) {
      return res.json({
        requests: db.accessRequests.filter(
          (item) =>
            item.patientId === patientId && item.status === STATUS.PENDING_PATIENT_APPROVAL
        ),
        source: "fallback-local"
      });
    }
    return res.status(500).json({ error: err.message || "Failed to query patient access requests" });
  }
});

app.get("/patient/consents", (req, res) => {
  const patientId = String(req.query.patientId || "");
  if (!patientId) {
    return res.status(400).json({ error: "patientId is required" });
  }
  const now = Date.now();
  db.consents.forEach((consent) => {
    if (consent.status === STATUS.ACTIVE && consent.expiry && now > Number(consent.expiry)) {
      consent.status = STATUS.EXPIRED;
      revokeKey(consent.recordId, consent.researcherWallet);
    }
  });
  return res.json({
    consents: db.consents.filter((consent) => consent.patientId === patientId)
  });
});

app.get("/researcher/access", async (req, res) => {
  try {
    const recordId = String(req.query.recordId || "");
    const researcherWallet = String(req.query.researcherWallet || "");
    if (!recordId || !researcherWallet) {
      return res.status(400).json({ error: "recordId and researcherWallet are required" });
    }
    const check = evaluateResearcherAccess(recordId, researcherWallet);
    if (!check.allowed) {
      return res.status(check.statusCode).json({
        status: check.status,
        message: check.message
      });
    }
    const chainAccess = await evaluateOnChainAccess(check.consent);
    if (!chainAccess.ok) {
      return res.status(chainAccess.statusCode).json({
        status: "DENIED",
        message: chainAccess.message
      });
    }
    const remainingMs = Number(check.consent.expiry) - Date.now();
    return res.json({
      status: STATUS.ACTIVE,
      expiresAt: Number(check.consent.expiry),
      remainingMs: Math.max(0, remainingMs)
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Failed to evaluate researcher access" });
  }
});

app.get("/minio-files", async (_req, res) => {
  try {
    const files = await listMedicalReports();
    res.json(
      files.map((file) => ({
        name: file.name,
        size: file.size,
        sizeLabel: bytesToMb(file.size),
        lastModified: file.lastModified
      }))
    );
  } catch (err) {
    res.status(500).json({ error: err.message || "Failed to list MinIO files" });
  }
});

app.get("/blockchain-info", async (_req, res) => {
  if (!FABRIC_USE_MOCK) {
    try {
      const cliInfo = await getBlockchainInfoFromCli();
      db.blockchain = {
        blockHeight: cliInfo.blockHeight || db.blockchain.blockHeight,
        currentBlockHash: cliInfo.currentBlockHash || db.blockchain.currentBlockHash,
        previousBlockHash: cliInfo.previousBlockHash || db.blockchain.previousBlockHash
      };
    } catch (_err) {
      // Fall back to in-memory values when CLI command is unavailable.
    }
  }
  const transactionCount = db.latestTransactions.filter(
    (item) => item.block === db.blockchain.blockHeight
  ).length;
  res.json({ ...db.blockchain, transactionCount });
});

app.get("/latest-transactions", async (_req, res) => {
  if (!FABRIC_USE_MOCK) {
    try {
      const logs = await getPeerLogs();
      const rows = logs
        .split("\n")
        .filter((line) => line.toLowerCase().includes("txid"))
        .slice(-20)
        .map((line, index) => {
          const txMatch = line.match(/txid[=: ]+([a-zA-Z0-9_-]+)/i);
          const recMatch = line.match(/rec_\d+/i);
          return {
            block: db.blockchain.blockHeight - index,
            txId: txMatch?.[1] || createTxId(),
            recordId: recMatch?.[0] || "-",
            action: "invoke",
            timestamp: new Date().toISOString()
          };
        });
      if (rows.length) {
        db.latestTransactions = rows.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1));
      }
    } catch (_err) {
      // Ignore log parsing errors and keep current transaction cache.
    }
  }
  res.json(db.latestTransactions.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1)));
});

app.get("/events", (req, res) => {
  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");
  res.flushHeaders?.();

  const handler = (payload) => {
    res.write(`data: ${JSON.stringify(payload)}\n\n`);
  };
  stream.on("activity", handler);
  req.on("close", () => {
    stream.off("activity", handler);
    res.end();
  });
});

// Legacy aliases retained for UI compatibility.
app.post("/uploadRecord", upload.single("file"), (req, res) => {
  req.url = "/upload-request";
  app._router.handle(req, res);
});

app.post("/requestAccess", (req, res) => {
  req.url = "/request-access";
  app._router.handle(req, res);
});

app.post("/approveAccess", (req, res) => {
  req.url = "/hospital/approve-access";
  req.body.hospitalWallet = req.body.approverWallet || req.body.hospitalWallet;
  app._router.handle(req, res);
});

app.post("/grant-consent", (req, res) => {
  req.url = "/patient/approve-access";
  req.body.patientId = req.body.patientWallet || req.body.patientId;
  app._router.handle(req, res);
});

app.post("/reject-consent", (req, res) => {
  req.url = "/patient/reject-access";
  req.body.patientId = req.body.patientWallet || req.body.patientId;
  app._router.handle(req, res);
});

app.post("/revoke-consent", (req, res) => {
  req.url = "/patient/revoke-access";
  app._router.handle(req, res);
});

app.listen(PORT, () => {
  console.log(`Backend running on http://localhost:${PORT}`);
});
