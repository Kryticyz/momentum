import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { WeekStat } from "../types";

interface Props {
  data: WeekStat[];
}

export function WeeklyTrendChart({ data }: Props) {
  if (data.length === 0) {
    return (
      <p style={{ color: "#475569", textAlign: "center", padding: "32px 0" }}>
        No data for selected range.
      </p>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={280}>
      <LineChart data={data} margin={{ top: 4, right: 16, bottom: 4, left: 8 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
        <XAxis
          dataKey="weekStart"
          tickFormatter={(d: string) => d.slice(5)} // MM-DD
          tick={{ fill: "#94a3b8", fontSize: 11 }}
        />
        <YAxis
          tickFormatter={(v: number) => `${(v / 60).toFixed(1)}h`}
          tick={{ fill: "#94a3b8", fontSize: 12 }}
        />
        <Tooltip
          contentStyle={{ background: "#1e293b", border: "1px solid #334155" }}
          labelStyle={{ color: "#e2e8f0" }}
          formatter={(value: number) => [
            `${value}m (${(value / 60).toFixed(1)}h)`,
            "Week total",
          ]}
        />
        <Line
          type="monotone"
          dataKey="minutes"
          stroke="#f59e0b"
          dot={{ fill: "#f59e0b", r: 4 }}
          strokeWidth={2}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}
