import { render, screen } from "@testing-library/react";
import { ProjectBreakdown } from "../components/ProjectBreakdown";
import { DailyHoursChart } from "../components/DailyHoursChart";
import { WeeklyTrendChart } from "../components/WeeklyTrendChart";

// Recharts uses ResizeObserver and SVG internals not available in jsdom.
// We mock ResizeObserver to avoid test failures.
class MockResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver;

// --- ProjectBreakdown ---

describe("ProjectBreakdown", () => {
  it("shows empty state when data is empty", () => {
    render(<ProjectBreakdown data={[]} />);
    expect(screen.getByText(/No data for selected range/)).toBeInTheDocument();
  });

  it("renders without crashing with data", () => {
    const data = [
      { project: "Alpha", minutes: 90, hours: 1.5 },
      { project: "Beta", minutes: 60, hours: 1.0 },
    ];
    const { container } = render(<ProjectBreakdown data={data} />);
    expect(container.firstChild).not.toBeNull();
  });
});

// --- DailyHoursChart ---

describe("DailyHoursChart", () => {
  it("shows empty state when data is empty", () => {
    render(<DailyHoursChart data={[]} />);
    expect(screen.getByText(/No data for selected range/)).toBeInTheDocument();
  });

  it("renders without crashing with data", () => {
    const data = [
      { date: "2026-02-01", minutes: 60, hours: 1.0 },
      { date: "2026-02-02", minutes: 0, hours: 0.0 },
    ];
    const { container } = render(<DailyHoursChart data={data} />);
    expect(container.firstChild).not.toBeNull();
  });
});

// --- WeeklyTrendChart ---

describe("WeeklyTrendChart", () => {
  it("shows empty state when data is empty", () => {
    render(<WeeklyTrendChart data={[]} />);
    expect(screen.getByText(/No data for selected range/)).toBeInTheDocument();
  });

  it("renders without crashing with data", () => {
    const data = [
      { weekStart: "2026-02-08", minutes: 300, hours: 5.0 },
      { weekStart: "2026-02-15", minutes: 420, hours: 7.0 },
    ];
    const { container } = render(<WeeklyTrendChart data={data} />);
    expect(container.firstChild).not.toBeNull();
  });
});
