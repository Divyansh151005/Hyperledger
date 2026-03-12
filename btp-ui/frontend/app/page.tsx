"use client";

import { useEffect, useState } from "react";
import { BrowserProvider } from "ethers";
import { Dashboard } from "@/components/dashboard";
import { LoginCard } from "@/components/login-card";
import { Role } from "@/lib/types";

declare global {
  interface Window {
    ethereum?: Record<string, unknown> & {
      request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
    };
  }
}

export default function HomePage() {
  const [walletAddress, setWalletAddress] = useState("");
  const [selectedRole, setSelectedRole] = useState<Role | "">("");
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);

  useEffect(() => {
    const savedWallet = window.localStorage.getItem("wallet");
    const savedRole = window.localStorage.getItem("role") as Role | null;
    if (savedWallet) setWalletAddress(savedWallet);
    if (savedRole) setSelectedRole(savedRole);
    if (savedWallet && savedRole) setIsAuthenticated(true);
  }, []);

  async function connectWallet() {
    if (!window.ethereum) {
      window.alert("MetaMask is required. Please install MetaMask and try again.");
      return;
    }

    setIsConnecting(true);
    try {
      const provider = new BrowserProvider(window.ethereum as never);
      const accounts = (await provider.send("eth_requestAccounts", [])) as string[];
      if (accounts?.[0]) {
        setWalletAddress(accounts[0]);
        window.localStorage.setItem("wallet", accounts[0]);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : "Wallet connection failed";
      window.alert(message);
    } finally {
      setIsConnecting(false);
    }
  }

  function handleContinue() {
    if (!walletAddress || !selectedRole) return;
    window.localStorage.setItem("wallet", walletAddress);
    window.localStorage.setItem("role", selectedRole);
    setIsAuthenticated(true);
  }

  function logout() {
    setIsAuthenticated(false);
    setWalletAddress("");
    setSelectedRole("");
    window.localStorage.removeItem("role");
  }

  function disconnectWallet() {
    setWalletAddress("");
    setSelectedRole("");
    setIsAuthenticated(false);
    window.localStorage.removeItem("wallet");
    window.localStorage.removeItem("role");
  }

  if (!isAuthenticated || !selectedRole) {
    return (
      <LoginCard
        walletAddress={walletAddress}
        role={selectedRole}
        isConnecting={isConnecting}
        onConnectWallet={connectWallet}
        onSelectRole={setSelectedRole}
        onContinue={handleContinue}
      />
    );
  }

  return (
    <Dashboard
      walletAddress={walletAddress}
      role={selectedRole}
      onLogout={logout}
      onDisconnect={disconnectWallet}
    />
  );
}
