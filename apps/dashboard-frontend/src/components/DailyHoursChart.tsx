import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { theme } from "../theme";
import type { DayStat } from "../types";

interface Props {
  data: DayStat[];
}

export function DailyHoursChart({ data }: Props) {
  if (data.length === 0) {
    return (
      <p style={{ color: theme.colors.emptyState, textAlign: "center", padding: "32px 0" }}>
        No data for selected range.
      </p>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={data} margin={{ top: 4, right: 16, bottom: 4, left: 8 }}>
        <CartesianGrid strokeDasharray="3 3" stroke={theme.colors.gridStroke} />
        <XAxis
          dataKey="date"
          tickFormatter={(d: string) => d.slice(5)} // MM-DD
          tick={{ fill: theme.colors.axisTick, fontSize: 11 }}
          interval="preserveStartEnd"
        />
        <YAxis
          tickFormatter={(v: number) => `${(v / 60).toFixed(1)}h`}
          tick={{ fill: theme.colors.axisTick, fontSize: 12 }}
        />
        <Tooltip
          contentStyle={{ background: theme.colors.tooltipBg, border: `1px solid ${theme.colors.tooltipBorder}` }}
          labelStyle={{ color: theme.colors.tooltipLabel }}
          formatter={(value: number) => [`${value}m`, "Minutes"]}
        />
        <Bar dataKey="minutes" fill={theme.colors.barDaily} radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
