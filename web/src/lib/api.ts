import type {
  CreateTaskReq,
  ListTaskRecordsResp,
  ListTasksResp,
  Task,
} from "@/types/task";

// All calls go through the Next.js proxy route -> taskd backend, so the browser
// stays same-origin with the frontend and never sees the taskd URL directly.
const BASE = "/api/proxy/v1";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`${res.status}: ${text}`);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

function withQuery(params: Record<string, string | number | undefined>): string {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") q.set(k, String(v));
  }
  const s = q.toString();
  return s ? `?${s}` : "";
}

export const api = {
  listTasks: (params: {
    status?: string;
    search?: string;
    created_from?: string;
    created_to?: string;
    limit?: number;
    offset?: number;
  }) => request<ListTasksResp>(`/tasks${withQuery(params)}`),
  getTask: (id: string) => request<Task>(`/tasks/${id}`),
  createTask: (body: CreateTaskReq) =>
    request<Task>(`/tasks`, { method: "POST", body: JSON.stringify(body) }),
  cancelTask: (id: string) =>
    request<void>(`/tasks/${id}/cancel`, { method: "POST" }),
  reactivateTask: (id: string, execTime?: number) =>
    request<void>(`/tasks/${id}/reactivate`, {
      method: "POST",
      body: JSON.stringify({ body: execTime ? { exec_time: execTime } : {} }),
    }),
  listRecords: (id: string, limit = 20, offset = 0) =>
    request<ListTaskRecordsResp>(
      `/tasks/${id}/records${withQuery({ limit, offset })}`,
    ),
};
