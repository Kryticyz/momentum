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
    let cancelled = false;
    setData((prev) => ({ ...prev, loading: true, error: null }));

    Promise.all([
      fetchProjects(range),
      fetchDays(range),
      fetchWeeks(range),
    ])
      .then(([projects, days, weeks]) => {
        if (!cancelled) {
          setData({ projects, days, weeks, loading: false, error: null });
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : String(err);
          setData({ projects: [], days: [], weeks: [], loading: false, error: message });
        }
      });

    return () => {
      cancelled = true;
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range.from, range.to, refreshKey]);

  return data;
}
