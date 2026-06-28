import { ProjectReleaseHistoryPage } from "./ProjectReleaseHistoryPage";

type PageProps = {
  params: Promise<{ projectId: string }>;
};

export default async function ReleasesRoute({ params }: PageProps) {
  const { projectId } = await params;
  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <ProjectReleaseHistoryPage projectId={projectId} />
    </main>
  );
}
