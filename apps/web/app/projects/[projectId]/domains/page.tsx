import { ProjectDomainsPage } from "./ProjectDomainsPage";

type PageProps = {
  params: Promise<{
    projectId: string;
  }>;
};

export default async function ProjectDomainsRoute({ params }: PageProps) {
  const { projectId } = await params;

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <ProjectDomainsPage projectId={projectId} />
    </main>
  );
}
