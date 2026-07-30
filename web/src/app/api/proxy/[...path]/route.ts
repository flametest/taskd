import { NextRequest, NextResponse } from "next/server";
import { getTargetEnv, taskdBase } from "../../../../../registry";

// Proxies browser requests to the taskd backend (REST). Keeps the backend URL
// server-side (avoids CORS) and centralizes where an auth header would be added
// if taskd ever introduces authentication. No auth is injected today.
async function handler(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  const { path } = await params;
  const searchParams = request.nextUrl.searchParams.toString();
  const base = taskdBase[getTargetEnv()];
  const targetUrl = `${base}/${path.join("/")}${searchParams ? `?${searchParams}` : ""}`;

  const response = await fetch(targetUrl, {
    method: request.method,
    headers: request.headers,
    body: request.body,
    // @ts-expect-error - duplex is required for streaming request bodies
    duplex: "half",
  });

  // fetch() auto-decompresses the body, so strip encoding headers.
  const headers = new Headers(response.headers);
  headers.delete("content-encoding");
  headers.delete("content-length");
  headers.delete("transfer-encoding");

  return new Response(response.body, { status: response.status, headers });
}

export { handler as GET, handler as POST, handler as PUT, handler as DELETE };
