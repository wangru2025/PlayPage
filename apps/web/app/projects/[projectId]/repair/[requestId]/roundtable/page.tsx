import { RoundtableLogPage } from "./RoundtableLogPage";

type PageProps = {
  params: Promise<{
    projectId: string;
    requestId: string;
  }>;
};

export default async function RoundtableLogRoute({ params }: PageProps) {
  const { projectId, requestId } = await params;

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <RoundtableLogPage projectId={projectId} requestId={requestId} />
    </main>
  );
}
