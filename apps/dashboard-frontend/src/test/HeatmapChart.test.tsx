import { render, screen } from "@testing-library/react";
import {
  minutesToLevel,
  computeGridStart,
  dateToCell,
  computeMonthLabels,
  HeatmapChart,
} from "../components/HeatmapChart";

// ── minutesToLevel ──────────────────────────────────────────────────────────

describe("minutesToLevel", () => {
  it("returns 0 for 0 minutes", () => {
    expect(minutesToLevel(0)).toBe(0);
  });
  it("returns 1 for 1 minute", () => {
    expect(minutesToLevel(1)).toBe(1);
  });
  it("returns 1 for exactly 60 minutes", () => {
    expect(minutesToLevel(60)).toBe(1);
  });
  it("returns 2 for 61 minutes", () => {
    expect(minutesToLevel(61)).toBe(2);
  });
  it("returns 2 for exactly 180 minutes", () => {
    expect(minutesToLevel(180)).toBe(2);
  });
  it("returns 3 for 181 minutes", () => {
    expect(minutesToLevel(181)).toBe(3);
  });
  it("returns 3 for exactly 360 minutes", () => {
    expect(minutesToLevel(360)).toBe(3);
  });
  it("returns 4 for 361 minutes", () => {
    expect(minutesToLevel(361)).toBe(4);
  });
  it("returns 4 for large values", () => {
    expect(minutesToLevel(999)).toBe(4);
  });
});

// ── computeGridStart ────────────────────────────────────────────────────────

describe("computeGridStart", () => {
  it("always returns a Sunday (getDay() === 0)", () => {
    const today = new Date("2026-02-18"); // Wednesday
    const start = computeGridStart(today);
    expect(start.getDay()).toBe(0);
  });

  it("returned date is at least 364 days before today", () => {
    const today = new Date("2026-02-18");
    const start = computeGridStart(today);
    const diffDays = Math.round(
      (today.getTime() - start.getTime()) / 86_400_000
    );
    expect(diffDays).toBeGreaterThanOrEqual(364);
  });

  it("does not mutate the passed-in date", () => {
    const today = new Date("2026-02-18");
    const originalTime = today.getTime();
    computeGridStart(today);
    expect(today.getTime()).toBe(originalTime);
  });
});

// ── dateToCell ──────────────────────────────────────────────────────────────

describe("dateToCell", () => {
  // 2025-01-05 is a Sunday — use as a fixed grid start
  const gridStart = new Date(2025, 0, 5); // local time

  it("maps the grid-start Sunday to col=0, row=0", () => {
    expect(dateToCell("2025-01-05", gridStart)).toEqual({ col: 0, row: 0 });
  });

  it("maps the Monday of the first week to col=0, row=1", () => {
    expect(dateToCell("2025-01-06", gridStart)).toEqual({ col: 0, row: 1 });
  });

  it("maps the Saturday of the first week to col=0, row=6", () => {
    expect(dateToCell("2025-01-11", gridStart)).toEqual({ col: 0, row: 6 });
  });

  it("maps the next Sunday to col=1, row=0", () => {
    expect(dateToCell("2025-01-12", gridStart)).toEqual({ col: 1, row: 0 });
  });

  it("returns null for a date before the grid start", () => {
    expect(dateToCell("2025-01-04", gridStart)).toBeNull();
  });
});

// ── computeMonthLabels ──────────────────────────────────────────────────────

describe("computeMonthLabels", () => {
  it("emits a label at col 0 when the grid starts in a new month", () => {
    const gridStart = new Date(2026, 0, 4); // Jan 4, 2026 (Sunday)
    const labels = computeMonthLabels(gridStart, 5);
    expect(labels[0].label).toBe("Jan");
    expect(labels[0].col).toBe(0);
  });

  it("never produces duplicate consecutive labels", () => {
    const gridStart = new Date(2026, 0, 4);
    const labels = computeMonthLabels(gridStart, 10);
    for (let i = 1; i < labels.length; i++) {
      expect(labels[i].label).not.toBe(labels[i - 1].label);
    }
  });

  it("produces at most 13 labels for 53 columns (a year can span 13 months)", () => {
    const gridStart = new Date(2025, 0, 5); // Jan 5, 2025
    const labels = computeMonthLabels(gridStart, 53);
    expect(labels.length).toBeLessThanOrEqual(13);
  });
});

// ── HeatmapChart component ──────────────────────────────────────────────────

describe("HeatmapChart", () => {
  it("shows empty state when data is empty", () => {
    render(<HeatmapChart data={[]} />);
    expect(screen.getByText(/No activity data for the past year/)).toBeInTheDocument();
  });

  it("renders an SVG when data is provided", () => {
    const data = [
      { date: "2026-02-18", minutes: 120, hours: 2.0 },
      { date: "2026-02-17", minutes: 0, hours: 0.0 },
    ];
    const { container } = render(<HeatmapChart data={data} />);
    expect(container.querySelector("svg")).not.toBeNull();
  });

  it("renders rect elements for day cells", () => {
    const data = [{ date: "2026-02-18", minutes: 120, hours: 2.0 }];
    const { container } = render(<HeatmapChart data={data} />);
    const rects = container.querySelectorAll("rect");
    expect(rects.length).toBeGreaterThan(0);
  });

  it("renders the SVG with role=img for accessibility", () => {
    const data = [{ date: "2026-02-18", minutes: 90, hours: 1.5 }];
    render(<HeatmapChart data={data} />);
    expect(screen.getByRole("img")).toBeInTheDocument();
  });
});
