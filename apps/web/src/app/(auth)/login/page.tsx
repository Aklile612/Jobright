"use client";

import { FormEvent, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { setToken } from "@/lib/auth";
import { SiteHeader } from "@/components/layout/SiteHeader";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const res = await api.login({ email, password });
      setToken(res.token);
      router.push("/jobs");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main>
      <SiteHeader solid />
      <div className="container grid min-h-[70vh] place-items-center py-12">
        <form onSubmit={onSubmit} className="surface w-full max-w-md rounded-[24px] p-8">
          <h1 className="font-[family-name:var(--font-display)] text-3xl font-extrabold">
            Welcome back
          </h1>
          <p className="mt-2 text-sm text-[var(--ink-soft)]">
            Log in to score roles and apply in-site.
          </p>
          <div className="mt-6 grid gap-4">
            <div className="field">
              <label htmlFor="email">Email</label>
              <input
                id="email"
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div className="field">
              <label htmlFor="password">Password</label>
              <input
                id="password"
                type="password"
                required
                minLength={8}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            {error ? <p className="text-sm text-[var(--warn)]">{error}</p> : null}
            <button className="btn btn-primary w-full" disabled={loading}>
              {loading ? "Signing in…" : "Log in"}
            </button>
          </div>
          <p className="mt-4 text-sm text-[var(--ink-soft)]">
            New here?{" "}
            <Link href="/signup" className="font-semibold text-[var(--accent-deep)]">
              Create an account
            </Link>
          </p>
        </form>
      </div>
    </main>
  );
}
