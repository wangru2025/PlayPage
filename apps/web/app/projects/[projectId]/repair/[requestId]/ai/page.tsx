import { AIroundtablePage } from "./AIroundtablePage";

type PageProps = {
  params: Promise<{
    projectId: string;
    requestId: string;
  }>;
  searchParams?: Promise<{
    autoStart?: string;
  }>;
};

export default async function AIRoundtableRoute({ params, searchParams }: PageProps) {
  const { projectId, requestId } = await params;
  const sp = searchParams ? await searchParams : {};

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <AIroundtablePage projectId={projectId} requestId={requestId} autoStart={sp.autoStart === "1"} />
    </main>
  );
}
