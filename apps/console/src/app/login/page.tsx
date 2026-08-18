"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { signIn, signUp } from "@/lib/auth-client";
import { Field, WorkspaceMark } from "@/components/ui";

function LoginInner() {
  const router = useRouter();
  const params = useSearchParams();
  const next = params.get("next")?.trim() || "/overview";
  const [mode, setMode] = useState<"signin" | "signup">("signin");
  const [email, setEmail] = useState("demo@raisin.run");
  const [password, setPassword] = useState("demo-demo-demo");
  const [name, setName] = useState("Demo User");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  function safeNext(path: string) {
    if (path.startsWith("/") && !path.startsWith("//")) return path;
    return "/overview";
  }

  useEffect(() => {
    if (localStorage.getItem("raisin_team_token")) {
      router.replace(safeNext(next));
    }
  }, [router, next]);

  async function provision() {
    const res = await fetch("/api/session/provision", { method: "POST" });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error ?? "provision failed");
    localStorage.setItem("raisin_team_token", data.token);
    localStorage.setItem("raisin_email", data.email);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      if (mode === "signup") {
        const { error: err } = await signUp.email({ email, password, name });
        if (err) throw new Error(err.message ?? "sign up failed");
      } else {
        const { error: err } = await signIn.email({ email, password });
        if (err) {
          if (email === "demo@raisin.run") {
            const { error: upErr } = await signUp.email({
              email,
              password,
              name: name || "Demo User",
            });
            if (upErr) throw new Error(err.message ?? "sign in failed");
            const { error: inErr } = await signIn.email({ email, password });
            if (inErr) throw new Error(inErr.message ?? "sign in failed");
          } else {
            throw new Error(err.message ?? "sign in failed");
          }
        }
      }
      await provision();
      router.replace(safeNext(next));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "login failed");
    } finally {
      setLoading(false);
    }
  }

  async function demoSkip() {
    setLoading(true);
    setError("");
    try {
      const res = await fetch("/api/session", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: "demo@raisin.run" }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error ?? "demo login failed");
      localStorage.setItem("raisin_team_token", data.token);
      localStorage.setItem("raisin_email", data.email);
      router.replace(safeNext(next));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="relative flex min-h-dvh flex-col">
      <header className="flex h-14 items-center px-5">
        <Link href="/" className="flex items-center gap-2">
          <WorkspaceMark />
          <span className="text-[13px] font-medium text-zinc-50">Raisin</span>
        </Link>
      </header>

      <main className="flex flex-1 items-center justify-center px-5 pb-24">
        <form onSubmit={onSubmit} className="w-full max-w-[360px]">
          <h1 className="text-[22px] font-medium tracking-tight text-zinc-50">
            {mode === "signin" ? "Sign in to Raisin" : "Create a Raisin account"}
          </h1>
          <p className="mt-1.5 text-[13px] text-[var(--muted)]">
            {mode === "signin"
              ? "Use your workspace email to manage sending."
              : "A local account for this console workspace."}
          </p>

          <div className="mt-8 grid gap-3">
            {mode === "signup" && (
              <Field label="Name" htmlFor="name">
                <input
                  id="name"
                  className="field w-full"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                />
              </Field>
            )}
            <Field label="Email" htmlFor="email">
              <input
                id="email"
                className="field w-full"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                type="email"
                required
              />
            </Field>
            <Field label="Password" htmlFor="password">
              <input
                id="password"
                className="field w-full"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                type="password"
                minLength={8}
                required
              />
            </Field>
          </div>

          {error && <p className="mt-3 text-[13px] text-red-400">{error}</p>}

          <button type="submit" disabled={loading} className="btn-primary mt-6 h-8 w-full">
            {loading ? "Signing in…" : mode === "signin" ? "Sign in" : "Create account"}
          </button>

          <button
            type="button"
            onClick={() => setMode(mode === "signin" ? "signup" : "signin")}
            className="mt-4 w-full text-[13px] text-[var(--muted)] hover:text-zinc-200"
          >
            {mode === "signin" ? "Need an account? Sign up" : "Have an account? Sign in"}
          </button>

          <div className="relative my-6">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-[var(--border)]" />
            </div>
            <div className="relative flex justify-center text-[12px]">
              <span className="bg-[var(--background)] px-2 text-[var(--muted-2)]">local</span>
            </div>
          </div>

          <button type="button" onClick={demoSkip} disabled={loading} className="btn-ghost w-full">
            Continue with seeded demo team
          </button>
        </form>
      </main>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={<div className="p-8 text-[13px] text-[var(--muted)]">Loading…</div>}>
      <LoginInner />
    </Suspense>
  );
}
