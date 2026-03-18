const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const grpc = require("@grpc/grpc-js");
const { connect, hash, signers } = require("@hyperledger/fabric-gateway");
const { exec } = require("child_process");
const { promisify } = require("util");

const execAsync = promisify(exec);
const FABRIC_USE_MOCK = process.env.FABRIC_USE_MOCK === "true";

const FABRIC_CHANNEL = process.env.FABRIC_CHANNEL || "medical-main-channel";
const MEDICAL_CC = process.env.FABRIC_CC_MEDICAL || "medical-records";
const AUTH_CC = process.env.FABRIC_CC_AUTH || "authorization-consent";

const PEER_ENDPOINT = process.env.FABRIC_PEER_ENDPOINT || "localhost:8051";
const PEER_HOST_ALIAS =
  process.env.FABRIC_PEER_HOST_ALIAS || "peer0.hospital1.medical-network.com";
const FABRIC_CRYPTO_PATH =
  process.env.FABRIC_CRYPTO_PATH ||
  path.resolve(__dirname, "../../network/crypto-config");
const FABRIC_IDENTITY_ORG = process.env.FABRIC_IDENTITY_ORG || "hospital1";
const FABRIC_IDENTITY_USER = process.env.FABRIC_IDENTITY_USER || "Admin";

const FABRIC_PEER_CONTAINER =
  process.env.FABRIC_PEER_CONTAINER || "peer0.hospital1.medical-network.com";

function createTxId() {
  return `tx_${crypto.randomUUID().replace(/-/g, "").slice(0, 16)}`;
}

function getOrgDomain(org) {
  const map = {
    hospital1: "hospital1.medical-network.com",
    hospital2: "hospital2.medical-network.com",
    research: "research.medical-network.com",
    regulator: "regulator.medical-network.com"
  };
  return map[org] || org;
}

function getMspId(org) {
  const map = {
    hospital1: "HospitalMSP1",
    hospital2: "HospitalMSP2",
    research: "ResearchOrgMSP",
    regulator: "RegulatorMSP"
  };
  return map[org] || "HospitalMSP1";
}

function loadIdentity(org, user) {
  const domain = getOrgDomain(org);
  const certPath = path.join(
    FABRIC_CRYPTO_PATH,
    "peerOrganizations",
    domain,
    "users",
    `${user}@${domain}`,
    "msp",
    "signcerts",
    `${user}@${domain}-cert.pem`
  );
  return fs.readFileSync(certPath);
}

function loadPrivateKey(org, user) {
  const domain = getOrgDomain(org);
  const keyDir = path.join(
    FABRIC_CRYPTO_PATH,
    "peerOrganizations",
    domain,
    "users",
    `${user}@${domain}`,
    "msp",
    "keystore"
  );
  const keyFile = fs.readdirSync(keyDir).find((file) => file.endsWith("_sk"));
  if (!keyFile) throw new Error(`No private key found in ${keyDir}`);
  return fs.readFileSync(path.join(keyDir, keyFile));
}

function createGrpcConnection(org) {
  const domain = getOrgDomain(org);
  const tlsPath = path.join(
    FABRIC_CRYPTO_PATH,
    "peerOrganizations",
    domain,
    "peers",
    `peer0.${domain}`,
    "tls",
    "ca.crt"
  );
  const tlsRootCert = fs.readFileSync(tlsPath);
  return new grpc.Client(
    PEER_ENDPOINT,
    grpc.credentials.createSsl(tlsRootCert),
    {
      "grpc.ssl_target_name_override": PEER_HOST_ALIAS,
      "grpc.default_authority": PEER_HOST_ALIAS,
      "grpc.max_receive_message_length": -1
    }
  );
}

async function withContract(org, chaincodeName, runner) {
  const grpcClient = createGrpcConnection(org);
  const credentials = loadIdentity(org, FABRIC_IDENTITY_USER);
  const privateKeyPem = loadPrivateKey(org, FABRIC_IDENTITY_USER);
  const privateKey = crypto.createPrivateKey(privateKeyPem);
  const signer = signers.newPrivateKeySigner(privateKey);

  const gateway = connect({
    client: grpcClient,
    identity: {
      mspId: getMspId(org),
      credentials
    },
    signer,
    hash: hash.sha256
  });

  try {
    const network = gateway.getNetwork(FABRIC_CHANNEL);
    const contract = network.getContract(chaincodeName);
    return await runner(contract);
  } finally {
    gateway.close();
    grpcClient.close();
  }
}

