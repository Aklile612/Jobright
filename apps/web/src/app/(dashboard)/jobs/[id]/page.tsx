import Link from "next/link";
import { notFound } from "next/navigation";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { resolveJob } from "@/lib/jobs/resolveJob";

export default async function JobDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const job = await resolveJob(id);
  if (!job) notFound();

  return (
    <main>
      <SiteHeader solid />
      <article className="container section">
        <p className="badge">Role</p>
        <h1 className="page-title" style={{ maxWidth: "40rem", marginTop: "0.75rem" }}>
          {job.title}
        </h1>
        <p className="section-sub" style={{ fontSize: "1.05rem", fontWeight: 600 }}>
          {job.company} · {job.location || "Remote"}
          {job.salary_range ? ` · ${job.salary_range}` : ""}
        </p>
        <div className="hero__cta" style={{ marginTop: "1.5rem" }}>
          <Link href={`/jobs/${job.id}/apply`} className="btn btn-primary">
            Apply in-site
          </Link>
          <a href={job.source_url} target="_blank" rel="noreferrer" className="btn btn-ghost">
            Original listing
          </a>
        </div>
        <div className="surface" style={{ marginTop: "2rem", borderRadius: 16, padding: "1.5rem" }}>
          <h2 className="section-title" style={{ fontSize: "1.35rem" }}>
            Description
          </h2>
          <p className="section-sub" style={{ maxWidth: "none", whiteSpace: "pre-wrap", marginTop: "1rem" }}>
            {job.description}
          </p>
        </div>
      </article>
    </main>
  );
}
