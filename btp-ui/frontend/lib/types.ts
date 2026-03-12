export type Role = "hospital" | "researcher" | "patient" | "regulator";

export type ActivityItem = {
  id: string;
  type: string;
  message: string;
  timestamp: string;
};

export type UploadResponse = {
  transactionId: string;
  recordId: string;
  fileHash: string;
};

export type PendingRequest = {
  requestId: string;
  recordId: string;
  researcherWallet: string;
  requestedAt: string;
};

export type BlockchainInfo = {
  blockHeight: number;
  currentBlockHash: string;
  previousBlockHash: string;
  transactionCount?: number;
};

export type ExplorerTx = {
  block: number;
  txId: string;
  recordId: string;
  action?: string;
  timestamp: string;
};

export type MinioFile = {
  name: string;
  size: number;
  sizeLabel: string;
  lastModified: string;
};

export type PatientConsent = {
  consentId: string;
  requestId: string;
  recordId: string;
  researcherWallet: string;
  approverWallet: string;
  expiresAt: string;
  status: string;
};
