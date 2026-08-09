import { SiteHeader } from "@/components/layout/SiteHeader";
import { JobFeed } from "@/features/jobs/JobFeed";
import { loadJobsPage } from "@/lib/jobs/loadJobs";

export const dynamic = "force-dynamic";

export default async function JobsPage() {
  const page = await loadJobsPage("", 24, 0, false);
  // If DB empty, try sync once.
  const data =
    page.total > 0 || page.items.length > 0
      ? page
      : await loadJobsPage("", 24, 0, true);

  return (
    <main>
      <SiteHeader solid />
      <JobFeed
        initialJobs={data.items}
        initialTotal={data.total}
        paginated
        title="All software roles"
        subtitle="Synced from Remotive, Arbeitnow, RemoteOK, The Muse, Jobspresso (+ Adzuna/JSearch when keys are set). Use Load more to browse the full catalog."
      />
    </main>
  );
}
