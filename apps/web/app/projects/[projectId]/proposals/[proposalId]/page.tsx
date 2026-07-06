import { ProjectProposalDetailPage } from "./ProjectProposalDetailPage";

type PageProps = { params: Promise<{ projectId: string; proposalId: string }> };

export default async function ProposalDetailRoute({ params }: PageProps) {
  const { projectId, proposalId } = await params;
  return <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}><ProjectProposalDetailPage projectId={projectId} proposalId={proposalId} /></main>;
}
