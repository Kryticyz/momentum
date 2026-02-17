import { renderHook, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { useTimeData } from "../hooks/useTimeData";
import * as api from "../api";

const range = { from: "2026-02-01", to: "2026-02-28" };

const mockProjects = [{ project: "Alpha", minutes: 90, hours: 1.5 }];
const mockDays = [{ date: "2026-02-01", minutes: 90, hours: 1.5 }];
const mockWeeks = [{ weekStart: "2026-02-01", minutes: 90, hours: 1.5 }];

describe("useTimeData", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("starts with loading=true then transitions to false", async () => {
    vi.spyOn(api, "fetchProjects").mockResolvedValue([]);
    vi.spyOn(api, "fetchDays").mockResolvedValue([]);
    vi.spyOn(api, "fetchWeeks").mockResolvedValue([]);

    const { result } = renderHook(() => useTimeData(range, 0));
    // Initial state must be loading before the promise resolves.
    expect(result.current.loading).toBe(true);
    await waitFor(() => expect(result.current.loading).toBe(false));
  });

  it("populates data after fetch resolves", async () => {
    vi.spyOn(api, "fetchProjects").mockResolvedValue(mockProjects);
    vi.spyOn(api, "fetchDays").mockResolvedValue(mockDays);
    vi.spyOn(api, "fetchWeeks").mockResolvedValue(mockWeeks);

    const { result } = renderHook(() => useTimeData(range, 0));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.projects).toEqual(mockProjects);
    expect(result.current.days).toEqual(mockDays);
    expect(result.current.weeks).toEqual(mockWeeks);
    expect(result.current.error).toBeNull();
  });

  it("sets error and empty arrays on fetch failure", async () => {
    vi.spyOn(api, "fetchProjects").mockRejectedValue(new Error("server down"));
    vi.spyOn(api, "fetchDays").mockResolvedValue([]);
    vi.spyOn(api, "fetchWeeks").mockResolvedValue([]);

    const { result } = renderHook(() => useTimeData(range, 0));

    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(result.current.error).toMatch(/server down/);
    expect(result.current.projects).toHaveLength(0);
    expect(result.current.days).toHaveLength(0);
    expect(result.current.weeks).toHaveLength(0);
  });

  it("re-fetches when refreshKey changes", async () => {
    const spy = vi.spyOn(api, "fetchProjects").mockResolvedValue([]);
    vi.spyOn(api, "fetchDays").mockResolvedValue([]);
    vi.spyOn(api, "fetchWeeks").mockResolvedValue([]);

    let refreshKey = 0;
    const { result, rerender } = renderHook(
      ({ rk }) => useTimeData(range, rk),
      { initialProps: { rk: 0 } }
    );
    await waitFor(() => expect(result.current.loading).toBe(false));

    refreshKey = 1;
    rerender({ rk: refreshKey });
    await waitFor(() => expect(result.current.loading).toBe(false));

    // fetchProjects should have been called twice (once per key).
    expect(spy).toHaveBeenCalledTimes(2);
  });

  it("re-fetches when date range changes", async () => {
    const spy = vi.spyOn(api, "fetchProjects").mockResolvedValue([]);
    vi.spyOn(api, "fetchDays").mockResolvedValue([]);
    vi.spyOn(api, "fetchWeeks").mockResolvedValue([]);

    const { result, rerender } = renderHook(
      ({ r }) => useTimeData(r, 0),
      { initialProps: { r: range } }
    );
    await waitFor(() => expect(result.current.loading).toBe(false));

    rerender({ r: { from: "2026-03-01", to: "2026-03-31" } });
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(spy).toHaveBeenCalledTimes(2);
  });
});
