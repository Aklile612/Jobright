import Link from "next/link";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { JobFeed } from "@/features/jobs/JobFeed";
import { loadSoftwareJobs } from "@/lib/jobs/loadJobs";

export const dynamic = "force-dynamic";

export default async function HomePage() {
  const jobs = await loadSoftwareJobs();

  return (
    <main>
      <SiteHeader />
      <section className="relative overflow-hidden">
        <div
          className="pointer-events-none absolute inset-0 opacity-70"
          style={{
            backgroundImage:
              "linear-gradient(120deg, rgba(15,122,95,0.12), transparent 40%), url(\"data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%230f7a5f' fill-opacity='0.05'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E\")",
          }}
        />
        <div className="container relative grid min-h-[78vh] items-center gap-10 py-16 lg:grid-cols-[1.1fr_0.9fr]">
          <div>
            <p className="rise badge mb-5">Software engineering first</p>
            <h1 className="rise rise-delay-1 font-[family-name:var(--font-display)] text-5xl font-extrabold leading-[0.95] tracking-tight md:text-7xl">
              Job<span className="text-[var(--accent)]">Right</span>
            </h1>
            <p className="rise rise-delay-2 mt-5 max-w-xl text-lg leading-relaxed text-[var(--ink-soft)]">
              Pull roles from multiple boards, score your resume, forge it for the
              job with Resume_forge, then apply inside JobRight — no paid extension.
            </p>
            <div className="rise rise-delay-3 mt-8 flex flex-wrap gap-3">
              <Link href="/jobs" className="btn btn-primary">
                Browse roles
              </Link>
              <Link href="/signup" className="btn btn-ghost">
                Create free account
              </Link>
            </div>
            <dl className="mt-10 grid max-w-lg grid-cols-3 gap-4 text-sm">
              <div>
                <dt className="text-[var(--ink-soft)]">Sources</dt>
                <dd className="font-[family-name:var(--font-display)] text-2xl font-bold">
                  3+
                </dd>
              </div>
              <div>
                <dt className="text-[var(--ink-soft)]">Focus</dt>
                <dd className="font-[family-name:var(--font-display)] text-2xl font-bold">
                  SWE
                </dd>
              </div>
              <div>
                <dt className="text-[var(--ink-soft)]">Apply</dt>
                <dd className="font-[family-name:var(--font-display)] text-2xl font-bold">
                  In-site
                </dd>
              </div>
            </dl>
          </div>
          <div className="rise rise-delay-2 relative hidden min-h-[420px] lg:block">
            <div
              className="absolute inset-0 rounded-[28px]"
              style={{
                background:
                  "linear-gradient(160deg, #0f7a5f 0%, #12352c 45%, #1c4d3f 100%)",
                animation: "drift 4s ease-in-out infinite alternate",
              }}
            />
            <div className="absolute inset-6 flex flex-col justify-between rounded-[22px] border border-white/15 bg-white/10 p-6 text-white backdrop-blur-sm">
              <div>
                <p className="text-sm uppercase tracking-[0.18em] text-white/70">
                  Apply workspace
                </p>
                <h2 className="mt-3 font-[family-name:var(--font-display)] text-3xl font-bold">
                  Autofill beside the real application URL
                </h2>
              </div>
              <p className="max-w-sm text-sm leading-relaxed text-white/80">
                Open Greenhouse, Lever, Remotive and more inside JobRight. Copy or
                push your name, email, and resume fields into the form without
                leaving the site.
              </p>
            </div>
          </div>
        </div>
      </section>

      <JobFeed initialJobs={jobs} />

      <section className="container pb-20">
        <div className="surface overflow-hidden rounded-[28px] p-8 md:p-12">
          <div className="grid gap-8 md:grid-cols-2 md:items-center">
            <div>
              <h2 className="font-[family-name:var(--font-display)] text-3xl font-extrabold">
                Score, forge, then apply
              </h2>
              <p className="mt-3 text-[var(--ink-soft)]">
                JobRight keeps your catalog and applications. Resume_forge handles
                ATS scoring and CV rewriting for each role.
              </p>
            </div>
            <ol className="grid gap-3 text-sm">
              {[
                "Upload one resume",
                "Get an ATS match score for any role",
                "Forge a tailored version for that job",
                "Open the listing URL in-site and autofill",
              ].map((step, i) => (
                <li
                  key={step}
                  className="flex items-center gap-3 rounded-2xl bg-[var(--paper-2)] px-4 py-3"
                >
                  <span className="flex h-8 w-8 items-center justify-center rounded-full bg-[var(--accent)] text-xs font-bold text-white">
                    {i + 1}
                  </span>
                  {step}
                </li>
              ))}
            </ol>
          </div>
        </div>
      </section>
    </main>
  );
}
