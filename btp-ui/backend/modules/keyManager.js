const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const { saveKey, getKey, updateKey, deleteKey } = require("../data/keyStore");

const KEYS_DIR = path.join(__dirname, "..", "keys");

function sanitizeUserId(user) {
  return String(user || "").replace(/[^a-zA-Z0-9_.-]/g, "_");
}

function ensureKeysDir() {
  if (!fs.existsSync(KEYS_DIR)) {
    fs.mkdirSync(KEYS_DIR, { recursive: true });
  }
}

function getUserKeyPaths(user) {
  const sanitized = sanitizeUserId(user);
  return {
    publicKeyPath: path.join(KEYS_DIR, `${sanitized}.public.pem`),
    privateKeyPath: path.join(KEYS_DIR, `${sanitized}.private.pem`)
  };
}

function ensureUserKeyPair(user) {
  ensureKeysDir();
  const { publicKeyPath, privateKeyPath } = getUserKeyPaths(user);
  const hasPub = fs.existsSync(publicKeyPath);
  const hasPriv = fs.existsSync(privateKeyPath);
  if (hasPub && hasPriv) return { publicKeyPath, privateKeyPath };

  const { publicKey, privateKey } = crypto.generateKeyPairSync("rsa", {
    modulusLength: 2048,
    publicKeyEncoding: { type: "spki", format: "pem" },
    privateKeyEncoding: { type: "pkcs8", format: "pem" }
  });
  fs.writeFileSync(publicKeyPath, publicKey, "utf8");
  fs.writeFileSync(privateKeyPath, privateKey, "utf8");
  return { publicKeyPath, privateKeyPath };
}

function readPublicKey(user) {
  ensureUserKeyPair(user);
  return fs.readFileSync(getUserKeyPaths(user).publicKeyPath, "utf8");
}

function readPrivateKey(user) {
  ensureUserKeyPair(user);
  return fs.readFileSync(getUserKeyPaths(user).privateKeyPath, "utf8");
}

function generateAESKey() {
  return crypto.randomBytes(32);
}

function encryptFile(buffer, key) {
  const iv = crypto.randomBytes(16);
  const cipher = crypto.createCipheriv("aes-256-cbc", key, iv);
  const encrypted = Buffer.concat([cipher.update(buffer), cipher.final()]);
  return { encrypted, iv };
}

function decryptFile(encryptedBuffer, key, iv) {
  const decipher = crypto.createDecipheriv("aes-256-cbc", key, iv);
  return Buffer.concat([decipher.update(encryptedBuffer), decipher.final()]);
}

function encryptKeyWithPublicKey(aesKey, publicKey) {
  return crypto.publicEncrypt(publicKey, aesKey);
}

function decryptKeyWithPrivateKey(encKey, privateKey) {
  return crypto.privateDecrypt(privateKey, encKey);
}

function validateKey(recordId, user) {
  const keyEntry = getKey(recordId, user);
  if (!keyEntry) {
    return { valid: false, reason: "KEY_NOT_FOUND", keyEntry: null };
  }
  if (!keyEntry.valid) {
    return { valid: false, reason: "KEY_REVOKED", keyEntry };
  }
  if (keyEntry.expiry && Date.now() > Number(keyEntry.expiry)) {
    updateKey(recordId, user, { valid: false });
    return { valid: false, reason: "KEY_EXPIRED", keyEntry: { ...keyEntry, valid: false } };
  }
  return { valid: true, reason: "OK", keyEntry };
}

function revokeKey(recordId, user, hardDelete = false) {
  if (hardDelete) {
    return deleteKey(recordId, user);
  }
  return Boolean(updateKey(recordId, user, { valid: false }));
}

function reEncryptKeyForUser({
  recordId,
  fromUser,
  toUser,
  expiry = null
}) {
  const sourceKey = getKey(recordId, fromUser);
  if (!sourceKey || !sourceKey.valid) {
    throw new Error("Source key is unavailable or invalid");
  }

  const fromPrivateKey = readPrivateKey(fromUser);
  const decryptedAes = decryptKeyWithPrivateKey(
    Buffer.from(sourceKey.encryptedKey, "base64"),
    fromPrivateKey
  );
  const toPublicKey = readPublicKey(toUser);
  const reEncrypted = encryptKeyWithPublicKey(decryptedAes, toPublicKey);

  return saveKey({
    recordId,
    user: toUser,
    encryptedKey: reEncrypted.toString("base64"),
    expiry,
    valid: true
  });
}

module.exports = {
  KEYS_DIR,
  ensureUserKeyPair,
  readPublicKey,
  readPrivateKey,
  generateAESKey,
  encryptFile,
  decryptFile,
  encryptKeyWithPublicKey,
  decryptKeyWithPrivateKey,
  reEncryptKeyForUser,
  validateKey,
  revokeKey,
  saveKey,
  getKey,
  updateKey,
  deleteKey
};
