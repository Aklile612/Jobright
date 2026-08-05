import { SiteHeader } from "@/components/layout/SiteHeader";
import { JobFeed } from "@/features/jobs/JobFeed";
import { loadSoftwareJobs } from "@/lib/jobs/loadJobs";

export const dynamic = "force-dynamic";

export default async function JobsPage() {
  const jobs = await loadSoftwareJobs();
  return (
    <main>
      <SiteHeader solid />
      <JobFeed
        initialJobs={jobs}
        title="All software roles"
        subtitle="Synced from Remotive, Arbeitnow, RemoteOK, plus anything you scrape into JobRight."
      />
    </main>
  );
}
