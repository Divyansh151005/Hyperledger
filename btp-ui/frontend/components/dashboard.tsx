"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import {
  Blocks,
  CheckCircle2,
  Database,
  FileUp,
  FolderOpen,
  GitBranch,
  Loader2,
  SearchCheck,
  ShieldCheck,
  ShieldEllipsis,
  TableProperties,
  UserCheck,
  UserRound
} from "lucide-react";
import {
  approveAccess,
  approveAccessByHospital,
  approveUploadByPatient,
  getPreviewRecordUrl,
  getActivity,
  getBlockchainInfo,
  getLatestTransactions,
  getMinioFiles,
  getPatientAccessRequests,
  getPatientConsents,
  getPatientUploadRequests,
  getPatientRecords,
  getPendingRequests,
  getResearcherAccess,
  getRecords,
  getTimeline,
  requestAccess,
  rejectAccessByPatient,
  rejectUploadByPatient,
  retrieveRecord,
  retrieveRecordById,
  revokeConsent,
  uploadRecord,
  verifyHash
} from "@/lib/api";
import {
  ActivityItem,
  BlockchainInfo,
  ExplorerTx,
  MinioFile,
  PatientConsent,
  PendingRequest,
  Role,
  UploadRequest,
  StatusBadge
} from "@/lib/types";
import { truncateAddress } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ActivityFeed } from "@/components/activity-feed";
import { PDFPreviewModal } from "@/components/pdf-preview-modal";

type DashboardProps = {
  walletAddress: string;
  role: Role;
  onLogout: () => void;
  onDisconnect: () => void;
};

type PanelType =
  | "upload"
  | "request"
  | "approve"
  | "retrieve"
  | "patient"
  | "minio"
  | "explorer";

const panels: Array<{
  id: PanelType;
  title: string;
  description: string;
  icon: React.ComponentType<{ className?: string }>;
}> = [
  { id: "upload", title: "Upload Record", description: "Hospital upload flow", icon: FileUp },
  { id: "request", title: "Request Access", description: "Research access flow", icon: UserCheck },
  { id: "approve", title: "Approve Access", description: "Hospital consent approval", icon: ShieldCheck },
  {
    id: "retrieve",
    title: "Retrieve Record",
    description: "Consent-gated PDF download",
    icon: SearchCheck
  },
  {
    id: "patient",
    title: "Patient Dashboard",
    description: "Patient-owned consent control",
    icon: UserRound
  },
  {
    id: "minio",
    title: "Off-Chain Storage (MinIO)",
    description: "Stored reports and preview",
    icon: FolderOpen
  },
  {
    id: "explorer",
    title: "Fabric Explorer",
    description: "Recent Fabric transactions",
    icon: TableProperties
  }
];

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function downloadBlob(name: string, blob: Blob) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = name;
  anchor.click();
  URL.revokeObjectURL(url);
}

function openBlobInBrowser(blob: Blob) {
  const url = URL.createObjectURL(blob);
  window.open(url, "_blank");
  setTimeout(() => URL.revokeObjectURL(url), 2000);
}

function shortHash(raw: string) {
  if (!raw) return "-";
  return `${raw.slice(0, 12)}...${raw.slice(-8)}`;
}

function statusBadgeClass(status?: string) {
  const normalized = String(status || "").toUpperCase() as StatusBadge;
  switch (normalized) {
    case "PENDING":
    case "PENDING_HOSPITAL_APPROVAL":
    case "PENDING_PATIENT_APPROVAL":
      return "bg-yellow-50 text-yellow-700";
    case "APPROVED":
    case "ACTIVE":
      return "bg-emerald-50 text-emerald-700";
    case "EXPIRED":
      return "bg-orange-50 text-orange-700";
    case "REVOKED":
      return "bg-red-50 text-red-700";
    case "REJECTED":
      return "bg-slate-100 text-slate-700";
    default:
      return "bg-slate-100 text-slate-700";
  }
}

