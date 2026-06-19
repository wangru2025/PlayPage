import { NextResponse } from "next/server";
import fs from "node:fs/promises";
import path from "node:path";

export const dynamic = "force-dynamic";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const name = url.searchParams.get("name");
  if (!name || (name !== "wei.jpg" && name !== "zhifubao.jpg")) {
    return new NextResponse("没有找到这个二维码", { status: 404 });
  }

  const filePath = path.join(process.cwd(), "..", "..", "..", "qrcode", name);
  const data = await fs.readFile(filePath);
  return new NextResponse(data, {
    status: 200,
    headers: {
      "Content-Type": "image/jpeg",
      "Cache-Control": "public, max-age=3600"
    }
  });
}
