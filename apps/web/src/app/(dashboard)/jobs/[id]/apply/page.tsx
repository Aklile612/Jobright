import { SiteHeader } from "@/components/layout/SiteHeader";
import { PrepareWorkspace } from "@/features/apply/PrepareWorkspace";
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
      <PrepareWorkspace job={job} />
    </main>
  );
}
