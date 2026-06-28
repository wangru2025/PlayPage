const browserAPIBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";
const serverAPIBase = process.env.INTERNAL_API_BASE_URL ?? browserAPIBase;
const csrfCookieName = "web_wangru_csrf";
const csrfHeaderName = "X-CSRF-Token";
const csrfErrorCode = "csrf_token_invalid";

type RequestBody = BodyInit | null | undefined;

export function buildURL(path: string): string {
  const apiBase = typeof window === "undefined" ? serverAPIBase : browserAPIBase;
  if (!apiBase) {
    return path;
  }

  return `${apiBase}${path}`;
}


export function buildWebSocketURL(path: string): string {
  const apiBase = browserAPIBase;
  if (apiBase) {
    const url = new URL(path, apiBase);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    return url.toString();
  }
  if (typeof window === "undefined") {
    return path;
  }
  const url = new URL(path, window.location.origin);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

export function getCSRFToken(): string {
  if (typeof document === "undefined") {
    return "";
  }
  const prefix = `${csrfCookieName}=`;
  const item = document.cookie
    .split(";")
    .map((value) => value.trim())
    .find((value) => value.startsWith(prefix));
  if (!item) {
    return "";
  }
  return decodeURIComponent(item.slice(prefix.length));
}

function isMutating(method: string): boolean {
  return ["POST", "PUT", "PATCH", "DELETE"].includes(method.toUpperCase());
}

function withCSRF(headers: HeadersInit | undefined, method: string): HeadersInit | undefined {
  if (!isMutating(method)) {
    return headers;
  }
  const token = getCSRFToken();
  if (!token) {
    return headers;
  }
  return {
    ...(headers ?? {}),
    [csrfHeaderName]: token
  };
}

async function refreshCSRFToken(): Promise<void> {
  if (typeof window === "undefined") {
    return;
  }
  await fetch(buildURL("/api/v1/me"), {
    cache: "no-store",
    credentials: "include"
  }).catch(() => undefined);
}

async function parseError(response: Response): Promise<{ message: string; code: string }> {
  const payload = await response.json().catch(() => null);
  return {
    message: payload?.error ?? `请求失败，状态码：${response.status}`,
    code: payload?.code ?? ""
  };
}

async function requestWithOptionalRetry(path: string, init: RequestInit, retryOnCSRF: boolean): Promise<Response> {
  const method = init.method ?? "GET";
  const response = await fetch(buildURL(path), {
    ...init,
    credentials: "include",
    headers: withCSRF(init.headers, method)
  });
  if (!retryOnCSRF || response.status !== 403 || !isMutating(method)) {
    return response;
  }
  const error = await response.clone().json().catch(() => null);
  if (error?.code !== csrfErrorCode) {
    return response;
  }
  await refreshCSRFToken();
  return fetch(buildURL(path), {
    ...init,
    credentials: "include",
    headers: withCSRF(init.headers, method)
  });
}

export async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(buildURL(path), {
    cache: "no-store",
    credentials: "include"
  });

  if (!response.ok) {
    throw new Error(`请求失败，状态码：${response.status}`);
  }

  return response.json() as Promise<T>;
}

export async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await requestWithOptionalRetry(path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  }, true);

  if (!response.ok) {
    const { message } = await parseError(response);
    throw new Error(message);
  }

  return response.json() as Promise<T>;
}

export async function postBlob(path: string, body: unknown): Promise<{ blob: Blob; filename: string }> {
  const response = await requestWithOptionalRetry(path, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  }, true);

  if (!response.ok) {
    const { message } = await parseError(response);
    throw new Error(message);
  }

  return {
    blob: await response.blob(),
    filename: parseDownloadFilename(response.headers.get("Content-Disposition")) || "download"
  };
}

function parseDownloadFilename(disposition: string | null): string {
  if (!disposition) {
    return "";
  }
  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match) {
    try {
      return decodeURIComponent(utf8Match[1]);
    } catch {
      return utf8Match[1];
    }
  }
  const fallbackMatch = disposition.match(/filename="([^"]+)"/i) ?? disposition.match(/filename=([^;]+)/i);
  return fallbackMatch ? fallbackMatch[1].trim() : "";
}

export async function postForm<T>(path: string, body: FormData): Promise<T> {
  const response = await requestWithOptionalRetry(path, {
    method: "POST",
    body
  }, true);

  if (!response.ok) {
    const { message } = await parseError(response);
    throw new Error(message);
  }

  return response.json() as Promise<T>;
}

export async function patchJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await requestWithOptionalRetry(path, {
    method: "PATCH",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  }, true);

  if (!response.ok) {
    const { message } = await parseError(response);
    throw new Error(message);
  }

  return response.json() as Promise<T>;
}

export async function deleteJSON<T>(path: string): Promise<T> {
  const response = await requestWithOptionalRetry(path, {
    method: "DELETE"
  }, true);

  if (!response.ok) {
    const { message } = await parseError(response);
    throw new Error(message);
  }

  return response.json() as Promise<T>;
}

export async function reportClientError(payload: Record<string, unknown>): Promise<void> {
  try {
    await fetch(buildURL("/api/v1/client-errors"), {
      method: "POST",
      credentials: "include",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify(payload)
    });
  } catch {
    // Ignore reporting failure.
  }
}
