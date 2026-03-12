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
  queryRecord,
  submitAccessRequest,
  submitApproval,
  submitUpload
} = require("./fabricService");
const {
  computeHash,
  downloadMedicalRecord,
  listMedicalReports,
  uploadMedicalRecord
} = require("./minioService");

const app = express();
const upload = multer({ storage: multer.memoryStorage() });
const PORT = Number(process.env.PORT || "4000");
const FABRIC_USE_MOCK = process.env.FABRIC_USE_MOCK !== "false";
const stream = new EventEmitter();

app.use(cors());
app.use(express.json());

const db = {
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

function pushActivity(type, message) {
  const entry = {
    id: createTxId(),
    type,
    message,
    timestamp: new Date().toISOString()
  };
  db.activity.unshift(entry);
  db.activity = db.activity.slice(0, 40);
  stream.emit("activity", {
    eventName:
      type === "upload"
        ? "RecordUploaded"
        : type === "request"
          ? "AccessRequested"
          : type === "approval"
            ? "ConsentGranted"
            : type === "retrieval"
              ? "RecordRetrieved"
              : "ActivityUpdated",
    data: entry
  });
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
  db.latestTransactions = db.latestTransactions.slice(0, 50);
}

function bytesToMb(size) {
  return `${(Number(size || 0) / (1024 * 1024)).toFixed(2)} MB`;
}

app.get("/", (_req, res) => {
  res.json({
    service: "BTP UI Backend",
    endpoints: [
      "POST /uploadRecord",
      "POST /requestAccess",
      "POST /approveAccess",
      "GET /retrieveRecord",
      "GET /records",
      "GET /activity"
    ]
  });
});

app.post("/uploadRecord", upload.single("file"), async (req, res) => {
  try {
    const { patientId, walletAddress } = req.body;
    const file = req.file;
    if (!patientId || !walletAddress || !file) {
      return res.status(400).json({ error: "patientId, walletAddress, and file are required" });
    }

    const recordId = `rec_${Date.now()}`;
    const { objectName, hash, extension, contentType } = await uploadMedicalRecord({
      recordId,
      fileBuffer: file.buffer,
      originalName: file.originalname
    });
    const fabricResponse = await submitUpload({
      recordId,
      patientId,
      walletAddress,
      objectName,
      hash
    });

    const transactionId = fabricResponse.transactionId || createTxId();
    db.records.unshift({
      recordId,
      patientId,
      walletAddress,
      objectName,
      fileHash: hash,
      extension,
      contentType,
      timestamp: new Date().toISOString(),
      transactionId
    });
    commitBlockchainEvent({ txId: transactionId, recordId, action: "upload" });
    pushActivity("upload", `Hospital uploaded record ${recordId}`);

    return res.json({
      transactionId,
      recordId,
      fileHash: hash
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Upload failed" });
  }
});

app.post("/requestAccess", async (req, res) => {
  try {
    const { recordId, researcherWallet } = req.body;
    if (!recordId || !researcherWallet) {
      return res.status(400).json({ error: "recordId and researcherWallet are required" });
    }

    const existingRecord = db.records.find((item) => item.recordId === recordId);
    if (!existingRecord) {
      return res.status(404).json({ error: "Record not found" });
    }

    const requestId = `req_${Date.now()}`;
    const tx = await submitAccessRequest({ requestId, recordId, researcherWallet });
    const transactionId = tx.transactionId || createTxId();

    db.accessRequests.unshift({
      requestId,
      recordId,
      patientId: existingRecord.patientId,
      researcherWallet,
      status: "pending",
      requestedAt: new Date().toISOString()
    });
    commitBlockchainEvent({ txId: transactionId, recordId, action: "request" });
    pushActivity("request", "Researcher requested access");
    return res.json({ message: "Access request submitted", requestId, transactionId });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Request failed" });
  }
});

app.post("/approveAccess", async (req, res) => {
  try {
    const { requestId, recordId, researcherWallet, expirySeconds, approverWallet } = req.body;
    if (!requestId || !recordId || !researcherWallet || !expirySeconds || !approverWallet) {
      return res.status(400).json({ error: "Missing required fields" });
    }

    const pending = db.accessRequests.find(
      (item) => item.requestId === requestId && item.status === "pending"
    );
    if (!pending) {
      return res.status(404).json({ error: "Pending request not found" });
    }

    const consentId = `cons_${Date.now()}`;
    const expiresAt = new Date(Date.now() + Number(expirySeconds) * 1000).toISOString();

    const tx = await submitApproval({
      consentId,
      requestId,
      recordId,
      researcherWallet,
      approverWallet,
      expiresAt
    });
    const transactionId = tx.transactionId || createTxId();

    pending.status = "approved";
    db.consents.unshift({
      consentId,
      requestId,
      recordId,
      researcherWallet,
      approverWallet,
      expiresAt,
      status: "active"
    });
    commitBlockchainEvent({ txId: transactionId, recordId, action: "approve" });
    pushActivity("approval", "Hospital approved access");

    return res.json({ message: "Consent granted successfully", consentId, transactionId });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Approval failed" });
  }
});

async function retrieveAndSendRecord({ recordId, researcherWallet, role, preview, res }) {
  try {
    if (!recordId || !researcherWallet) {
      return res.status(400).json({ error: "recordId and researcherWallet are required" });
    }

    await queryRecord({ recordId, researcherWallet });
    const record = db.records.find((item) => item.recordId === recordId);
    if (!record) {
      return res.status(404).json({ status: "denied", message: "Record not found" });
    }

    if (role === "researcher") {
      const consent = db.consents.find(
        (item) => item.recordId === recordId && item.researcherWallet === researcherWallet
      );
      if (!consent) {
        return res.status(403).json({ status: "denied", message: "Consent not granted" });
      }

      if (new Date(consent.expiresAt).getTime() < Date.now()) {
        consent.status = "expired";
        pushActivity("expired", "Consent expired");
        return res.status(403).json({ status: "expired", message: "Consent expired" });
      }
      if (consent.status !== "active") {
        return res.status(403).json({ status: "denied", message: "Consent not active" });
      }
    }

    const fileBuffer = await downloadMedicalRecord(record.objectName);
    pushActivity("retrieval", `Researcher retrieved record ${recordId}`);
    const extension = record.extension || path.extname(record.objectName) || ".pdf";
    const contentType = record.contentType || (extension === ".pdf" ? "application/pdf" : "application/octet-stream");
    res.setHeader("Content-Type", contentType);
    res.setHeader(
      "Content-Disposition",
      `${preview ? "inline" : "attachment"}; filename="${recordId}${extension}"`
    );
    commitBlockchainEvent({ txId: createTxId(), recordId, action: "retrieve" });
    return res.send(fileBuffer);
  } catch (err) {
    return res.status(500).json({ error: err.message || "Retrieve failed" });
  }
}

app.get("/retrieveRecord", async (req, res) => {
  return retrieveAndSendRecord({
    recordId: String(req.query.recordId || ""),
    researcherWallet: String(req.query.researcherWallet || ""),
    role: String(req.query.role || "researcher"),
    preview: String(req.query.preview || "") === "true",
    res
  });
});

app.get("/retrieveRecord/:recordId", async (req, res) => {
  return retrieveAndSendRecord({
    recordId: String(req.params.recordId || ""),
    researcherWallet: String(req.query.researcherWallet || "viewer"),
    role: String(req.query.role || "hospital"),
    preview: String(req.query.preview || "") === "true",
    res
  });
});

app.post("/verifyIntegrity", async (req, res) => {
  try {
    const { recordId, patientId } = req.body;
    if (!recordId || !patientId) {
      return res.status(400).json({ error: "recordId and patientId are required" });
    }

    const record = db.records.find((item) => item.recordId === recordId);
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
    const record = db.records.find((item) => item.recordId === recordId);
    if (!record) return res.status(404).json({ error: "Record not found" });
    const fileBuffer = await downloadMedicalRecord(record.objectName);
    const fileHash = computeHash(fileBuffer);
    return res.json({
      blockchainHash: record.fileHash,
      fileHash,
      verified: record.fileHash === fileHash
    });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Verification failed" });
  }
});

app.get("/records", async (_req, res) => {
  try {
    const patientId = String(_req.query.patientId || "");
    const chainRecords = await listRecords();
    const allRecords = chainRecords.records?.length ? chainRecords.records : db.records;
    const records = patientId
      ? allRecords.filter((record) => String(record.patientId) === patientId)
      : allRecords;
    return res.json({ records });
  } catch (err) {
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
    requests: db.accessRequests.filter((item) => item.status === "pending")
  });
});

app.get("/patient/access-requests", (req, res) => {
  const patientId = String(req.query.patientId || "");
  if (!patientId) {
    return res.status(400).json({ error: "patientId is required" });
  }
  return res.json({
    requests: db.accessRequests.filter(
      (item) => item.status === "pending" && item.patientId === patientId
    )
  });
});

app.get("/patient/consents", (req, res) => {
  const patientId = String(req.query.patientId || "");
  if (!patientId) {
    return res.status(400).json({ error: "patientId is required" });
  }
  return res.json({
    consents: db.consents.filter((consent) => {
      const record = db.records.find((item) => item.recordId === consent.recordId);
      return record?.patientId === patientId;
    })
  });
});

app.post("/grant-consent", async (req, res) => {
  try {
    const { requestId, expirySeconds, patientWallet } = req.body;
    if (!requestId || !expirySeconds || !patientWallet) {
      return res.status(400).json({ error: "requestId, expirySeconds, patientWallet are required" });
    }
    const pending = db.accessRequests.find(
      (item) => item.requestId === requestId && item.status === "pending"
    );
    if (!pending) return res.status(404).json({ error: "Pending request not found" });
    const consentId = `cons_${Date.now()}`;
    const expiresAt = new Date(Date.now() + Number(expirySeconds) * 1000).toISOString();
    pending.status = "approved";
    db.consents.unshift({
      consentId,
      requestId,
      recordId: pending.recordId,
      researcherWallet: pending.researcherWallet,
      approverWallet: patientWallet,
      expiresAt,
      status: "active"
    });
    commitBlockchainEvent({
      txId: createTxId(),
      recordId: pending.recordId,
      action: "approve"
    });
    pushActivity("approval", `Patient granted consent for ${pending.recordId}`);
    return res.json({ message: "Consent granted", consentId });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Grant consent failed" });
  }
});

app.post("/reject-consent", async (req, res) => {
  try {
    const { requestId } = req.body;
    if (!requestId) return res.status(400).json({ error: "requestId is required" });
    const pending = db.accessRequests.find(
      (item) => item.requestId === requestId && item.status === "pending"
    );
    if (!pending) return res.status(404).json({ error: "Pending request not found" });
    pending.status = "rejected";
    commitBlockchainEvent({
      txId: createTxId(),
      recordId: pending.recordId,
      action: "reject"
    });
    pushActivity("request", `Patient rejected access for ${pending.recordId}`);
    return res.json({ message: "Request rejected" });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Reject failed" });
  }
});

app.post("/revoke-consent", async (req, res) => {
  try {
    const { consentId } = req.body;
    if (!consentId) return res.status(400).json({ error: "consentId is required" });
    const consent = db.consents.find((item) => item.consentId === consentId);
    if (!consent) return res.status(404).json({ error: "Consent not found" });
    consent.status = "revoked";
    commitBlockchainEvent({
      txId: createTxId(),
      recordId: consent.recordId,
      action: "revoke"
    });
    pushActivity("expired", `Consent revoked for ${consent.recordId}`);
    return res.json({ message: "Consent revoked" });
  } catch (err) {
    return res.status(500).json({ error: err.message || "Revoke failed" });
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
        db.latestTransactions = rows.sort((a, b) =>
          a.timestamp < b.timestamp ? 1 : -1
        );
      }
    } catch (_err) {
      // Ignore log parsing errors and keep current transaction cache.
    }
  }
  res.json(
    db.latestTransactions.sort((a, b) => (a.timestamp < b.timestamp ? 1 : -1))
  );
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

app.listen(PORT, () => {
  console.log(`Backend running on http://localhost:${PORT}`);
});
