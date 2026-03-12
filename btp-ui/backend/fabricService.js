const { exec } = require("child_process");
const crypto = require("crypto");
const { promisify } = require("util");

const execAsync = promisify(exec);

const FABRIC_USE_MOCK = process.env.FABRIC_USE_MOCK !== "false";
const FABRIC_SCRIPT_DIR = process.env.FABRIC_SCRIPT_DIR || "../../network/scripts";
const FABRIC_UPLOAD_CMD =
  process.env.FABRIC_UPLOAD_CMD || `${FABRIC_SCRIPT_DIR}/upload-record.sh`;
const FABRIC_REQUEST_CMD =
  process.env.FABRIC_REQUEST_CMD || `${FABRIC_SCRIPT_DIR}/request-access.sh`;
const FABRIC_APPROVE_CMD =
  process.env.FABRIC_APPROVE_CMD || `${FABRIC_SCRIPT_DIR}/approve-access.sh`;
const FABRIC_QUERY_CMD =
  process.env.FABRIC_QUERY_CMD || `${FABRIC_SCRIPT_DIR}/query-record.sh`;
const FABRIC_LIST_CMD =
  process.env.FABRIC_LIST_CMD || `${FABRIC_SCRIPT_DIR}/list-records.sh`;
const FABRIC_CHANNEL = process.env.FABRIC_CHANNEL || "medical-main-channel";
const FABRIC_PEER_CONTAINER =
  process.env.FABRIC_PEER_CONTAINER || "peer0.hospital1.medical-network.com";

function shellEscape(raw) {
  return `'${String(raw).replace(/'/g, `'\\''`)}'`;
}

function createTxId() {
  return `tx_${crypto.randomUUID().replace(/-/g, "").slice(0, 16)}`;
}

async function runCommand(cmd, payload) {
  const payloadArg = shellEscape(JSON.stringify(payload));
  const full = `bash -lc ${shellEscape(`${cmd} ${payloadArg}`)}`;
  const { stdout, stderr } = await execAsync(full, { timeout: 20000 });
  if (stderr && stderr.trim().length > 0) {
    console.warn("[Fabric CLI warning]", stderr.trim());
  }
  const clean = stdout.trim();
  if (!clean) return {};
  try {
    return JSON.parse(clean);
  } catch (_err) {
    return { raw: clean };
  }
}

async function runRawCommand(command) {
  const full = `bash -lc ${shellEscape(command)}`;
  const { stdout } = await execAsync(full, { timeout: 20000 });
  return (stdout || "").trim();
}

async function submitUpload(payload) {
  if (FABRIC_USE_MOCK) {
    return { transactionId: createTxId(), status: "submitted" };
  }
  return runCommand(FABRIC_UPLOAD_CMD, payload);
}

async function submitAccessRequest(payload) {
  if (FABRIC_USE_MOCK) {
    return { transactionId: createTxId(), status: "requested" };
  }
  return runCommand(FABRIC_REQUEST_CMD, payload);
}

async function submitApproval(payload) {
  if (FABRIC_USE_MOCK) {
    return { transactionId: createTxId(), status: "approved" };
  }
  return runCommand(FABRIC_APPROVE_CMD, payload);
}

async function queryRecord(payload) {
  if (FABRIC_USE_MOCK) return { exists: true };
  return runCommand(FABRIC_QUERY_CMD, payload);
}

async function listRecords() {
  if (FABRIC_USE_MOCK) return { records: [] };
  return runCommand(FABRIC_LIST_CMD, {});
}

async function getBlockchainInfoFromCli() {
  const output = await runRawCommand(
    `docker exec ${FABRIC_PEER_CONTAINER} peer channel getinfo -c ${FABRIC_CHANNEL}`
  );
  const heightMatch = output.match(/height:\s*(\d+)/i);
  const currentBlockHashMatch = output.match(/currentBlockHash:\s*([a-fA-F0-9]+)/i);
  const previousBlockHashMatch = output.match(/previousBlockHash:\s*([a-fA-F0-9]+)/i);
  return {
    blockHeight: Number(heightMatch?.[1] || "0"),
    currentBlockHash: currentBlockHashMatch?.[1] || "",
    previousBlockHash: previousBlockHashMatch?.[1] || ""
  };
}

async function getPeerLogs() {
  return runRawCommand(`docker logs --tail 500 ${FABRIC_PEER_CONTAINER}`);
}

module.exports = {
  submitUpload,
  submitAccessRequest,
  submitApproval,
  queryRecord,
  listRecords,
  createTxId,
  getBlockchainInfoFromCli,
  getPeerLogs
};
