import { ProjectDiscussionsPage } from "./ProjectDiscussionsPage";

type PageProps = { params: Promise<{ projectId: string }> };

export default async function DiscussionsRoute({ params }: PageProps) {
  const { projectId } = await params;
  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <ProjectDiscussionsPage projectId={projectId} />
    </main>
  );
}
