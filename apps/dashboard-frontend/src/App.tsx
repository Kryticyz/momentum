import { useState } from "react";
import { DateRangePicker } from "./components/DateRangePicker";
import { DailyHoursChart } from "./components/DailyHoursChart";
import { ProjectBreakdown } from "./components/ProjectBreakdown";
import { RefreshButton } from "./components/RefreshButton";
import { WeeklyTrendChart } from "./components/WeeklyTrendChart";
import { useTimeData } from "./hooks/useTimeData";
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
        <h1 style={{ fontSize: 22, fontWeight: 700, color: "#f1f5f9" }}>
          Project Insights
        </h1>
        <div style={{ display: "flex", flexWrap: "wrap", gap: 16, alignItems: "center" }}>
          <DateRangePicker range={range} onChange={setRange} />
          <RefreshButton onRefresh={() => setRefreshKey((k) => k + 1)} />
        </div>
      </header>

      {error && (
        <div
          style={{
            background: "#450a0a",
            border: "1px solid #7f1d1d",
            borderRadius: 8,
            padding: "12px 16px",
            color: "#fca5a5",
            marginBottom: 24,
            fontSize: 13,
          }}
        >
          {error}
        </div>
      )}

      {loading && (
        <p style={{ color: "#64748b", textAlign: "center", padding: "48px 0" }}>
          Loading…
        </p>
      )}

      {!loading && (
        <div style={{ display: "flex", flexDirection: "column", gap: 40 }}>
          <Section title="Project Breakdown">
            <ProjectBreakdown data={projects} />
          </Section>
          <Section title="Daily Hours">
            <DailyHoursChart data={days} />
          </Section>
          <Section title="Weekly Trend">
            <WeeklyTrendChart data={weeks} />
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
          color: "#94a3b8",
          textTransform: "uppercase",
          letterSpacing: "0.06em",
          marginBottom: 16,
        }}
      >
        {title}
      </h2>
      <div
        style={{
          background: "#1e293b",
          borderRadius: 10,
          padding: "20px 16px",
          border: "1px solid #334155",
        }}
      >
        {children}
      </div>
    </section>
  );
}
