export type Role = "hospital" | "researcher" | "patient" | "regulator";
export type StatusBadge =
  | "PENDING"
  | "PENDING_HOSPITAL_APPROVAL"
  | "PENDING_PATIENT_APPROVAL"
  | "APPROVED"
  | "ACTIVE"
  | "EXPIRED"
  | "REVOKED"
  | "REJECTED";

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
  patientId?: string;
  hospitalID?: string;
  researcherWallet: string;
  status?: StatusBadge;
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
  patientId?: string;
  researcherWallet: string;
  hospitalID?: string;
  expiry: number;
  status: StatusBadge;
};

export type UploadRequest = {
  requestId: string;
  patientId: string;
  hospitalID: string;
  fileName: string;
  status: StatusBadge;
  createdAt: string;
  recordId?: string | null;
};
