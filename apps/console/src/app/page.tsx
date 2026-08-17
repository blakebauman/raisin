"use client";

import { useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";

const API_HEALTH =
  (typeof process !== "undefined" && process.env.NEXT_PUBLIC_API_URL
    ? `${process.env.NEXT_PUBLIC_API_URL.replace(/\/$/, "")}/health`
    : "https://api.raisin.run/health");

export default function LandingPage() {
  const router = useRouter();

  useEffect(() => {
    if (localStorage.getItem("raisin_team_token")) {
      router.replace("/overview");
    }
  }, [router]);

  return (
    <div className="relative min-h-screen overflow-hidden">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10"
        style={{
          background:
            "radial-gradient(ellipse 90% 60% at 50% -20%, rgba(249,115,22,0.18), transparent 55%), linear-gradient(180deg, #0c0c0e 0%, #121218 100%)",
        }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 opacity-[0.35]"
        style={{
          backgroundImage:
            "linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px)",
          backgroundSize: "64px 64px",
          maskImage: "radial-gradient(ellipse 70% 50% at 50% 30%, black, transparent)",
        }}
      />

      <header className="mx-auto flex w-full max-w-5xl items-center justify-between px-6 pt-8">
        <div className="font-[family-name:var(--font-display)] text-2xl tracking-tight text-zinc-50 animate-[fadeIn_0.7s_ease-out]">
          Raisin
        </div>
        <nav className="flex items-center gap-4 text-sm animate-[fadeIn_0.7s_ease-out_0.1s_both]">
          <a href={API_HEALTH} className="text-zinc-500 hover:text-zinc-200">
            API
          </a>
          <Link href="/login" className="rounded-md bg-orange-500 px-3.5 py-2 font-medium text-black hover:bg-orange-400">
            Console
          </Link>
        </nav>
      </header>

      <main className="mx-auto flex min-h-[calc(100vh-5rem)] w-full max-w-5xl flex-col justify-center px-6 pb-24 pt-16">
        <p className="font-[family-name:var(--font-display)] text-6xl leading-[0.95] tracking-tight text-zinc-50 sm:text-7xl md:text-8xl animate-[rise_0.9s_ease-out]">
          Raisin
        </p>
        <h1 className="mt-6 max-w-xl text-xl text-zinc-300 sm:text-2xl animate-[rise_0.9s_ease-out_0.12s_both]">
          Email for developers.
        </h1>
        <p className="mt-4 max-w-md text-base leading-relaxed text-zinc-500 animate-[rise_0.9s_ease-out_0.2s_both]">
          Send transactional and marketing mail through a clean API — powered by Amazon SES, built for
          raisin.run.
        </p>
        <div className="mt-10 flex flex-wrap gap-3 animate-[rise_0.9s_ease-out_0.28s_both]">
          <Link
            href="/login"
            className="rounded-md bg-orange-500 px-5 py-2.5 text-sm font-medium text-black hover:bg-orange-400"
          >
            Open console
          </Link>
          <a
            href="https://github.com/blakebauman/raisin"
            className="rounded-md border border-zinc-700 px-5 py-2.5 text-sm text-zinc-200 hover:bg-zinc-900"
          >
            Docs & SDKs
          </a>
        </div>
      </main>

      <style>{`
        @keyframes fadeIn {
          from { opacity: 0; }
          to { opacity: 1; }
        }
        @keyframes rise {
          from { opacity: 0; transform: translateY(12px); }
          to { opacity: 1; transform: translateY(0); }
        }
      `}</style>
    </div>
  );
}
