import { ProjectStatsPage } from "./ProjectStatsPage";

type PageProps = {
  params: Promise<{
    projectId: string;
  }>;
};

export default async function StatsPage({ params }: PageProps) {
  const { projectId } = await params;

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <ProjectStatsPage projectId={projectId} />
    </main>
  );
}
