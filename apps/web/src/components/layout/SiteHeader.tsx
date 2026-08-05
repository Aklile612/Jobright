"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { clearToken, isAuthenticated } from "@/lib/auth";
import { useEffect, useState } from "react";

const links = [
  { href: "/jobs", label: "Jobs" },
  { href: "/applications", label: "Applications" },
  { href: "/resumes", label: "Resumes" },
];

export function SiteHeader({ solid = false }: { solid?: boolean }) {
  const pathname = usePathname();
  const [authed, setAuthed] = useState(false);

  useEffect(() => {
    setAuthed(isAuthenticated());
  }, [pathname]);

  return (
    <header
      className={`sticky top-0 z-40 border-b ${
        solid
          ? "border-[var(--line)] bg-[rgba(247,250,248,0.92)] backdrop-blur-md"
          : "border-transparent bg-transparent"
      }`}
    >
      <div className="container flex items-center justify-between py-4">
        <Link href="/" className="font-[family-name:var(--font-display)] text-2xl font-extrabold tracking-tight">
          Job<span className="text-[var(--accent)]">Right</span>
        </Link>
        <nav className="hidden items-center gap-6 md:flex">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={`text-sm font-semibold transition ${
                pathname?.startsWith(link.href)
                  ? "text-[var(--accent-deep)]"
                  : "text-[var(--ink-soft)] hover:text-[var(--ink)]"
              }`}
            >
              {link.label}
            </Link>
          ))}
        </nav>
        <div className="flex items-center gap-2">
          {authed ? (
            <>
              <Link href="/jobs" className="btn btn-ghost text-sm">
                Dashboard
              </Link>
              <button
                className="btn btn-primary text-sm"
                onClick={() => {
                  clearToken();
                  setAuthed(false);
                  window.location.href = "/";
                }}
              >
                Sign out
              </button>
            </>
          ) : (
            <>
              <Link href="/login" className="btn btn-ghost text-sm">
                Log in
              </Link>
              <Link href="/signup" className="btn btn-primary text-sm">
                Get started
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
