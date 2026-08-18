"use client";

import { useEffect } from "react";
import Link from "next/link";
import { WorkspaceMark } from "@/components/ui";

const SPEC_URL = "/api/proxy/openapi.yaml";

export default function DocsPage() {
  useEffect(() => {
    const existing = document.getElementById("scalar-api-reference-script");
    if (existing) return;

    const config = document.createElement("script");
    config.id = "api-reference";
    config.setAttribute("data-url", SPEC_URL);
    config.setAttribute(
      "data-configuration",
      JSON.stringify({
        theme: "kepler",
        hideModels: true,
        operationsSorter: "alpha",
        defaultHttpClient: { targetKey: "shell", clientKey: "curl" },
      })
    );
    document.body.appendChild(config);

    const script = document.createElement("script");
    script.id = "scalar-api-reference-script";
    script.src = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.28.11";
    document.body.appendChild(script);

    return () => {
      config.remove();
      script.remove();
    };
  }, []);

  return (
    <div className="flex min-h-dvh flex-col bg-[#0a0a0b]">
      <header className="flex h-12 shrink-0 items-center justify-between border-b border-[var(--border)] px-4">
        <Link href="/" className="flex items-center gap-2">
          <WorkspaceMark size={18} />
          <span className="text-[13px] font-medium text-zinc-50">Raisin</span>
          <span className="text-[13px] text-zinc-500">API Reference</span>
        </Link>
        <div className="flex items-center gap-4 text-[13px]">
          <a
            href={
              (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:18080").replace(/\/$/, "") +
              "/openapi.yaml"
            }
            className="text-[var(--muted)] hover:text-zinc-200"
          >
            openapi.yaml
          </a>
          <Link href="/login" className="text-[var(--muted)] hover:text-zinc-200">
            Console
          </Link>
        </div>
      </header>
      <div className="min-h-0 flex-1" id="scalar-mount" />
    </div>
  );
}
