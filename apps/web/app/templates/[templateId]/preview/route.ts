import { buildURL } from "@/lib/api";

export async function GET(_request: Request, { params }: { params: Promise<{ templateId: string }> }) {
  const { templateId } = await params;
  const response = await fetch(buildURL(`/api/v1/templates/${encodeURIComponent(templateId)}/preview`), {
    cache: "no-store"
  });

  const body = await response.text();
  return new Response(body, {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") ?? "text/html; charset=utf-8",
      "Cache-Control": "no-store"
    }
  });
}
