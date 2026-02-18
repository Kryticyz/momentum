import { theme } from "../theme";
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
        <span style={{ color: theme.colors.label, fontSize: 13 }}>From</span>
        <input
          type="date"
          value={range.from}
          onChange={handleFrom}
          style={inputStyle}
        />
      </label>
      <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
        <span style={{ color: theme.colors.label, fontSize: 13 }}>To</span>
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
  background: theme.colors.inputBg,
  border: `1px solid ${theme.colors.inputBorder}`,
  borderRadius: 6,
  color: theme.colors.inputText,
  padding: "4px 8px",
  fontSize: 13,
};
