"use client";

import { useEffect, useMemo, useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

type PDFPreviewModalProps = {
  open: boolean;
  recordId: string;
  previewUrl: string;
  onClose: () => void;
};

export function PDFPreviewModal({ open, recordId, previewUrl, onClose }: PDFPreviewModalProps) {
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!open) return;

    const blockContextMenu = (event: MouseEvent) => event.preventDefault();
    const blockShortcuts = (event: KeyboardEvent) => {
      const key = event.key.toLowerCase();
      if (event.ctrlKey && ["s", "p", "u"].includes(key)) {
        event.preventDefault();
      }
    };

    document.addEventListener("contextmenu", blockContextMenu);
    document.addEventListener("keydown", blockShortcuts);
    return () => {
      document.removeEventListener("contextmenu", blockContextMenu);
      document.removeEventListener("keydown", blockShortcuts);
    };
  }, [open]);

  useEffect(() => {
    if (open) setIsLoading(true);
  }, [open, previewUrl]);

  const framedUrl = useMemo(() => `${previewUrl}#toolbar=0`, [previewUrl]);

  return (
    <AnimatePresence>
      {open ? (
        <motion.div
          className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
        >
          <motion.div
            className="relative w-full max-w-5xl rounded-2xl bg-white p-5 shadow-2xl"
            initial={{ opacity: 0, y: 16, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 12, scale: 0.98 }}
            transition={{ duration: 0.2 }}
          >
            <div className="mb-3 flex items-center justify-between">
              <div>
                <h3 className="text-lg font-semibold text-slate-900">Medical Record Preview</h3>
                <p className="text-xs text-slate-500">Record: {recordId}</p>
              </div>
              <Button variant="outline" onClick={onClose}>
                Close
              </Button>
            </div>

            <div className="relative overflow-hidden rounded-xl border border-slate-200 bg-slate-50">
              <div className="pointer-events-none absolute inset-0 z-20 flex items-center justify-center">
                <p className="select-none -rotate-[30deg] text-5xl font-semibold text-slate-900/15">
                  Research Access Only
                  <br />
                  Do Not Distribute
                </p>
              </div>

              {isLoading ? (
                <div className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-2 bg-white/85">
                  <Loader2 className="h-5 w-5 animate-spin text-primary" />
                  <p className="text-sm text-slate-700">Verifying consent...</p>
                  <p className="text-sm text-slate-700">Loading document...</p>
                </div>
              ) : null}

              <iframe
                src={framedUrl}
                title="Medical Record Preview"
                className="h-[700px] w-full"
                onLoad={() => setIsLoading(false)}
              />
            </div>
          </motion.div>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );
}
