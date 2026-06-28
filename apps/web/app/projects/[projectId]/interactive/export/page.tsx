import { InteractiveDataExportPage } from "./InteractiveDataExportPage";

type PageProps = {
  params: Promise<{
    projectId: string;
  }>;
};

export default async function InteractiveExportRoute({ params }: PageProps) {
  const { projectId } = await params;

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <InteractiveDataExportPage projectId={projectId} />
    </main>
  );
}
