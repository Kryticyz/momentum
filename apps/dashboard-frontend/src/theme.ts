const colors = {
  // Surfaces
  cardBg: "#1e293b",
  cardBorder: "#334155",

  // Text
  heading: "#f1f5f9",
  sectionTitle: "#94a3b8",
  body: "#e2e8f0",
  muted: "#64748b",
  emptyState: "#475569",
  label: "#94a3b8",

  // Form inputs
  inputBg: "#1e293b",
  inputBorder: "#334155",
  inputText: "#e2e8f0",

  // Buttons
  buttonPrimary: "#3b82f6",
  buttonDisabled: "#334155",
  buttonText: "#ffffff",

  // Error states
  errorBg: "#450a0a",
  errorBorder: "#7f1d1d",
  errorText: "#fca5a5",
  errorInline: "#f87171",
  errorBoundaryText: "#f87171",

  // Chart shared
  gridStroke: "#1e293b",
  axisTick: "#94a3b8",
  axisLabel: "#e2e8f0",
  tooltipBg: "#1e293b",
  tooltipBorder: "#334155",
  tooltipLabel: "#e2e8f0",

  // Chart accent colors
  barProject: "#6366f1",
  barDaily: "#22c55e",
  lineWeekly: "#f59e0b",

  // Heatmap
  heatmap: {
    0: "#1e293b",
    1: "#14532d",
    2: "#166534",
    3: "#15803d",
    4: "#22c55e",
  } as Record<0 | 1 | 2 | 3 | 4, string>,
  heatmapBorder: "#334155",
} as const;

export const theme = { colors } as const;

export type Theme = typeof theme;
