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
            Less noise. More shipped applications — tailor your resume to each
            role, copy a cover letter, and apply on the employer site.
          </p>
          <div className="hero__cta">
            <Link href="/jobs" className="btn btn-primary">
              Browse jobs
            </Link>
            <Link href="/profile" className="btn btn-ghost">
              Upload resume
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
                <h2 className="section-title">Resume → tailor → apply</h2>
                <p className="section-sub">
                  Upload once. For each job we show ATS gaps, weave in missing
                  keywords, and give you a downloadable resume plus cover letter.
                </p>
              </div>
              <ol className="steps">
                {[
                  "Create an account",
                  "Upload your resume on Profile",
                  "Open a job → Prepare resume & letter",
                  "Download / copy, then apply on their site",
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
          <p>Jobs · ATS tailor · cover letters</p>
        </div>
      </footer>
    </main>
  );
}
