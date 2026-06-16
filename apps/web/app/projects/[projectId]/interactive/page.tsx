import { InteractiveWorkspace } from "./InteractiveWorkspace";

type PageProps = {
  params: Promise<{
    projectId: string;
  }>;
};

export default async function InteractivePage({ params }: PageProps) {
  const { projectId } = await params;

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <InteractiveWorkspace projectId={projectId} />
    </main>
  );
}
