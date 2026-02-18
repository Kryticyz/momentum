import { useState } from "react";
import { ChartErrorBoundary } from "./components/ChartErrorBoundary";
import { DateRangePicker } from "./components/DateRangePicker";
import { DailyHoursChart } from "./components/DailyHoursChart";
import { HeatmapChart } from "./components/HeatmapChart";
import { ProjectBreakdown } from "./components/ProjectBreakdown";
import { RefreshButton } from "./components/RefreshButton";
import { WeeklyTrendChart } from "./components/WeeklyTrendChart";
import { useHeatmapData } from "./hooks/useHeatmapData";
import { useTimeData } from "./hooks/useTimeData";
import { theme } from "./theme";
import type { DateRange } from "./types";

function defaultRange(): DateRange {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 30);
  return {
    from: from.toISOString().slice(0, 10),
    to: to.toISOString().slice(0, 10),
  };
}

export function App() {
  const [range, setRange] = useState<DateRange>(defaultRange);
  const [refreshKey, setRefreshKey] = useState(0);
  const { projects, days, weeks, loading, error } = useTimeData(range, refreshKey);
  const { days: heatmapDays, loading: heatmapLoading } = useHeatmapData(refreshKey);

  return (
    <div style={{ maxWidth: 1100, margin: "0 auto", padding: "24px 16px" }}>
      <header
        style={{
          display: "flex",
          flexWrap: "wrap",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 16,
          marginBottom: 32,
        }}
      >
        <h1 style={{ fontSize: 22, fontWeight: 700, color: theme.colors.heading }}>
          Momentum
        </h1>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 16, alignItems: "center" }}>
          <DateRangePicker range={range} onChange={setRange} />
          <RefreshButton onRefresh={() => setRefreshKey((k) => k + 1)} />
        </div>
      </header>

      {error && (
        <div
          style={{
            background: theme.colors.errorBg,
            border: `1px solid ${theme.colors.errorBorder}`,
            borderRadius: 8,
            padding: "12px 16px",
            color: theme.colors.errorText,
            marginBottom: 24,
            fontSize: 13,
          }}
        >
          {error}
        </div>
      )}

      {!heatmapLoading && (
        <div style={{ marginBottom: 40 }}>
          <Section title="Activity — Past Year">
            <ChartErrorBoundary>
              <HeatmapChart data={heatmapDays} />
            </ChartErrorBoundary>
          </Section>
        </div>
      )}

      {loading && (
        <p style={{ color: theme.colors.muted, textAlign: "center", padding: "48px 0" }}>
          Loading…
        </p>
      )}

      {!loading && (
        <div style={{ display: "flex", flexDirection: "column", gap: 40 }}>
          <Section title="Project Breakdown">
            <ChartErrorBoundary>
              <ProjectBreakdown data={projects} />
            </ChartErrorBoundary>
          </Section>
          <Section title="Daily Hours">
            <ChartErrorBoundary>
              <DailyHoursChart data={days} />
            </ChartErrorBoundary>
          </Section>
          <Section title="Weekly Trend">
            <ChartErrorBoundary>
              <WeeklyTrendChart data={weeks} />
            </ChartErrorBoundary>
          </Section>
        </div>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h2
        style={{
          fontSize: 15,
          fontWeight: 600,
          color: theme.colors.sectionTitle,
          textTransform: "uppercase",
          letterSpacing: "0.06em",
          marginBottom: 16,
        }}
      >
        {title}
      </h2>
      <div
        style={{
          background: theme.colors.cardBg,
          borderRadius: 10,
          padding: "20px 16px",
          border: `1px solid ${theme.colors.cardBorder}`,
        }}
      >
        {children}
      </div>
    </section>
  );
}
