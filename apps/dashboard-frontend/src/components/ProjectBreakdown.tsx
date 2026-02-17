import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { ProjectStat } from "../types";

interface Props {
  data: ProjectStat[];
}

export function ProjectBreakdown({ data }: Props) {
  if (data.length === 0) {
    return <EmptyState />;
  }

  const height = Math.max(200, data.length * 44);

  return (
    <ResponsiveContainer width="100%" height={height}>
      <BarChart
        layout="vertical"
        data={data}
        margin={{ top: 4, right: 32, bottom: 4, left: 8 }}
      >
        <CartesianGrid strokeDasharray="3 3" stroke="#1e293b" />
        <XAxis
          type="number"
          tickFormatter={(v: number) => `${(v / 60).toFixed(1)}h`}
          tick={{ fill: "#94a3b8", fontSize: 12 }}
        />
        <YAxis
          type="category"
          dataKey="project"
          width={150}
          tick={{ fill: "#e2e8f0", fontSize: 12 }}
        />
        <Tooltip
          contentStyle={{ background: "#1e293b", border: "1px solid #334155" }}
          labelStyle={{ color: "#e2e8f0" }}
          formatter={(value: number) => [
            `${value}m (${(value / 60).toFixed(1)}h)`,
            "Time",
          ]}
        />
        <Bar dataKey="minutes" fill="#6366f1" radius={[0, 4, 4, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

function EmptyState() {
  return (
    <p style={{ color: "#475569", textAlign: "center", padding: "32px 0" }}>
      No data for selected range.
    </p>
  );
}