export function Dashboard({ walletAddress, role, onLogout, onDisconnect }: DashboardProps) {
  const [activePanel, setActivePanel] = useState<PanelType>("upload");
  const [activity, setActivity] = useState<ActivityItem[]>([]);
  const [timeline, setTimeline] = useState<ActivityItem[]>([]);
  const [pendingRequests, setPendingRequests] = useState<PendingRequest[]>([]);
  const [records, setRecords] = useState<Array<Record<string, string>>>([]);
  const [minioFiles, setMinioFiles] = useState<MinioFile[]>([]);
  const [blockchainInfo, setBlockchainInfo] = useState<BlockchainInfo>({
    blockHeight: 0,
    currentBlockHash: "",
    previousBlockHash: "",
    transactionCount: 0
  });
  const [latestTransactions, setLatestTransactions] = useState<ExplorerTx[]>([]);
  const [busy, setBusy] = useState<string>("");
  const [status, setStatus] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [txSteps, setTxSteps] = useState<string[]>([]);

  const [patientId, setPatientId] = useState("");
  const [uploadFileState, setUploadFileState] = useState<File | null>(null);
  const [uploadResult, setUploadResult] = useState<{
    requestId: string;
    status: string;
  } | null>(null);
  const uploadFileInputRef = useRef<HTMLInputElement>(null);

  const [requestRecordId, setRequestRecordId] = useState("");
  const [retrieveRecordId, setRetrieveRecordId] = useState("");
  const [integrityRecordId, setIntegrityRecordId] = useState("");
  const [integrityResult, setIntegrityResult] = useState<{
    blockchainHash: string;
    fileHash: string;
    verified: boolean;
  } | null>(null);

  const [approveModalOpen, setApproveModalOpen] = useState(false);
  const [selectedRequest, setSelectedRequest] = useState<PendingRequest | null>(null);
  const [expirySeconds, setExpirySeconds] = useState(3600);
  const [previewModalOpen, setPreviewModalOpen] = useState(false);
  const [previewRecordId, setPreviewRecordId] = useState("");
  const [previewUrl, setPreviewUrl] = useState("");
  const [researcherAccessInfo, setResearcherAccessInfo] = useState<{
    status: string;
    expiresAt: number;
    remainingMs: number;
  } | null>(null);

  const [patientRecords, setPatientRecords] = useState<Array<Record<string, string>>>([]);
  const [patientUploadRequests, setPatientUploadRequests] = useState<UploadRequest[]>([]);
  const [patientRequests, setPatientRequests] = useState<PendingRequest[]>([]);
  const [patientConsents, setPatientConsents] = useState<PatientConsent[]>([]);

  const isHospital = role === "hospital";
  const isResearcher = role === "researcher";
  const isPatient = role === "patient";
  const isRegulator = role === "regulator";

  async function runTxAnimation() {
    setTxSteps(["Submitting transaction..."]);
    await sleep(350);
    setTxSteps(["Submitting transaction...", "Waiting for endorsement..."]);
    await sleep(350);
    setTxSteps([
      "Submitting transaction...",
      "Waiting for endorsement...",
      "Transaction committed"
    ]);
  }

  async function refreshData() {
    try {
      const [activityRes, pendingRes, recordsRes, minioRes, infoRes, txRes, timelineRes] =
        await Promise.all([
          getActivity(),
          getPendingRequests(),
          getRecords(),
          getMinioFiles(),
          getBlockchainInfo(),
          getLatestTransactions(),
          getTimeline()
        ]);
      setActivity(activityRes.activity || []);
      setPendingRequests(pendingRes.requests || []);
      setRecords(recordsRes.records || []);
      setMinioFiles(minioRes || []);
      setBlockchainInfo(infoRes);
      setLatestTransactions(txRes || []);
      setTimeline(timelineRes.timeline || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to refresh dashboard");
    }
  }

  async function refreshPatientData() {
    if (!patientId) return;
    try {
      const [recordsRes, uploadRequestsRes, requestsRes, consentsRes] = await Promise.all([
        getPatientRecords(patientId),
        getPatientUploadRequests(patientId),
        getPatientAccessRequests(patientId),
        getPatientConsents(patientId)
      ]);
      setPatientRecords(recordsRes.records || []);
      setPatientUploadRequests(uploadRequestsRes.requests || []);
      setPatientRequests(requestsRes.requests || []);
      setPatientConsents(consentsRes.consents || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load patient dashboard");
    }
  }

  useEffect(() => {
    refreshData();
    const timer = setInterval(refreshData, 3000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => {
    const eventSource = new EventSource(
      `${process.env.NEXT_PUBLIC_API_URL || "http://localhost:4000"}/events`
    );
    eventSource.onmessage = () => {
      refreshData();
    };
    eventSource.onerror = () => {
      eventSource.close();
    };
    return () => eventSource.close();
  }, []);

  useEffect(() => {
    refreshPatientData();
  }, [patientId]);

  const visiblePanels = useMemo(
    () =>
      panels.filter((panel) => {
        if (panel.id === "upload" || panel.id === "approve") return isHospital;
        if (panel.id === "request") return isResearcher;
        if (panel.id === "retrieve") return isResearcher || isHospital;
        if (panel.id === "patient") return isPatient;
        return true;
      }),
    [isHospital, isResearcher, isPatient]
  );

  useEffect(() => {
    if (!visiblePanels.find((item) => item.id === activePanel)) {
      setActivePanel(visiblePanels[0]?.id || "upload");
    }
  }, [activePanel, visiblePanels]);

  async function handleUploadRecord() {
    if (!uploadFileState || !patientId) return;
    setError("");
    setStatus("");
    setBusy("upload");
    setUploadResult(null);
    await runTxAnimation();
    try {
      const formData = new FormData();
      formData.append("patientId", patientId);
      formData.append("hospitalID", walletAddress);
      formData.append("file", uploadFileState);
      const response = await uploadRecord(formData);
      setUploadResult(response);
      setStatus("Upload request submitted. Waiting for patient approval.");
      refreshData();
      refreshPatientData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setBusy("");
    }
  }

  async function handleChooseUploadFile() {
    try {
      const pickerWindow = window as Window & {
        showOpenFilePicker?: (options?: {
          multiple?: boolean;
          excludeAcceptAllOption?: boolean;
          types?: Array<{ description?: string; accept: Record<string, string[]> }>;
        }) => Promise<Array<{ getFile: () => Promise<File> }>>;
      };

      if (pickerWindow.showOpenFilePicker) {
        const [handle] = await pickerWindow.showOpenFilePicker({
          multiple: false,
          excludeAcceptAllOption: false,
          types: [{ description: "PDF files", accept: { "application/pdf": [".pdf"] } }]
        });
        if (handle) {
          const file = await handle.getFile();
          setUploadFileState(file);
          return;
        }
      }
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") return;
    }

    uploadFileInputRef.current?.click();
  }

  async function handleRequestAccess() {
    if (!requestRecordId) return;
    setBusy("request");
    setStatus("");
    setError("");
    await runTxAnimation();
    try {
      const response = await requestAccess({
        recordId: requestRecordId,
        researcherWallet: walletAddress
      });
      setStatus(`${response.message}. Waiting for approval.`);
      refreshData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setBusy("");
    }
  }

  async function handleApproveAccess() {
    if (!selectedRequest) return;
    setBusy("approve");
    setStatus("");
    setError("");
    await runTxAnimation();
    try {
      const response = await approveAccessByHospital({
        requestId: selectedRequest.requestId,
        hospitalWallet: walletAddress
      });
      setStatus(`Hospital approval complete (${response.status}). Awaiting patient approval.`);
      setApproveModalOpen(false);
      setSelectedRequest(null);
      refreshData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Approval failed");
    } finally {
      setBusy("");
    }
  }

  async function handleRetrieveRecord() {
    if (!retrieveRecordId) return;
    if (isResearcher) {
      setBusy("retrieve");
      setError("");
      setStatus("Checking consent + key validity...");
      try {
        const access = await checkResearcherAccess(retrieveRecordId);
        setStatus(`Access ${access.status}. Opening secure preview...`);
        setPreviewRecordId(retrieveRecordId);
        setPreviewUrl(getPreviewRecordUrl(retrieveRecordId, walletAddress, role));
        setPreviewModalOpen(true);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Access denied");
      } finally {
        setBusy("");
      }
      return;
    }

    setBusy("retrieve");
    setStatus("");
    setError("");
    await runTxAnimation();
    try {
      const response = await retrieveRecord(retrieveRecordId, walletAddress, role);
      downloadBlob(response.fileName, response.blob);
      setStatus("Consent valid. PDF downloaded.");
      refreshData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Retrieve failed");
    } finally {
      setBusy("");
    }
  }

  async function handleVerifyIntegrity() {
    if (!integrityRecordId) return;
    setBusy("integrity");
    setStatus("");
    setError("");
    try {
      const response = await verifyHash(integrityRecordId);
      setIntegrityResult(response);
      setStatus(response.verified ? "Integrity verified." : "Tampering detected.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Verification failed");
    } finally {
      setBusy("");
    }
  }

  async function handleViewMinioFile(fileName: string, preview: boolean) {
    const rawName = fileName.split("/").pop() || fileName;
    const recordId = rawName.replace(/\.[^/.]+$/, "");
    setBusy(`view-${recordId}`);
    setStatus("");
    setError("");
    try {
      if (isResearcher) {
        await checkResearcherAccess(recordId);
        setPreviewRecordId(recordId);
        setPreviewUrl(getPreviewRecordUrl(recordId, walletAddress, role));
        setPreviewModalOpen(true);
        setStatus(`Previewing ${recordId} in secure mode`);
        return;
      }
      const result = await retrieveRecordById(recordId, walletAddress, role, preview);
      if (preview) openBlobInBrowser(result.blob);
      else downloadBlob(result.fileName, result.blob);
      setStatus(preview ? `Previewing ${result.fileName}` : `Opened ${result.fileName}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to open MinIO file");
    } finally {
      setBusy("");
    }
  }

  async function handlePatientApproveUpload(requestId: string) {
    setBusy(`approve-upload-${requestId}`);
    setStatus("");
    setError("");
    await runTxAnimation();
    try {
      const response = await approveUploadByPatient({ requestId, patientId });
      setStatus(
        `Upload approved. Record ${response.recordId} encrypted and stored at ${response.minioPath}.`
      );
      refreshData();
      refreshPatientData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload approval failed");
    } finally {
      setBusy("");
    }
  }

  async function handlePatientRejectUpload(requestId: string) {
    setBusy(`reject-upload-${requestId}`);
    setStatus("");
    setError("");
    try {
      await rejectUploadByPatient({ requestId, patientId });
      setStatus("Upload request rejected by patient.");
      refreshData();
      refreshPatientData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload rejection failed");
    } finally {
      setBusy("");
    }
  }

  async function handlePatientGrant(requestId: string) {
    setBusy(`grant-${requestId}`);
    setStatus("");
    setError("");
    await runTxAnimation();
    try {
      const response = await approveAccess({
        requestId,
        patientId,
        expirySeconds
      });
      setStatus(`Consent granted by patient. Status: ${response.status}`);
      refreshData();
      refreshPatientData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Grant failed");
    } finally {
      setBusy("");
    }
  }

  async function handlePatientReject(requestId: string) {
    setBusy(`reject-${requestId}`);
    setStatus("");
    setError("");
    try {
      await rejectAccessByPatient({ requestId, patientId });
      setStatus("Request rejected.");
      refreshData();
      refreshPatientData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Reject failed");
    } finally {
      setBusy("");
    }
  }

  async function handlePatientRevoke(consentId: string) {
    setBusy(`revoke-${consentId}`);
    setStatus("");
    setError("");
    try {
      await revokeConsent({ consentId });
      setStatus("Consent revoked.");
      refreshData();
      refreshPatientData();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Revoke failed");
    } finally {
      setBusy("");
    }
  }

  async function checkResearcherAccess(recordId: string) {
    try {
      const result = await getResearcherAccess(recordId, walletAddress);
      setResearcherAccessInfo(result);
      return result;
    } catch (err) {
      setResearcherAccessInfo(null);
      throw err;
    }
  }

  const latestBlockTxCount = useMemo(
    () =>
      blockchainInfo.transactionCount ||
      latestTransactions.filter((tx) => tx.block === blockchainInfo.blockHeight).length,
    [latestTransactions, blockchainInfo]
  );

  return (
    <div className="h-screen overflow-hidden bg-background p-4">
      <div className="mx-auto grid h-full max-w-[1600px] grid-cols-[260px_1fr_340px] gap-4">
        <Card className="h-full overflow-y-auto p-4">
          <p className="mb-1 text-xs font-semibold tracking-[0.2em] text-primary">FABRIC DAPP</p>
          <h2 className="text-lg font-semibold text-slate-900">Healthcare Dashboard</h2>
          <p className="mt-1 text-xs text-slate-500">Wallet: {truncateAddress(walletAddress)}</p>
          <p className="text-xs text-slate-500">Role: {role}</p>
          <div className="mt-3 flex gap-2">
            <Button size="sm" variant="outline" onClick={onDisconnect}>
              Switch Role
            </Button>
            <Button size="sm" variant="outline" onClick={onLogout}>
              Back to Home
            </Button>
          </div>
          <div className="mt-5 space-y-2">
            {visiblePanels.map((panel) => {
              const Icon = panel.icon;
              const active = panel.id === activePanel;
              return (
                <motion.button
                  key={panel.id}
                  whileHover={{ x: 2 }}
                  onClick={() => setActivePanel(panel.id)}
                  className={`flex w-full items-start gap-2 rounded-xl border px-3 py-2 text-left ${
                    active
                      ? "border-primary/40 bg-blue-50 text-slate-900"
                      : "border-slate-200 bg-white text-slate-600"
                  }`}
                >
                  <Icon className="mt-0.5 h-4 w-4" />
                  <span className="text-sm">{panel.title}</span>
                </motion.button>
              );
            })}
          </div>
        </Card>

        <div className="h-full overflow-y-auto pr-1">
          <div className="space-y-4 pb-4">
            <div className="grid gap-4 lg:grid-cols-3">
              <Card className="p-4">
                <div className="mb-2 flex items-center gap-2 text-slate-700">
                  <GitBranch className="h-4 w-4 text-primary" />
                  <p className="text-sm font-medium">Live Blockchain</p>
                </div>
                <p className="text-xs text-slate-500">Block Height</p>
                <motion.p
                  key={blockchainInfo.blockHeight}
                  initial={{ opacity: 0.5, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  className="text-2xl font-semibold text-slate-900"
                >
                  {blockchainInfo.blockHeight}
                </motion.p>
                <p className="mt-2 text-xs text-slate-500">
                  Latest Hash: {shortHash(blockchainInfo.currentBlockHash)}
                </p>
                <p className="text-xs text-slate-500">Tx Count: {latestBlockTxCount}</p>
              </Card>

              <Card className="p-4">
                <div className="mb-2 flex items-center gap-2 text-slate-700">
                  <Blocks className="h-4 w-4 text-primary" />
                  <p className="text-sm font-medium">Latest Block</p>
                </div>
                <p className="text-2xl font-semibold text-slate-900">
                  Block {blockchainInfo.blockHeight}
                </p>
                <p className="mt-2 text-xs text-slate-500">
                  Prev Hash: {shortHash(blockchainInfo.previousBlockHash)}
                </p>
                <p className="text-xs text-slate-500">
                  Curr Hash: {shortHash(blockchainInfo.currentBlockHash)}
                </p>
              </Card>

              <Card className="p-4">
                <div className="mb-2 flex items-center gap-2 text-slate-700">
                  <Database className="h-4 w-4 text-primary" />
                  <p className="text-sm font-medium">Ledger Proof</p>
                </div>
                <p className="text-xs text-slate-500">Transactions tracked</p>
                <p className="text-2xl font-semibold text-slate-900">{latestTransactions.length}</p>
                <p className="mt-2 text-xs text-slate-500">Off-chain files stored</p>
                <p className="text-xl font-semibold text-slate-900">{minioFiles.length}</p>
              </Card>
            </div>

            <Card className="p-6">
              <AnimatePresence mode="wait">
                {activePanel === "upload" ? (
                  <motion.div
                    key="upload"
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    className="space-y-4"
                  >
                    <h3 className="text-xl font-semibold text-slate-900">Upload Medical Record</h3>
                    <Input
                      placeholder="Patient ID"
                      value={patientId}
                      onChange={(event) => setPatientId(event.target.value)}
                    />
                    <div className="flex flex-wrap items-center gap-3">
                      <Button type="button" variant="outline" onClick={handleChooseUploadFile}>
                        Choose File
                      </Button>
                      <input
                        ref={uploadFileInputRef}
                        type="file"
                        accept=".pdf,application/pdf"
                        className="hidden"
                        onChange={(event) => setUploadFileState(event.target.files?.[0] || null)}
                      />
                      <span className="text-sm text-slate-500">
                        {uploadFileState ? uploadFileState.name : "No file selected"}
                      </span>
                    </div>
                    <Button
                      disabled={busy === "upload" || !uploadFileState || !patientId}
                      onClick={handleUploadRecord}
                    >
                      {busy === "upload" ? (
                        <span className="inline-flex items-center gap-2">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          Submitting...
                        </span>
                      ) : (
                        "Submit"
                      )}
                    </Button>
                    {uploadResult ? (
                      <div className="rounded-lg border border-emerald-100 bg-emerald-50 p-4 text-sm text-emerald-700">
                        <div className="mb-2 flex items-center gap-2">
                          <CheckCircle2 className="h-4 w-4" />
                          <span>Upload request created</span>
                        </div>
                        <p>Request ID: {uploadResult.requestId}</p>
                        <p>
                          Status:{" "}
                          <Badge className={statusBadgeClass(uploadResult.status)}>
                            {uploadResult.status}
                          </Badge>
                        </p>
                      </div>
                    ) : null}
                  </motion.div>
                ) : null}

                {activePanel === "request" ? (
                  <motion.div
                    key="request"
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    className="space-y-4"
                  >
                    <h3 className="text-xl font-semibold text-slate-900">Request Record Access</h3>
                    <Input
                      placeholder="Enter Record ID"
                      value={requestRecordId}
                      onChange={(event) => setRequestRecordId(event.target.value)}
                    />
                    <Button
                      onClick={handleRequestAccess}
                      disabled={busy === "request" || !requestRecordId}
                    >
                      {busy === "request" ? "Submitting..." : "Request Access"}
                    </Button>
                  </motion.div>
                ) : null}

                {activePanel === "approve" ? (
                  <motion.div
                    key="approve"
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    className="space-y-4"
                  >
                    <h3 className="text-xl font-semibold text-slate-900">Approve Access Requests</h3>
                    <div className="overflow-hidden rounded-xl border border-slate-200">
                      <table className="w-full text-sm">
                        <thead className="bg-slate-50 text-left text-slate-500">
                          <tr>
                            <th className="px-3 py-2">Record ID</th>
                            <th className="px-3 py-2">Researcher Wallet</th>
                            <th className="px-3 py-2">Status</th>
                            <th className="px-3 py-2">Action</th>
                          </tr>
                        </thead>
                        <tbody>
                          {pendingRequests.length === 0 ? (
                            <tr>
                              <td className="px-3 py-3 text-slate-500" colSpan={4}>
                                No pending requests.
                              </td>
                            </tr>
                          ) : (
                            pendingRequests.map((item) => (
                              <tr key={item.requestId} className="border-t border-slate-100">
                                <td className="px-3 py-2">{item.recordId}</td>
                                <td className="px-3 py-2">{truncateAddress(item.researcherWallet)}</td>
                                <td className="px-3 py-2">
                                  <Badge className={statusBadgeClass(item.status)}>
                                    {item.status || "PENDING"}
                                  </Badge>
                                </td>
                                <td className="px-3 py-2">
                                  <Button
                                    size="sm"
                                    onClick={() => {
                                      setSelectedRequest(item);
                                      setApproveModalOpen(true);
                                    }}
                                  >
                                    Approve
                                  </Button>
                                </td>
                              </tr>
                            ))
                          )}
                        </tbody>
                      </table>
                    </div>
                  </motion.div>
                ) : null}

                {activePanel === "retrieve" ? (
                  <motion.div
                    key="retrieve"
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    className="space-y-4"
                  >
                    <h3 className="text-xl font-semibold text-slate-900">Retrieve Medical Record</h3>
                    <Input
                      placeholder="Enter Record ID"
                      value={retrieveRecordId}
                      onChange={(event) => setRetrieveRecordId(event.target.value)}
                    />
                    <Button
                      onClick={handleRetrieveRecord}
                      disabled={busy === "retrieve" || !retrieveRecordId}
                    >
                      {isResearcher
                        ? "Preview Record"
                        : busy === "retrieve"
                          ? "Retrieving..."
                          : "Retrieve PDF"}
                    </Button>
                    {isResearcher ? (
                      <div className="rounded-lg border border-blue-100 bg-blue-50 p-3 text-sm text-blue-700">
                        <p>Preview-only mode enabled (no downloads).</p>
                        {researcherAccessInfo ? (
                          <p>
                            Access expiry countdown:{" "}
                            {Math.max(0, Math.floor(researcherAccessInfo.remainingMs / 1000))}s
                          </p>
                        ) : (
                          <p>Access check will run before opening preview.</p>
                        )}
                      </div>
                    ) : null}
                    <div className="space-y-2 rounded-lg bg-slate-50 p-4">
                      <p className="text-sm font-medium text-slate-700">Verify Integrity</p>
                      <Input
                        placeholder="Record ID"
                        value={integrityRecordId}
                        onChange={(event) => setIntegrityRecordId(event.target.value)}
                      />
                      <Button
                        variant="outline"
                        disabled={busy === "integrity" || !integrityRecordId}
                        onClick={handleVerifyIntegrity}
                      >
                        {busy === "integrity" ? "Verifying..." : "Verify Integrity"}
                      </Button>
                    </div>
                    {integrityResult ? (
                      <div
                        className={`rounded-lg p-4 text-sm ${
                          integrityResult.verified
                            ? "bg-emerald-50 text-emerald-700"
                            : "bg-red-50 text-red-700"
                        }`}
                      >
                        <p className="break-all">Blockchain Hash: {integrityResult.blockchainHash}</p>
                        <p className="break-all">File Hash: {integrityResult.fileHash}</p>
                        <p className="font-semibold">
                          {integrityResult.verified
                            ? "✔ Integrity verified"
                            : "⚠ Tampering detected"}
                        </p>
                      </div>
                    ) : null}
                  </motion.div>
                ) : null}

                {activePanel === "patient" ? (
                  <motion.div
                    key="patient"
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    className="space-y-5"
                  >
                    <h3 className="text-xl font-semibold text-slate-900">Patient Dashboard</h3>
                    <div className="flex gap-2">
                      <Input
                        placeholder="Enter patient ID"
                        value={patientId}
                        onChange={(event) => setPatientId(event.target.value)}
                      />
                      <Button variant="outline" onClick={refreshPatientData} disabled={!patientId}>
                        Load
                      </Button>
                    </div>

                    <div>
                      <p className="mb-2 text-sm font-medium text-slate-700">My Medical Records</p>
                      <div className="overflow-hidden rounded-xl border border-slate-200">
                        <table className="w-full text-sm">
                          <thead className="bg-slate-50 text-left text-slate-500">
                            <tr>
                              <th className="px-3 py-2">Record ID</th>
                              <th className="px-3 py-2">Hospital</th>
                              <th className="px-3 py-2">Date</th>
                            </tr>
                          </thead>
                          <tbody>
                            {patientRecords.map((record) => (
                              <tr key={record.recordId} className="border-t border-slate-100">
                                <td className="px-3 py-2">{record.recordId}</td>
                                <td className="px-3 py-2">{truncateAddress(record.walletAddress)}</td>
                                <td className="px-3 py-2">
                                  {new Date(record.timestamp).toLocaleDateString()}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>

                    <div>
                      <p className="mb-2 text-sm font-medium text-slate-700">Upload Approval Requests</p>
                      <div className="overflow-hidden rounded-xl border border-slate-200">
                        <table className="w-full text-sm">
                          <thead className="bg-slate-50 text-left text-slate-500">
                            <tr>
                              <th className="px-3 py-2">Request</th>
                              <th className="px-3 py-2">Hospital</th>
                              <th className="px-3 py-2">File</th>
                              <th className="px-3 py-2">Status</th>
                              <th className="px-3 py-2">Approve</th>
                              <th className="px-3 py-2">Reject</th>
                            </tr>
                          </thead>
                          <tbody>
                            {patientUploadRequests.length === 0 ? (
                              <tr>
                                <td className="px-3 py-3 text-slate-500" colSpan={6}>
                                  No upload approvals pending.
                                </td>
                              </tr>
                            ) : (
                              patientUploadRequests.map((request) => (
                                <tr key={request.requestId} className="border-t border-slate-100">
                                  <td className="px-3 py-2">{request.requestId}</td>
                                  <td className="px-3 py-2">{truncateAddress(request.hospitalID)}</td>
                                  <td className="px-3 py-2">{request.fileName}</td>
                                  <td className="px-3 py-2">
                                    <Badge className={statusBadgeClass(request.status)}>
                                      {request.status}
                                    </Badge>
                                  </td>
                                  <td className="px-3 py-2">
                                    <Button
                                      size="sm"
                                      onClick={() => handlePatientApproveUpload(request.requestId)}
                                      disabled={
                                        request.status !== "PENDING_PATIENT_APPROVAL" ||
                                        busy === `approve-upload-${request.requestId}`
                                      }
                                    >
                                      Approve
                                    </Button>
                                  </td>
                                  <td className="px-3 py-2">
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      onClick={() => handlePatientRejectUpload(request.requestId)}
                                      disabled={
                                        request.status !== "PENDING_PATIENT_APPROVAL" ||
                                        busy === `reject-upload-${request.requestId}`
                                      }
                                    >
                                      Reject
                                    </Button>
                                  </td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>
                    </div>

                    <div>
                      <p className="mb-2 text-sm font-medium text-slate-700">Access Requests</p>
                      <div className="overflow-hidden rounded-xl border border-slate-200">
                        <table className="w-full text-sm">
                          <thead className="bg-slate-50 text-left text-slate-500">
                            <tr>
                              <th className="px-3 py-2">Researcher Wallet</th>
                              <th className="px-3 py-2">Record</th>
                              <th className="px-3 py-2">Status</th>
                              <th className="px-3 py-2">Approve</th>
                              <th className="px-3 py-2">Reject</th>
                            </tr>
                          </thead>
                          <tbody>
                            {patientRequests.map((request) => (
                              <tr key={request.requestId} className="border-t border-slate-100">
                                <td className="px-3 py-2">{truncateAddress(request.researcherWallet)}</td>
                                <td className="px-3 py-2">{request.recordId}</td>
                                <td className="px-3 py-2">
                                  <Badge className={statusBadgeClass(request.status)}>
                                    {request.status || "PENDING"}
                                  </Badge>
                                </td>
                                <td className="px-3 py-2">
                                  <Button
                                    size="sm"
                                    onClick={() => handlePatientGrant(request.requestId)}
                                    disabled={busy === `grant-${request.requestId}`}
                                  >
                                    Approve
                                  </Button>
                                </td>
                                <td className="px-3 py-2">
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => handlePatientReject(request.requestId)}
                                    disabled={busy === `reject-${request.requestId}`}
                                  >
                                    Reject
                                  </Button>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>

                    <div>
                      <p className="mb-2 text-sm font-medium text-slate-700">Active Consents</p>
                      <div className="overflow-hidden rounded-xl border border-slate-200">
                        <table className="w-full text-sm">
                          <thead className="bg-slate-50 text-left text-slate-500">
                            <tr>
                              <th className="px-3 py-2">Record</th>
                              <th className="px-3 py-2">Researcher</th>
                              <th className="px-3 py-2">Expiry</th>
                              <th className="px-3 py-2">Status</th>
                              <th className="px-3 py-2">Revoke</th>
                            </tr>
                          </thead>
                          <tbody>
                            {patientConsents.map((consent) => (
                              <tr key={consent.consentId} className="border-t border-slate-100">
                                <td className="px-3 py-2">{consent.recordId}</td>
                                <td className="px-3 py-2">{truncateAddress(consent.researcherWallet)}</td>
                                <td className="px-3 py-2">
                                  {new Date(consent.expiry).toLocaleString()}
                                </td>
                                <td className="px-3 py-2">
                                  <Badge className={statusBadgeClass(consent.status)}>
                                    {consent.status}
                                  </Badge>
                                </td>
                                <td className="px-3 py-2">
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => handlePatientRevoke(consent.consentId)}
                                    disabled={busy === `revoke-${consent.consentId}`}
                                  >
                                    Revoke
                                  </Button>
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>

                    <div className="max-w-xs">
                      <label className="mb-1 block text-xs text-slate-500">
                        Expiry selector ({Math.round(expirySeconds / 60)} minutes)
                      </label>
                      <input
                        type="range"
                        min={300}
                        max={86400}
                        step={300}
                        value={expirySeconds}
                        onChange={(event) => setExpirySeconds(Number(event.target.value))}
                        className="w-full"
                      />
                    </div>
                  </motion.div>
                ) : null}

                {activePanel === "minio" ? (
                  <motion.div
                    key="minio"
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    className="space-y-4"
                  >
                    <h3 className="text-xl font-semibold text-slate-900">Off-Chain Storage (MinIO)</h3>
                    <p className="text-sm text-slate-500">
                      Files are stored encrypted (AES-256-CBC) and keys are managed separately.
                    </p>
                    <div className="overflow-hidden rounded-xl border border-slate-200">
                      <table className="w-full text-sm">
                        <thead className="bg-slate-50 text-left text-slate-500">
                          <tr>
                            <th className="px-3 py-2">File Name</th>
                            <th className="px-3 py-2">Size</th>
                            <th className="px-3 py-2">
                              {isResearcher ? "Preview" : "View"}
                            </th>
                            {!isResearcher ? <th className="px-3 py-2">Preview PDF</th> : null}
                          </tr>
                        </thead>
                        <tbody>
                          {minioFiles.map((file) => (
                            <tr key={file.name} className="border-t border-slate-100">
                              <td className="px-3 py-2">{file.name.split("/").pop()}</td>
                              <td className="px-3 py-2">{file.sizeLabel}</td>
                              <td className="px-3 py-2">
                                <Button
                                  size="sm"
                                  variant="outline"
                                  onClick={() => handleViewMinioFile(file.name, isResearcher)}
                                >
                                  {isResearcher ? "Preview" : "View"}
                                </Button>
                              </td>
                              {!isResearcher ? (
                                <td className="px-3 py-2">
                                  <Button
                                    size="sm"
                                    variant="outline"
                                    onClick={() => handleViewMinioFile(file.name, true)}
                                  >
                                    Preview
                                  </Button>
                                </td>
                              ) : null}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </motion.div>
                ) : null}

                {activePanel === "explorer" ? (
                  <motion.div
                    key="explorer"
                    initial={{ opacity: 0, x: 12 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -12 }}
                    className="space-y-4"
                  >
                    <h3 className="text-xl font-semibold text-slate-900">
                      Fabric Blockchain Explorer
                    </h3>
                    <div className="overflow-hidden rounded-xl border border-slate-200">
                      <table className="w-full text-sm">
                        <thead className="bg-slate-50 text-left text-slate-500">
                          <tr>
                            <th className="px-3 py-2">Block</th>
                            <th className="px-3 py-2">Transaction</th>
                            <th className="px-3 py-2">Record ID</th>
                            <th className="px-3 py-2">Action</th>
                            <th className="px-3 py-2">Time</th>
                          </tr>
                        </thead>
                        <tbody>
                          {latestTransactions.map((tx) => (
                            <tr key={`${tx.txId}-${tx.timestamp}`} className="border-t border-slate-100">
                              <td className="px-3 py-2">{tx.block}</td>
                              <td className="px-3 py-2">{shortHash(tx.txId)}</td>
                              <td className="px-3 py-2">{tx.recordId}</td>
                              <td className="px-3 py-2">{tx.action || "invoke"}</td>
                              <td className="px-3 py-2">{new Date(tx.timestamp).toLocaleTimeString()}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </motion.div>
                ) : null}
              </AnimatePresence>

              {txSteps.length > 0 ? (
                <div className="mt-4 rounded-lg border border-blue-100 bg-blue-50 p-3 text-sm text-blue-700">
                  {txSteps.map((step) => (
                    <p key={step}>{step}</p>
                  ))}
                </div>
              ) : null}
              {status ? (
                <p className="mt-4 rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700">
                  {status}
                </p>
              ) : null}
              {error ? (
                <p className="mt-4 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
              ) : null}
            </Card>

            {isRegulator ? (
              <Card className="space-y-4 p-5">
                <h3 className="text-lg font-semibold text-slate-900">Regulator Dashboard</h3>
                <div className="overflow-hidden rounded-xl border border-slate-200">
                  <table className="w-full text-sm">
                    <thead className="bg-slate-50 text-left text-slate-500">
                      <tr>
                        <th className="px-3 py-2">Record ID</th>
                        <th className="px-3 py-2">Patient</th>
                        <th className="px-3 py-2">Hospital</th>
                        <th className="px-3 py-2">Timestamp</th>
                      </tr>
                    </thead>
                    <tbody>
                      {records.slice(0, 8).map((record) => (
                        <tr key={record.recordId} className="border-t border-slate-100">
                          <td className="px-3 py-2">{record.recordId}</td>
                          <td className="px-3 py-2">{record.patientId}</td>
                          <td className="px-3 py-2">{truncateAddress(record.walletAddress)}</td>
                          <td className="px-3 py-2">{new Date(record.timestamp).toLocaleString()}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </Card>
            ) : null}
          </div>
        </div>

        <div className="h-full overflow-y-auto pl-1">
          <div className="space-y-4 pb-4">
            <ActivityFeed items={activity} />
            <Card className="p-4">
              <div className="mb-3 flex items-center gap-2">
                <ShieldEllipsis className="h-4 w-4 text-primary" />
                <h3 className="text-sm font-semibold text-slate-900">Blockchain Timeline</h3>
              </div>
              <div className="relative ml-2 space-y-3 border-l border-slate-200 pl-4">
                {timeline.slice(0, 12).map((item) => (
                  <motion.div
                    key={item.id}
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    className="relative rounded-lg border border-slate-100 bg-slate-50 px-3 py-2"
                  >
                    <span className="absolute -left-[22px] top-3 h-2.5 w-2.5 rounded-full bg-primary" />
                    <p className="text-sm text-slate-700">{item.message}</p>
                    <p className="text-xs text-slate-400">
                      {new Date(item.timestamp).toLocaleTimeString()}
                    </p>
                  </motion.div>
                ))}
              </div>
            </Card>
          </div>
        </div>
      </div>

      {approveModalOpen && selectedRequest ? (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-slate-900/35 p-4">
          <Card className="w-full max-w-md space-y-4 p-6">
            <h4 className="text-lg font-semibold text-slate-900">Hospital Approval</h4>
            <p className="text-sm text-slate-500">
              Confirm hospital-side validation for record{" "}
              <span className="font-medium">{selectedRequest.recordId}</span>.
            </p>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setApproveModalOpen(false)}>
                Cancel
              </Button>
              <Button disabled={busy === "approve"} onClick={handleApproveAccess}>
                {busy === "approve" ? "Submitting..." : "Submit"}
              </Button>
            </div>
          </Card>
        </div>
      ) : null}
      <PDFPreviewModal
        open={previewModalOpen}
        recordId={previewRecordId}
        previewUrl={previewUrl}
        onClose={() => setPreviewModalOpen(false)}
      />
    </div>
  );
}
