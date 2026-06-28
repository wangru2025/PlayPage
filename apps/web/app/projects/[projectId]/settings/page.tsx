import { ProjectSettingsPage } from "./ProjectSettingsPage";

type PageProps = {
  params: Promise<{
    projectId: string;
  }>;
};

export default async function SettingsRoute({ params }: PageProps) {
  const { projectId } = await params;

  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <ProjectSettingsPage projectId={projectId} />
    </main>
  );
}
