"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { signIn, signUp } from "@/lib/auth-client";

export default function LoginPage() {
  const router = useRouter();
  const [mode, setMode] = useState<"signin" | "signup">("signin");
  const [email, setEmail] = useState("demo@raisin.run");
  const [password, setPassword] = useState("demo-demo-demo");
  const [name, setName] = useState("Demo User");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (localStorage.getItem("raisin_team_token")) {
      router.replace("/overview");
    }
  }, [router]);

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
      router.replace("/overview");
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
      router.replace("/overview");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <form
        onSubmit={onSubmit}
        className="w-full max-w-md rounded-xl border border-zinc-800 bg-[var(--panel)]/95 p-8 shadow-2xl shadow-black/40"
      >
        <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-zinc-500">raisin.run</p>
        <h1 className="mt-2 font-[family-name:var(--font-display)] text-4xl tracking-tight text-zinc-50">
          Console
        </h1>
        <p className="mt-2 text-sm text-zinc-500">
          {mode === "signin" ? "Sign in to manage email for your team." : "Create an account for this workspace."}
        </p>

        <div className="mt-8 space-y-4">
          {mode === "signup" && (
            <div>
              <label className="mb-1.5 block text-xs uppercase tracking-wider text-zinc-500">Name</label>
              <input className="field" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
          )}
          <div>
            <label className="mb-1.5 block text-xs uppercase tracking-wider text-zinc-500">Email</label>
            <input
              className="field"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              type="email"
              required
            />
          </div>
          <div>
            <label className="mb-1.5 block text-xs uppercase tracking-wider text-zinc-500">Password</label>
            <input
              className="field"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              type="password"
              minLength={8}
              required
            />
          </div>
        </div>

        {error && <p className="mt-3 text-sm text-red-400">{error}</p>}

        <button type="submit" disabled={loading} className="btn-primary mt-6 w-full py-2.5">
          {loading ? "…" : mode === "signin" ? "Sign in" : "Create account"}
        </button>

        <button
          type="button"
          onClick={() => setMode(mode === "signin" ? "signup" : "signin")}
          className="mt-4 w-full text-sm text-zinc-400 hover:text-zinc-200"
        >
          {mode === "signin" ? "Need an account? Sign up" : "Have an account? Sign in"}
        </button>

        <div className="relative my-6">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-zinc-800" />
          </div>
          <div className="relative flex justify-center text-[11px] uppercase tracking-wider">
            <span className="bg-[var(--panel)] px-2 text-zinc-600">local</span>
          </div>
        </div>

        <button type="button" onClick={demoSkip} disabled={loading} className="btn-ghost w-full text-xs text-zinc-500">
          Continue with seeded demo team
        </button>
      </form>
    </div>
  );
}
