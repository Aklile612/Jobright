import { SiteHeader } from "@/components/layout/SiteHeader";
import { ApplyWorkspace } from "@/features/apply/ApplyWorkspace";
import { resolveJob } from "@/lib/jobs/resolveJob";
import { notFound } from "next/navigation";

export default async function ApplyPage({
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
      <ApplyWorkspace job={job} />
    </main>
  );
}
