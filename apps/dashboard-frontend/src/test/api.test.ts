import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { fetchDays, fetchProjects, fetchWeeks, postRefresh } from "../api";

const range = { from: "2026-02-01", to: "2026-02-28" };

describe("api client", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("uses versioned endpoint for projects", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: [], meta: null }), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    await fetchProjects(range);

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/projects?from=2026-02-01&to=2026-02-28",
      { signal: undefined, headers: {} }
    );
  });

  it("uses versioned endpoints for days and weeks", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: [], meta: null }), { status: 200, headers: { "Content-Type": "application/json" } })
    );
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: [], meta: null }), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    await fetchDays(range);
    await fetchWeeks(range);

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      "/api/v1/days?from=2026-02-01&to=2026-02-28",
      { signal: undefined, headers: {} }
    );
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/v1/weeks?from=2026-02-01&to=2026-02-28",
      { signal: undefined, headers: {} }
    );
  });

  it("passes abort signal to fetch when provided", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: [], meta: null }), { status: 200, headers: { "Content-Type": "application/json" } })
    );

    const controller = new AbortController();
    await fetchProjects(range, controller.signal);

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/projects?from=2026-02-01&to=2026-02-28",
      { signal: controller.signal, headers: {} }
    );
  });

  it("posts refresh to /refresh", async () => {
    const fetchMock = vi.mocked(fetch);
    fetchMock.mockResolvedValueOnce(new Response("", { status: 200 }));

    await postRefresh();

    expect(fetchMock).toHaveBeenCalledWith("/refresh", { method: "POST", headers: {} });
  });
});
