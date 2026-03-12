import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Medical Data Sharing Dashboard",
  description: "Professional Web3 healthcare dashboard with Fabric + MinIO"
};

export default function RootLayout({
  children
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
