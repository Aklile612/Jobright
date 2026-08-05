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
        <div className="container hero__content">
          <p className="badge">Software engineering jobs</p>
          <h1 className="hero__brand">JobRight</h1>
          <p className="hero__copy">
            Find SWE roles, upload your resume once, fill applications in-site,
            and track what you applied to.
          </p>
          <div className="hero__cta">
            <Link href="/jobs" className="btn btn-primary">
              Browse jobs
            </Link>
            <Link href="/profile" className="btn btn-ghost">
              Set up profile
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
                <p className="badge">How it works</p>
                <h2 className="section-title">Resume → profile → apply</h2>
                <p className="section-sub">
                  Your profile and resume live in the database. When you open a
                  job in-site, those fields are ready to copy or autofill.
                </p>
              </div>
              <ol className="steps">
                {[
                  "Create an account",
                  "Upload resume + save profile fields",
                  "Open a job and apply in-site",
                  "Score / forge when Resume_forge is running",
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
          <p className="brand">JobRight</p>
          <p>Jobs · profile · in-site apply</p>
        </div>
      </footer>
    </main>
  );
}
