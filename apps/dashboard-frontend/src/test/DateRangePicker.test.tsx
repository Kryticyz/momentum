import { render, screen, fireEvent } from "@testing-library/react";
import { DateRangePicker } from "../components/DateRangePicker";

const defaultRange = { from: "2026-01-01", to: "2026-01-31" };

describe("DateRangePicker", () => {
  it("renders two date inputs with current values", () => {
    render(<DateRangePicker range={defaultRange} onChange={() => {}} />);
    const inputs = screen.getAllByDisplayValue(/2026/);
    expect(inputs).toHaveLength(2);
  });

  it("calls onChange with updated from date when valid", () => {
    const onChange = vi.fn();
    render(<DateRangePicker range={defaultRange} onChange={onChange} />);
    const [fromInput] = screen.getAllByDisplayValue(/2026/);
    fireEvent.change(fromInput, { target: { value: "2026-01-10" } });
    expect(onChange).toHaveBeenCalledWith({ from: "2026-01-10", to: "2026-01-31" });
  });

  it("does not call onChange when from > to", () => {
    const onChange = vi.fn();
    render(<DateRangePicker range={defaultRange} onChange={onChange} />);
    const [fromInput] = screen.getAllByDisplayValue(/2026/);
    // Setting from to Feb 01 which is after to (Jan 31).
    fireEvent.change(fromInput, { target: { value: "2026-02-01" } });
    expect(onChange).not.toHaveBeenCalled();
  });

  it("calls onChange with updated to date when valid", () => {
    const onChange = vi.fn();
    render(<DateRangePicker range={defaultRange} onChange={onChange} />);
    const inputs = screen.getAllByDisplayValue(/2026/);
    const toInput = inputs[1];
    fireEvent.change(toInput, { target: { value: "2026-01-20" } });
    expect(onChange).toHaveBeenCalledWith({ from: "2026-01-01", to: "2026-01-20" });
  });

  it("does not call onChange when to < from", () => {
    const onChange = vi.fn();
    render(<DateRangePicker range={defaultRange} onChange={onChange} />);
    const inputs = screen.getAllByDisplayValue(/2026/);
    const toInput = inputs[1];
    fireEvent.change(toInput, { target: { value: "2025-12-31" } });
    expect(onChange).not.toHaveBeenCalled();
  });
});
