import { ProjectProposalsPage } from "./ProjectProposalsPage";

type PageProps = { params: Promise<{ projectId: string }> };

export default async function ProposalsRoute({ params }: PageProps) {
  const { projectId } = await params;
  return <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}><ProjectProposalsPage projectId={projectId} /></main>;
}
