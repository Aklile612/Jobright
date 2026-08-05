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
      <div className="container auth-shell">
        <div className="surface auth-card">
          <h1>Welcome back</h1>
          <p>Log in to score roles and apply in-site.</p>
          <form onSubmit={onSubmit}>
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
            {error ? <p style={{ color: "var(--warn)", margin: 0 }}>{error}</p> : null}
            <button className="btn btn-primary" disabled={loading}>
              {loading ? "Signing in…" : "Log in"}
            </button>
          </form>
          <p style={{ marginTop: "1rem" }}>
            New here?{" "}
            <Link href="/signup" style={{ color: "var(--accent)", fontWeight: 700 }}>
              Create an account
            </Link>
          </p>
        </div>
      </div>
    </main>
  );
}
