"use client";

import { motion } from "framer-motion";
import { Wallet } from "lucide-react";
import { Role } from "@/lib/types";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Select } from "@/components/ui/select";
import { truncateAddress } from "@/lib/utils";

type LoginCardProps = {
  walletAddress: string;
  role: Role | "";
  isConnecting: boolean;
  onConnectWallet: () => void;
  onSelectRole: (role: Role) => void;
  onContinue: () => void;
};

export function LoginCard({
  walletAddress,
  role,
  isConnecting,
  onConnectWallet,
  onSelectRole,
  onContinue
}: LoginCardProps) {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <motion.div
        initial={{ opacity: 0, y: 28 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.45 }}
        className="w-full max-w-2xl"
      >
        <Card className="relative overflow-hidden p-10">
          <div className="absolute inset-0 -z-0 bg-[radial-gradient(circle_at_top_right,rgba(47,107,255,0.16),transparent_55%)]" />
          <div className="relative z-10 space-y-8">
            <div className="space-y-3 text-center">
              <p className="text-sm font-medium tracking-[0.2em] text-primary">
                WEB3 HEALTHCARE
              </p>
              <h1 className="text-3xl font-semibold text-slate-900">
                Blockchain Medical Data Sharing System
              </h1>
              <p className="text-slate-500">
                Secure Healthcare Data Exchange using Hyperledger Fabric
              </p>
            </div>

            <div className="space-y-5 rounded-xl bg-slate-50/80 p-6">
              <Button
                size="lg"
                className="w-full gap-2"
                onClick={onConnectWallet}
                disabled={isConnecting}
              >
                <Wallet className="h-4 w-4" />
                {isConnecting ? "Connecting..." : "Connect Wallet (MetaMask)"}
              </Button>

              {walletAddress ? (
                <div className="space-y-4">
                  <p className="rounded-lg border border-emerald-100 bg-emerald-50 px-4 py-2 text-sm text-emerald-700">
                    Connected wallet: {truncateAddress(walletAddress)}
                  </p>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-slate-600">
                      Select Role
                    </label>
                    <Select
                      value={role}
                      onChange={(event) => onSelectRole(event.target.value as Role)}
                    >
                      <option value="">Choose your role</option>
                      <option value="hospital">Hospital</option>
                      <option value="researcher">Researcher</option>
                      <option value="patient">Patient</option>
                      <option value="regulator">Regulator</option>
                    </Select>
                  </div>
                  <Button
                    className="w-full"
                    onClick={onContinue}
                    disabled={!role}
                  >
                    Enter Dashboard
                  </Button>
                </div>
              ) : null}
            </div>
          </div>
        </Card>
      </motion.div>
    </div>
  );
}
