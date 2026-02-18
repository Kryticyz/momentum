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
        <CartesianGrid strokeDasharray="3 3" stroke={theme.colors.gridStroke} />
        <XAxis
          type="number"
          tickFormatter={(v: number) => `${(v / 60).toFixed(1)}h`}
          tick={{ fill: theme.colors.axisTick, fontSize: 12 }}
        />
        <YAxis
          type="category"
          dataKey="project"
          width={150}
          tick={{ fill: theme.colors.axisLabel, fontSize: 12 }}
        />
        <Tooltip
          contentStyle={{ background: theme.colors.tooltipBg, border: `1px solid ${theme.colors.tooltipBorder}` }}
          labelStyle={{ color: theme.colors.tooltipLabel }}
          formatter={(value: number) => [
            `${value}m (${(value / 60).toFixed(1)}h)`,
            "Time",
          ]}
        />
        <Bar dataKey="minutes" fill={theme.colors.barProject} radius={[0, 4, 4, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

function EmptyState() {
  return (
    <p style={{ color: theme.colors.emptyState, textAlign: "center", padding: "32px 0" }}>
      No data for selected range.
    </p>
  );
}
