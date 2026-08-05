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

      <section className="hero">
        <div className="hero__glow" />
        <div className="hero__plane" aria-hidden />
        <div className="container hero__content">
          <p className="rise badge">Built for software engineers</p>
          <h1 className="rise rise-delay-1 hero__brand">
            Job<span>Right</span>
          </h1>
          <p className="rise rise-delay-2 hero__copy">
            Pull SWE roles from Remotive, Arbeitnow, and RemoteOK. Score your
            resume, forge it for the job, then apply inside the site — no paid
            extension.
          </p>
          <div className="rise rise-delay-3 hero__cta">
            <Link href="/jobs" className="btn btn-primary">
              Browse open roles
            </Link>
            <Link href="/signup" className="btn btn-ghost">
              Create free account
            </Link>
          </div>
        </div>
      </section>

      <JobFeed initialJobs={jobs} />

      <section className="section" style={{ paddingTop: 0 }}>
        <div className="container">
          <div className="feature-band">
            <div className="feature-band__grid">
              <div>
                <p className="badge">Workflow</p>
                <h2 className="section-title">Score, forge, apply in one place</h2>
                <p className="section-sub">
                  JobRight holds your catalog and applications. Resume_forge
                  scores ATS fit and rewrites your CV for each role. The apply
                  workspace opens the real listing URL beside your autofill panel.
                </p>
              </div>
              <ol className="steps">
                {[
                  "Upload one resume",
                  "Match against any software role",
                  "Forge a tailored version",
                  "Autofill the application in-site",
                ].map((step, i) => (
                  <li key={step} className="step">
                    <span className="step-num">{i + 1}</span>
                    <span>{step}</span>
                  </li>
                ))}
              </ol>
            </div>
          </div>
        </div>
      </section>

      <footer className="site-footer">
        <div className="container site-footer__inner">
          <p className="brand" style={{ fontSize: "1.1rem" }}>
            Job<span>Right</span>
          </p>
          <p>Software roles · ATS scoring · in-site apply</p>
        </div>
      </footer>
    </main>
  );
}
