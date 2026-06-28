import { TemplateDetailPage } from "./TemplateDetailPage";

export default async function TemplatePage({ params }: { params: Promise<{ templateId: string }> }) {
  const { templateId } = await params;
  return (
    <main id="main-content" className="shell" style={{ padding: "32px 0 72px" }}>
      <TemplateDetailPage templateId={templateId} />
    </main>
  );
}
