import { useEffect, useState } from "react";
import { fetchDays } from "../api";
import type { DayStat } from "../types";

export interface HeatmapData {
  days: DayStat[];
  loading: boolean;
  error: string | null;
}

// Returns a fixed DateRange covering the last 52 weeks ending today.
// The start date is snapped back to the preceding Sunday so week columns
// are always complete and aligned.
function heatmapDateRange(): { from: string; to: string } {
  const to = new Date();
  const from = new Date(to);
  from.setDate(from.getDate() - 364);
  // Walk back to the preceding Sunday (0 = Sunday)
  from.setDate(from.getDate() - from.getDay());
  return {
    from: from.toISOString().slice(0, 10),
    to: to.toISOString().slice(0, 10),
  };
}

export function useHeatmapData(refreshKey: number): HeatmapData {
  const [data, setData] = useState<HeatmapData>({
    days: [],
    loading: true,
    error: null,
  });

  useEffect(() => {
    let cancelled = false;
    setData((prev) => ({ ...prev, loading: true, error: null }));

    const range = heatmapDateRange();
    fetchDays(range)
      .then((days) => {
        if (!cancelled) {
          setData({ days, loading: false, error: null });
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          const message = err instanceof Error ? err.message : String(err);
          setData({ days: [], loading: false, error: message });
        }
      });

    return () => {
      cancelled = true;
    };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshKey]);

  return data;
}
