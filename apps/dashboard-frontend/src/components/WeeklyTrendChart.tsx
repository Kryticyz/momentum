import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { theme } from "../theme";
import type { WeekStat } from "../types";

interface Props {
  data: WeekStat[];
}

export function WeeklyTrendChart({ data }: Props) {
  if (data.length === 0) {
    return (
      <p style={{ color: theme.colors.emptyState, textAlign: "center", padding: "32px 0" }}>
        No data for selected range.
      </p>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <LineChart data={data} margin={{ top: 4, right: 16, bottom: 4, left: 8 }}>
        <CartesianGrid strokeDasharray="3 3" stroke={theme.colors.gridStroke} />
        <XAxis
          dataKey="weekStart"
          tickFormatter={(d: string) => d.slice(5)} // MM-DD
          tick={{ fill: theme.colors.axisTick, fontSize: 11 }}
        />
        <YAxis
          tickFormatter={(v: number) => `${(v / 60).toFixed(1)}h`}
          tick={{ fill: theme.colors.axisTick, fontSize: 12 }}
        />
        <Tooltip
          contentStyle={{ background: theme.colors.tooltipBg, border: `1px solid ${theme.colors.tooltipBorder}` }}
          labelStyle={{ color: theme.colors.tooltipLabel }}
          formatter={(value: number) => [
            `${value}m (${(value / 60).toFixed(1)}h)`,
            "Week total",
          ]}
        />
        <Line
          type="monotone"
          dataKey="minutes"
          stroke={theme.colors.lineWeekly}
          dot={{ fill: theme.colors.lineWeekly, r: 4 }}
          strokeWidth={2}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}
