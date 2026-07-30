import type { Metadata } from "next";
import GlobalProviders from "@/components/providers/global-providers";
import "@/globals.css";

export const metadata: Metadata = {
  title: "taskd",
  description: "taskd admin console",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body className="antialiased h-full">
        <GlobalProviders>
          <div className="w-screen h-screen flex">{children}</div>
        </GlobalProviders>
      </body>
    </html>
  );
}
