import type { ApiResponse, DateRange, DayStat, ProjectStat, WeekStat } from "./types";

const BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "";
const API_KEY = (import.meta.env.VITE_API_KEY as string | undefined) ?? "";
const API_PREFIX = "/api/v1";

function authHeaders(): Record<string, string> {
  if (API_KEY) {
    return { Authorization: `Bearer ${API_KEY}` };
  }
  return {};
}

async function fetchJSON<T>(path: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { signal, headers: authHeaders() });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text}`);
  }
  const envelope = (await res.json()) as ApiResponse<T>;
  return envelope.data;
}

function rangeParams(range: DateRange): string {
  return `from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}`;
}

export function fetchProjects(range: DateRange, signal?: AbortSignal): Promise<ProjectStat[]> {
  return fetchJSON<ProjectStat[]>(`${API_PREFIX}/projects?${rangeParams(range)}`, signal);
}

export function fetchDays(range: DateRange, signal?: AbortSignal): Promise<DayStat[]> {
  return fetchJSON<DayStat[]>(`${API_PREFIX}/days?${rangeParams(range)}`, signal);
}

export function fetchWeeks(range: DateRange, signal?: AbortSignal): Promise<WeekStat[]> {
  return fetchJSON<WeekStat[]>(`${API_PREFIX}/weeks?${rangeParams(range)}`, signal);
}

export async function postRefresh(): Promise<void> {
  const res = await fetch(`${BASE}/refresh`, { method: "POST", headers: authHeaders() });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Refresh failed: ${res.status} ${text}`);
  }
}
