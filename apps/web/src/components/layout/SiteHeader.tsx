"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { clearToken, isAuthenticated } from "@/lib/auth";
import { useEffect, useState } from "react";

const links = [
  { href: "/jobs", label: "Jobs" },
  { href: "/applications", label: "Applications" },
  { href: "/profile", label: "Profile" },
];

export function SiteHeader({ solid = false }: { solid?: boolean }) {
  const pathname = usePathname();
  const [authed, setAuthed] = useState(false);

  useEffect(() => {
    setAuthed(isAuthenticated());
  }, [pathname]);

  return (
    <header className={`site-header ${solid ? "is-solid" : ""}`}>
      <div className="container site-header__inner">
        <Link href="/" className="brand">
          JobRight
        </Link>
        <nav className="site-nav">
          {links.map((link) => (
            <Link
              key={link.href}
              href={link.href}
              className={pathname?.startsWith(link.href) ? "is-active" : ""}
            >
              {link.label}
            </Link>
          ))}
        </nav>
        <div className="header-actions">
          {authed ? (
            <>
              <Link href="/profile" className="btn btn-ghost btn-sm">
                My profile
              </Link>
              <button
                className="btn btn-primary btn-sm"
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
              <Link href="/login" className="btn btn-ghost btn-sm">
                Log in
              </Link>
              <Link href="/signup" className="btn btn-primary btn-sm">
                Sign up
              </Link>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
