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
        if (err) throw new Error(err.message ?? "sign in failed");
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
      // Fallback: mint demo team token without Better Auth (local seed)
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
    <div className="min-h-screen flex items-center justify-center px-4">
      <form
        onSubmit={onSubmit}
        className="w-full max-w-md rounded-xl border border-zinc-800 bg-[#141417]/90 p-8 shadow-2xl shadow-black/40"
      >
        <h1 className="font-[family-name:var(--font-display)] text-4xl text-zinc-50 mb-2">
          Raisin
        </h1>
        <p className="text-sm text-zinc-500 mb-8">
          {mode === "signin" ? "Sign in to the console" : "Create your Raisin account"}
        </p>

        {mode === "signup" && (
          <>
            <label className="block text-xs uppercase tracking-wider text-zinc-500 mb-2">Name</label>
            <input
              className="mb-4 w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2.5 text-sm outline-none focus:border-orange-500/60"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </>
        )}

        <label className="block text-xs uppercase tracking-wider text-zinc-500 mb-2">Email</label>
        <input
          className="mb-4 w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2.5 text-sm outline-none focus:border-orange-500/60"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          type="email"
          required
        />
        <label className="block text-xs uppercase tracking-wider text-zinc-500 mb-2">Password</label>
        <input
          className="w-full rounded-md border border-zinc-700 bg-zinc-950 px-3 py-2.5 text-sm outline-none focus:border-orange-500/60"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          type="password"
          minLength={8}
          required
        />

        {error && <p className="mt-3 text-sm text-red-400">{error}</p>}

        <button
          type="submit"
          disabled={loading}
          className="mt-6 w-full rounded-md bg-orange-500 px-3 py-2.5 text-sm font-medium text-black hover:bg-orange-400 disabled:opacity-50"
        >
          {loading ? "…" : mode === "signin" ? "Sign in" : "Create account"}
        </button>

        <button
          type="button"
          onClick={() => setMode(mode === "signin" ? "signup" : "signin")}
          className="mt-4 w-full text-sm text-zinc-500 hover:text-zinc-200"
        >
          {mode === "signin" ? "Need an account? Sign up" : "Have an account? Sign in"}
        </button>

        <button
          type="button"
          onClick={demoSkip}
          className="mt-2 w-full text-xs text-zinc-600 hover:text-zinc-400"
        >
          Continue with seeded demo team
        </button>
      </form>
    </div>
  );
}
