import type { DayStat } from "../types";

// ── Layout constants ──────────────────────────────────────────────────────

const CELL_SIZE = 13;
const CELL_GAP = 2;
const CELL_STEP = CELL_SIZE + CELL_GAP; // 15px per cell
const LABEL_LEFT = 28; // px reserved for Mon/Wed/Fri labels
const LABEL_TOP = 20;  // px reserved for month labels above grid

// ── Color scale ───────────────────────────────────────────────────────────
// Level 0 = empty (matches card background), Level 4 = brightest green
// (matches DailyHoursChart bar color #22c55e).

const COLORS: Record<0 | 1 | 2 | 3 | 4, string> = {
  0: "#1e293b",
  1: "#14532d",
  2: "#166534",
  3: "#15803d",
  4: "#22c55e",
};

const BORDER_COLOR = "#334155";

const DOW_LABELS = [
  { row: 1, label: "Mon" },
  { row: 3, label: "Wed" },
  { row: 5, label: "Fri" },
];

// ── Pure helpers (exported for unit testing) ──────────────────────────────

export function minutesToLevel(minutes: number): 0 | 1 | 2 | 3 | 4 {
  if (minutes === 0) return 0;
  if (minutes <= 60) return 1;
  if (minutes <= 180) return 2;
  if (minutes <= 360) return 3;
  return 4;
}

// Returns the Sunday that is at or before (today - 364 days).
// The returned date is a plain local Date; the .getDay() is guaranteed to be 0.
export function computeGridStart(today: Date): Date {
  const d = new Date(today);
  d.setDate(d.getDate() - 364);
  d.setDate(d.getDate() - d.getDay()); // walk back to preceding Sunday
  return d;
}

// Maps a YYYY-MM-DD string to a grid (col, row) coordinate relative to
// gridStartSunday. Parses as UTC noon to match the backend's addDays()
// convention, avoiding DST midnight-boundary drift.
export function dateToCell(
  dateStr: string,
  gridStartSunday: Date
): { col: number; row: number } | null {
  const [y, m, d] = dateStr.split("-").map(Number);
  const date = new Date(Date.UTC(y, m - 1, d, 12, 0, 0));
  // Also anchor gridStart to UTC noon for consistent arithmetic
  const startNoon = new Date(
    Date.UTC(
      gridStartSunday.getFullYear(),
      gridStartSunday.getMonth(),
      gridStartSunday.getDate(),
      12,
      0,
      0
    )
  );
  const msPerDay = 86_400_000;
  const dayOffset = Math.round((date.getTime() - startNoon.getTime()) / msPerDay);
  if (dayOffset < 0) return null;
  return { col: Math.floor(dayOffset / 7), row: dayOffset % 7 };
}

// Returns the list of month labels to render above the grid.
// Emits one label per column where the month changes.
export function computeMonthLabels(
  gridStartSunday: Date,
  totalCols: number
): Array<{ label: string; col: number }> {
  const labels: Array<{ label: string; col: number }> = [];
  let lastMonth = -1;
  for (let col = 0; col < totalCols; col++) {
    const colDate = new Date(gridStartSunday);
    colDate.setDate(colDate.getDate() + col * 7);
    const month = colDate.getMonth();
    if (month !== lastMonth) {
      labels.push({
        label: colDate.toLocaleString("en", { month: "short" }),
        col,
      });
      lastMonth = month;
    }
  }
  return labels;
}

// ── Component ─────────────────────────────────────────────────────────────

interface Props {
  data: DayStat[];
}

export function HeatmapChart({ data }: Props) {
  if (data.length === 0) {
    return (
      <p style={{ color: "#475569", textAlign: "center", padding: "32px 0" }}>
        No activity data for the past year.
      </p>
    );
  }

  const today = new Date();
  const todayStr = today.toISOString().slice(0, 10);
  const gridStart = computeGridStart(today);

  // Build a lookup: date string → minutes
  const minutesByDate = new Map<string, number>(
    data.map((d) => [d.date, d.minutes])
  );

  // How many columns do we need? (today's column + 1)
  const todayCell = dateToCell(todayStr, gridStart);
  const totalCols = (todayCell?.col ?? 52) + 1;

  const svgWidth = LABEL_LEFT + totalCols * CELL_STEP;
  const svgHeight = LABEL_TOP + 7 * CELL_STEP;

  const monthLabels = computeMonthLabels(gridStart, totalCols);

  // Build all visible cell data upfront
  const cells: Array<{
    key: string;
    x: number;
    y: number;
    fill: string;
    title: string;
  }> = [];

  for (let col = 0; col < totalCols; col++) {
    for (let row = 0; row < 7; row++) {
      const cellDate = new Date(gridStart);
      cellDate.setDate(cellDate.getDate() + col * 7 + row);
      const dateStr = cellDate.toISOString().slice(0, 10);

      // Skip future cells (partial last column beyond today)
      if (dateStr > todayStr) continue;

      const minutes = minutesByDate.get(dateStr) ?? 0;
      const level = minutesToLevel(minutes);
      const hours = (minutes / 60).toFixed(1);
      const title =
        minutes === 0
          ? `${dateStr} — no activity`
          : `${dateStr} — ${hours}h (${minutes}m)`;

      cells.push({
        key: dateStr,
        x: LABEL_LEFT + col * CELL_STEP,
        y: LABEL_TOP + row * CELL_STEP,
        fill: COLORS[level],
        title,
      });
    }
  }

  return (
    <div style={{ overflowX: "auto" }}>
      <svg
        width="100%"
        viewBox={`0 0 ${svgWidth} ${svgHeight}`}
        role="img"
        aria-label="Activity heatmap — past year"
        style={{ display: "block", minWidth: svgWidth }}
      >
        {/* Month labels */}
        {monthLabels.map(({ label, col }) => (
          <text
            key={`month-${label}-${col}`}
            x={LABEL_LEFT + col * CELL_STEP + CELL_SIZE / 2}
            y={LABEL_TOP - 6}
            fontSize={10}
            fill="#94a3b8"
            textAnchor="middle"
          >
            {label}
          </text>
        ))}

        {/* Day-of-week labels (Mon, Wed, Fri only) */}
        {DOW_LABELS.map(({ row, label }) => (
          <text
            key={label}
            x={LABEL_LEFT - 4}
            y={LABEL_TOP + row * CELL_STEP + CELL_SIZE - 2}
            fontSize={10}
            fill="#94a3b8"
            textAnchor="end"
          >
            {label}
          </text>
        ))}

        {/* Day cells */}
        {cells.map(({ key, x, y, fill, title }) => (
          <rect
            key={key}
            x={x}
            y={y}
            width={CELL_SIZE}
            height={CELL_SIZE}
            rx={2}
            fill={fill}
            stroke={BORDER_COLOR}
            strokeWidth={0.5}
            aria-label={title}
          >
            <title>{title}</title>
          </rect>
        ))}
      </svg>
    </div>
  );
}
