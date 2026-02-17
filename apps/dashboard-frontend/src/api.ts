import type { DateRange, DayStat, ProjectStat, WeekStat } from "./types";

const BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "";

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text}`);
  }
  return res.json() as Promise<T>;
}

function rangeParams(range: DateRange): string {
  return `from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}`;
}

export function fetchProjects(range: DateRange): Promise<ProjectStat[]> {
  return fetchJSON<ProjectStat[]>(`/api/projects?${rangeParams(range)}`);
}

export function fetchDays(range: DateRange): Promise<DayStat[]> {
  return fetchJSON<DayStat[]>(`/api/days?${rangeParams(range)}`);
}

export function fetchWeeks(range: DateRange): Promise<WeekStat[]> {
  return fetchJSON<WeekStat[]>(`/api/weeks?${rangeParams(range)}`);
}

export async function postRefresh(): Promise<void> {
  const res = await fetch(`${BASE}/refresh`, { method: "POST" });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Refresh failed: ${res.status} ${text}`);
  }
}
