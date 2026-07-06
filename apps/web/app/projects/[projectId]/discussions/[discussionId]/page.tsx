import { ProjectDiscussionDetailPage } from "./ProjectDiscussionDetailPage";

type PageProps = { params: Promise<{ projectId: string; discussionId: string }> };

export default async function DiscussionDetailRoute({ params }: PageProps) {
  const { projectId, discussionId } = await params;
  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <ProjectDiscussionDetailPage projectId={projectId} discussionId={discussionId} />
    </main>
  );
}
