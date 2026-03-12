import {
  ActivityItem,
  BlockchainInfo,
  ExplorerTx,
  MinioFile,
  PendingRequest,
  PatientConsent,
  UploadResponse
} from "@/lib/types";

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:4000";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      ...(init?.headers || {})
    }
  });

  if (!res.ok) {
    let message = "Request failed";
    try {
      const parsed = await res.json();
      message = parsed.error || message;
    } catch (_err) {
      // Ignore JSON parse failures and keep fallback error message.
    }
    throw new Error(message);
  }

  return res.json();
}

export async function uploadRecord(formData: FormData) {
  return request<UploadResponse>("/uploadRecord", {
    method: "POST",
    body: formData
  });
}

export async function requestAccess(payload: {
  recordId: string;
  researcherWallet: string;
}) {
  return request<{ message: string; requestId: string }>("/requestAccess", {
    method: "POST",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" }
  });
}

export async function approveAccess(payload: {
  requestId: string;
  recordId: string;
  researcherWallet: string;
  expirySeconds: number;
  approverWallet: string;
}) {
  return request<{ message: string; consentId: string }>("/approveAccess", {
    method: "POST",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" }
  });
}

export async function retrieveRecord(
  recordId: string,
  researcherWallet: string,
  role: string = "researcher"
) {
  const query = new URLSearchParams({ recordId, researcherWallet, role });
  const res = await fetch(`${API_BASE_URL}/retrieveRecord?${query.toString()}`);
  if (!res.ok) {
    let message = "Retrieve failed";
    try {
      const parsed = await res.json();
      message = parsed.message || parsed.error || message;
    } catch (_err) {
      // Keep fallback message if body is not JSON.
    }
    throw new Error(message);
  }
  const blob = await res.blob();
  const contentDisposition = res.headers.get("content-disposition") || "";
  const filenameMatch = contentDisposition.match(/filename="(.+)"/i);
  return {
    blob,
    fileName: filenameMatch?.[1] || `${recordId}.pdf`
  };
}

export async function retrieveRecordById(
  recordId: string,
  wallet: string,
  role: string,
  preview: boolean = false
) {
  const query = new URLSearchParams({
    researcherWallet: wallet,
    role,
    preview: String(preview)
  });
  const res = await fetch(`${API_BASE_URL}/retrieveRecord/${recordId}?${query.toString()}`);
  if (!res.ok) {
    let message = "Retrieve failed";
    try {
      const parsed = await res.json();
      message = parsed.message || parsed.error || message;
    } catch (_err) {
      // Keep fallback message if body is not JSON.
    }
    throw new Error(message);
  }
  const blob = await res.blob();
  const contentDisposition = res.headers.get("content-disposition") || "";
  const filenameMatch = contentDisposition.match(/filename="(.+)"/i);
  return {
    blob,
    fileName: filenameMatch?.[1] || `${recordId}.pdf`
  };
}

export async function getRecords() {
  return request<{ records: Array<Record<string, string>> }>("/records");
}

export async function getPatientRecords(patientId: string) {
  const query = new URLSearchParams({ patientId });
  return request<{ records: Array<Record<string, string>> }>(`/records?${query.toString()}`);
}

export async function getActivity() {
  return request<{ activity: ActivityItem[] }>("/activity");
}

export async function getPendingRequests() {
  return request<{ requests: PendingRequest[] }>("/pendingRequests");
}

export async function getPatientAccessRequests(patientId: string) {
  const query = new URLSearchParams({ patientId });
  return request<{ requests: PendingRequest[] }>(`/patient/access-requests?${query.toString()}`);
}

export async function getPatientConsents(patientId: string) {
  const query = new URLSearchParams({ patientId });
  return request<{ consents: PatientConsent[] }>(`/patient/consents?${query.toString()}`);
}

export async function verifyIntegrity(payload: {
  recordId: string;
  patientId: string;
}) {
  return request<{
    blockchainHash: string;
    computedHash: string;
    verified: boolean;
  }>("/verifyIntegrity", {
    method: "POST",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" }
  });
}

export async function getMinioFiles() {
  return request<MinioFile[]>("/minio-files");
}

export async function getBlockchainInfo() {
  return request<BlockchainInfo>("/blockchain-info");
}

export async function getLatestTransactions() {
  return request<ExplorerTx[]>("/latest-transactions");
}

export async function getTimeline() {
  return request<{ timeline: ActivityItem[] }>("/timeline");
}

export async function verifyHash(recordId: string) {
  return request<{
    blockchainHash: string;
    fileHash: string;
    verified: boolean;
  }>(`/verify-hash/${recordId}`);
}

export async function grantConsent(payload: {
  requestId: string;
  expirySeconds: number;
  patientWallet: string;
}) {
  return request<{ message: string; consentId: string }>("/grant-consent", {
    method: "POST",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" }
  });
}

export async function rejectConsent(payload: { requestId: string }) {
  return request<{ message: string }>("/reject-consent", {
    method: "POST",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" }
  });
}

export async function revokeConsent(payload: { consentId: string }) {
  return request<{ message: string }>("/revoke-consent", {
    method: "POST",
    body: JSON.stringify(payload),
    headers: { "Content-Type": "application/json" }
  });
}
