"use client";

import { useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const path = usePathname();
  const [state, setState] = useState<"loading" | "ok" | "denied">("loading");

  useEffect(() => {
    if (typeof window === "undefined") return;
    const token = localStorage.getItem("raisin_team_token");
    if (!token) {
      setState("denied");
      router.replace("/login");
      return;
    }
    setState("ok");
  }, [router, path]);

  if (state === "loading") {
    return <div className="min-h-dvh bg-[var(--background)]" />;
  }
  if (state === "denied") {
    return null;
  }
  return <>{children}</>;
}
