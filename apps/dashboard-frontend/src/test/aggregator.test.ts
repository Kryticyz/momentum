// Pure logic tests for date range utilities used in the frontend.

/**
 * defaultRange() logic — mirrors App.tsx defaultRange().
 * Computes a 30-day window ending today.
 */
function defaultRange() {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 30);
  return {
    from: from.toISOString().slice(0, 10),
    to: to.toISOString().slice(0, 10),
  };
}

describe("defaultRange", () => {
  it("from is 30 days before to", () => {
    const range = defaultRange();
    const fromDate = new Date(range.from);
    const toDate = new Date(range.to);
    const diffDays = Math.round((toDate.getTime() - fromDate.getTime()) / (1000 * 60 * 60 * 24));
    expect(diffDays).toBe(30);
  });

  it("produces valid YYYY-MM-DD strings", () => {
    const range = defaultRange();
    expect(range.from).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(range.to).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("from is always <= to", () => {
    const range = defaultRange();
    expect(range.from <= range.to).toBe(true);
  });
});

/**
 * rangeParams() — mirrors api.ts rangeParams().
 */
function rangeParams(range: { from: string; to: string }): string {
  return `from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}`;
}

describe("rangeParams", () => {
  it("produces correct query string", () => {
    const result = rangeParams({ from: "2026-02-01", to: "2026-02-28" });
    expect(result).toBe("from=2026-02-01&to=2026-02-28");
  });
});

/**
 * formatMinutes helper — used for tooltip formatting in charts.
 */
function formatMinutesTooltip(minutes: number): string {
  return `${minutes}m (${(minutes / 60).toFixed(1)}h)`;
}

describe("formatMinutesTooltip", () => {
  it("formats 60 minutes", () => {
    expect(formatMinutesTooltip(60)).toBe("60m (1.0h)");
  });
  it("formats 35 minutes", () => {
    expect(formatMinutesTooltip(35)).toBe("35m (0.6h)");
  });
  it("formats 90 minutes", () => {
    expect(formatMinutesTooltip(90)).toBe("90m (1.5h)");
  });
  it("formats 0 minutes", () => {
    expect(formatMinutesTooltip(0)).toBe("0m (0.0h)");
  });
});

/**
 * Date axis tick formatter — slices YYYY-MM-DD to MM-DD.
 */
function tickFormatter(d: string): string {
  return d.slice(5);
}

describe("tickFormatter", () => {
  it("returns MM-DD from YYYY-MM-DD", () => {
    expect(tickFormatter("2026-02-12")).toBe("02-12");
  });
  it("handles month boundary", () => {
    expect(tickFormatter("2026-01-01")).toBe("01-01");
  });
});
