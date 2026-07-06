import { NewProjectDiscussionPage } from "./NewProjectDiscussionPage";

type PageProps = { params: Promise<{ projectId: string }> };

export default async function NewDiscussionRoute({ params }: PageProps) {
  const { projectId } = await params;
  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <NewProjectDiscussionPage projectId={projectId} />
    </main>
  );
}
