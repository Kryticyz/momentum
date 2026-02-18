import { useEffect, useState } from "react";
import { fetchDays, fetchProjects, fetchWeeks } from "../api";
import type { DateRange, DayStat, ProjectStat, WeekStat } from "../types";

export interface TimeData {
  projects: ProjectStat[];
  days: DayStat[];
  weeks: WeekStat[];
  loading: boolean;
  error: string | null;
}

export function useTimeData(range: DateRange, refreshKey: number): TimeData {
  const [data, setData] = useState<TimeData>({
    projects: [],
    days: [],
    weeks: [],
    loading: true,
    error: null,
  });

  useEffect(() => {
    const controller = new AbortController();
    const { signal } = controller;
    setData((prev) => ({ ...prev, loading: true, error: null }));

    Promise.all([
      fetchProjects(range, signal),
      fetchDays(range, signal),
      fetchWeeks(range, signal),
    ])
      .then(([projects, days, weeks]) => {
        if (!signal.aborted) {
          setData({ projects, days, weeks, loading: false, error: null });
        }
      })
      .catch((err: unknown) => {
        if (signal.aborted) return;
        const message = err instanceof Error ? err.message : String(err);
        setData({ projects: [], days: [], weeks: [], loading: false, error: message });
      });

    return () => {
      controller.abort();
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range.from, range.to, refreshKey]);

  return data;
}