function parseMaybeJson(raw) {
  const text = raw?.toString?.() || "";
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch (_err) {
    return text;
  }
}

async function submit(org, chaincode, fn, args = []) {
  if (FABRIC_USE_MOCK) return { transactionId: createTxId(), mock: true };
  return withContract(org, chaincode, async (contract) => {
    const result = await contract.submitTransaction(fn, ...args.map(String));
    return {
      transactionId: createTxId(),
      result: parseMaybeJson(result)
    };
  });
}

async function evaluate(org, chaincode, fn, args = []) {
  if (FABRIC_USE_MOCK) return null;
  return withContract(org, chaincode, async (contract) => {
    const result = await contract.evaluateTransaction(fn, ...args.map(String));
    return parseMaybeJson(result);
  });
}

async function submitUpload(payload) {
  const response = await submit("hospital1", MEDICAL_CC, "CreateUploadRequest", [
    payload.recordID,
    payload.patientID,
    payload.hospitalID,
    payload.fileHash,
    payload.minioPath
  ]);
  return { transactionId: response.transactionId, status: "PENDING_PATIENT_APPROVAL" };
}

async function approveUploadByPatient({ recordId }) {
  const response = await submit("hospital1", MEDICAL_CC, "ApproveUploadByPatient", [recordId]);
  return { transactionId: response.transactionId, status: "APPROVED" };
}

async function submitAccessRequest(payload) {
  const response = await submit("research", AUTH_CC, "RequestAccess", [
    payload.recordId,
    payload.researcherID
  ]);
  return {
    transactionId: response.transactionId,
    requestId: typeof response.result === "string" ? response.result : payload.requestId || ""
  };
}

async function approveByHospital({ requestId }) {
  const response = await submit("hospital1", AUTH_CC, "ApproveByHospital", [requestId]);
  return { transactionId: response.transactionId, status: "PENDING" };
}

async function submitApproval(payload) {
  const response = await submit("hospital1", AUTH_CC, "ApproveByPatient", [
    payload.requestId,
    payload.expiry
  ]);
  return { transactionId: response.transactionId, status: "ACTIVE" };
}

async function revokeAccess(payload) {
  const response = await submit("hospital1", AUTH_CC, "RevokeAccess", [payload.requestId]);
  return { transactionId: response.transactionId, status: "REVOKED" };
}

async function queryRecord(payload) {
  const result = await evaluate("hospital1", MEDICAL_CC, "QueryRecord", [payload.recordId]);
  if (!result) return { exists: false };
  return result;
}

async function listRecords() {
  const records = await evaluate("hospital1", MEDICAL_CC, "QueryRecords", []);
  if (!Array.isArray(records)) return { records: [] };
  return { records };
}

async function queryAccessRequest(payload) {
  const result = await evaluate("hospital1", AUTH_CC, "QueryRequest", [payload.requestId]);
  return result || null;
}

async function listAccessRequests() {
  const result = await evaluate("hospital1", AUTH_CC, "QueryRequests", []);
  return Array.isArray(result) ? result : [];
}

async function getBlockchainInfoFromCli() {
  const output = await execAsync(
    `docker exec ${FABRIC_PEER_CONTAINER} peer channel getinfo -c ${FABRIC_CHANNEL}`,
    { timeout: 20000 }
  );
  const text = (output.stdout || "").trim();
  const heightMatch = text.match(/height:\s*(\d+)/i);
  const currentBlockHashMatch = text.match(/currentBlockHash:\s*([a-fA-F0-9]+)/i);
  const previousBlockHashMatch = text.match(/previousBlockHash:\s*([a-fA-F0-9]+)/i);
  return {
    blockHeight: Number(heightMatch?.[1] || "0"),
    currentBlockHash: currentBlockHashMatch?.[1] || "",
    previousBlockHash: previousBlockHashMatch?.[1] || ""
  };
}

async function getPeerLogs() {
  const output = await execAsync(`docker logs --tail 500 ${FABRIC_PEER_CONTAINER}`, {
    timeout: 20000
  });
  return (output.stdout || "").trim();
}

module.exports = {
  submitUpload,
  approveUploadByPatient,
  submitAccessRequest,
  approveByHospital,
  submitApproval,
  revokeAccess,
  queryRecord,
  listRecords,
  queryAccessRequest,
  listAccessRequests,
  createTxId,
  getBlockchainInfoFromCli,
  getPeerLogs
};
