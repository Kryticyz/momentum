import type { DateRange } from "../types";

interface Props {
  range: DateRange;
  onChange: (range: DateRange) => void;
}

export function DateRangePicker({ range, onChange }: Props) {
  function handleFrom(e: React.ChangeEvent<HTMLInputElement>) {
    const from = e.target.value;
    if (from <= range.to) {
      onChange({ from, to: range.to });
    }
  }

  function handleTo(e: React.ChangeEvent<HTMLInputElement>) {
    const to = e.target.value;
    if (range.from <= to) {
      onChange({ from: range.from, to });
    }
  }

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
      <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
        <span style={{ color: "#94a3b8", fontSize: 13 }}>From</span>
        <input
          type="date"
          value={range.from}
          onChange={handleFrom}
          style={inputStyle}
        />
      </label>
      <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
        <span style={{ color: "#94a3b8", fontSize: 13 }}>To</span>
        <input
          type="date"
          value={range.to}
          onChange={handleTo}
          style={inputStyle}
        />
      </label>
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  background: "#1e293b",
  border: "1px solid #334155",
  borderRadius: 6,
  color: "#e2e8f0",
  padding: "4px 8px",
  fontSize: 13,
};
