import { renderHook, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { useHeatmapData } from "../hooks/useHeatmapData";
import * as api from "../api";

describe("useHeatmapData", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("starts with loading=true then transitions to false", async () => {
    vi.spyOn(api, "fetchDays").mockResolvedValue([]);

    const { result } = renderHook(() => useHeatmapData(0));
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));
  });

  it("populates days after fetch resolves", async () => {
    const mockDays = [{ date: "2026-02-18", minutes: 60, hours: 1.0 }];
    vi.spyOn(api, "fetchDays").mockResolvedValue(mockDays);

    const { result } = renderHook(() => useHeatmapData(0));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.days).toEqual(mockDays);
    expect(result.current.error).toBeNull();
  });

  it("sets error and empty days on fetch failure", async () => {
    vi.spyOn(api, "fetchDays").mockRejectedValue(new Error("timeout"));

    const { result } = renderHook(() => useHeatmapData(0));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toMatch(/timeout/);
    expect(result.current.days).toHaveLength(0);
  });

  it("re-fetches when refreshKey changes", async () => {
    const spy = vi.spyOn(api, "fetchDays").mockResolvedValue([]);

    const { result, rerender } = renderHook(
      ({ k }) => useHeatmapData(k),
      { initialProps: { k: 0 } }
    );
    await waitFor(() => expect(result.current.loading).toBe(false));

    rerender({ k: 1 });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(spy).toHaveBeenCalledTimes(2);
  });
});
