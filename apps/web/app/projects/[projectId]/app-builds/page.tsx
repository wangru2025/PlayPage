import { AppBuildsPage } from "./AppBuildsPage";

export default async function Page({ params }: { params: Promise<{ projectId: string }> }) {
  const { projectId } = await params;
  return <AppBuildsPage projectId={projectId} />;
}