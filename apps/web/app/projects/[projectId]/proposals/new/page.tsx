import { NewProjectProposalPage } from "./NewProjectProposalPage";

type PageProps = { params: Promise<{ projectId: string }> };

export default async function NewProposalRoute({ params }: PageProps) {
  const { projectId } = await params;
  return <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}><NewProjectProposalPage projectId={projectId} /></main>;
}
