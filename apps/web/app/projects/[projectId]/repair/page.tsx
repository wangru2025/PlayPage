import { RepairRequestForm } from "./RepairRequestForm";

type PageProps = {
  params: Promise<{
    projectId: string;
  }>;
};

export default async function RepairRequestPage({ params }: PageProps) {
  const { projectId } = await params;

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <RepairRequestForm projectId={projectId} />
    </main>
  );
}
