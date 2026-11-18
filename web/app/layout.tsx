import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "APIScope: verified, runnable open APIs",
  description: "A living directory of open APIs with measured health and a multi-language playground.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <header className="site">
          <Link href="/" className="brand">APIScope</Link>
          <span className="muted">verified, runnable open APIs</span>
          <a href="https://github.com/mugambi645/apiscope" className="gh">GitHub</a>
        </header>
        <main>{children}</main>
      </body>
    </html>
  );
}
