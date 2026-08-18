"use client";

import { useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ProductPreview } from "@/components/product-preview";
import { WorkspaceMark } from "@/components/ui";

const API_HEALTH =
  typeof process !== "undefined" && process.env.NEXT_PUBLIC_API_URL
    ? `${process.env.NEXT_PUBLIC_API_URL.replace(/\/$/, "")}/health`
    : "https://api.raisin.run/health";

export default function LandingPage() {
  const router = useRouter();

  useEffect(() => {
    if (localStorage.getItem("raisin_team_token")) {
      router.replace("/overview");
    }
  }, [router]);

  return (
    <div className="relative min-h-dvh overflow-x-hidden">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[720px]"
        style={{
          background:
            "radial-gradient(ellipse 70% 55% at 50% -10%, rgba(255,255,255,0.07), transparent 58%)",
        }}
      />

      <header className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between px-5">
        <Link href="/" className="flex items-center gap-2">
          <WorkspaceMark />
          <span className="text-[13px] font-medium text-zinc-50">Raisin</span>
        </Link>
        <nav className="flex items-center gap-5 text-[13px]">
          <a href={API_HEALTH} className="hidden text-[var(--muted)] hover:text-zinc-200 sm:inline">
            API
          </a>
          <a
            href="https://github.com/blakebauman/raisin"
            className="hidden text-[var(--muted)] hover:text-zinc-200 sm:inline"
          >
            Docs
          </a>
          <Link href="/login" className="text-[var(--muted)] hover:text-zinc-200">
            Log in
          </Link>
          <Link
            href="/login"
            className="inline-flex h-7 items-center rounded-full bg-white px-3 text-[13px] font-medium text-[#0a0a0b] transition hover:bg-zinc-200"
          >
            Open console
          </Link>
        </nav>
      </header>

      <main>
        <section className="mx-auto max-w-6xl px-5 pb-6 pt-10 text-center sm:pt-12">
          <h1 className="mx-auto max-w-3xl text-[2.5rem] font-medium leading-[1.08] tracking-[-0.03em] text-zinc-50 text-balance sm:text-5xl md:text-[3.5rem] motion-safe:animate-[heroIn_0.7s_cubic-bezier(0.16,1,0.3,1)_both]">
            The email platform for products that ship
          </h1>
          <p className="mx-auto mt-4 max-w-xl text-[15px] leading-relaxed text-zinc-400 text-pretty motion-safe:animate-[heroIn_0.7s_cubic-bezier(0.16,1,0.3,1)_0.08s_both]">
            Send transactional and marketing mail through a clean API. Inspect delivery, manage audiences, and operate
            domains from one console on raisin.run.
          </p>
          <div className="mt-7 flex flex-wrap items-center justify-center gap-3 motion-safe:animate-[heroIn_0.7s_cubic-bezier(0.16,1,0.3,1)_0.14s_both]">
            <Link
              href="/login"
              className="inline-flex h-8 items-center rounded-full bg-white px-4 text-[13px] font-medium text-[#0a0a0b] transition hover:bg-zinc-200"
            >
              Open console
            </Link>
            <a
              href="https://github.com/blakebauman/raisin"
              className="inline-flex h-8 items-center rounded-full border border-[var(--border)] px-4 text-[13px] text-zinc-200 transition hover:bg-white/[0.04]"
            >
              Docs & SDKs
            </a>
          </div>
        </section>

        <section className="relative mx-auto max-w-6xl px-5 pb-20 motion-safe:animate-[heroRise_0.9s_cubic-bezier(0.16,1,0.3,1)_0.18s_both]">
          <div
            aria-hidden
            className="pointer-events-none absolute left-1/2 top-[12%] h-[70%] w-[80%] -translate-x-1/2 rounded-full blur-3xl"
            style={{ background: "radial-gradient(ellipse at center, rgba(255,255,255,0.08), transparent 70%)" }}
          />
          <div className="relative">
            <ProductPreview />
          </div>
        </section>

        <section className="mx-auto grid w-full max-w-6xl gap-12 px-5 pb-28 sm:grid-cols-3 sm:gap-8">
          <div>
            <h2 className="text-[15px] font-medium text-zinc-50">Send</h2>
            <p className="mt-2 max-w-[36ch] text-[13px] leading-relaxed text-[var(--muted)]">
              REST, SMTP, templates, and broadcasts. Queue a message and watch it land through Amazon SES.
            </p>
          </div>
          <div>
            <h2 className="text-[15px] font-medium text-zinc-50">Inspect</h2>
            <p className="mt-2 max-w-[36ch] text-[13px] leading-relaxed text-[var(--muted)]">
              Events, webhooks, and API logs. Open a send and see every bounce, open, and click as it happens.
            </p>
          </div>
          <div>
            <h2 className="text-[15px] font-medium text-zinc-50">Operate</h2>
            <p className="mt-2 max-w-[36ch] text-[13px] leading-relaxed text-[var(--muted)]">
              Domains, keys, contacts, and suppressions. The same surfaces you need after the first hello-world.
            </p>
          </div>
        </section>
      </main>

      <footer className="mx-auto flex w-full max-w-6xl items-center justify-between border-t border-[var(--border)] px-5 py-6 text-[12px] text-[var(--muted-2)]">
        <span>Raisin</span>
        <a href="https://github.com/blakebauman/raisin" className="hover:text-zinc-300">
          GitHub
        </a>
      </footer>

      <style>{`
        @keyframes heroIn {
          from { opacity: 0; transform: translateY(8px); }
          to { opacity: 1; transform: translateY(0); }
        }
        @keyframes heroRise {
          from { opacity: 0; transform: translateY(18px); }
          to { opacity: 1; transform: translateY(0); }
        }
        @media (prefers-reduced-motion: reduce) {
          @keyframes heroIn {
            from { opacity: 0; transform: none; }
            to { opacity: 1; transform: none; }
          }
          @keyframes heroRise {
            from { opacity: 0; transform: none; }
            to { opacity: 1; transform: none; }
          }
        }
      `}</style>
    </div>
  );
}
