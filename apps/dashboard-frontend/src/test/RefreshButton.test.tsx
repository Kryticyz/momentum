import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { RefreshButton } from "../components/RefreshButton";
import * as api from "../api";

describe("RefreshButton", () => {
  it("renders with 'Refresh data' label", () => {
    render(<RefreshButton onRefresh={() => {}} />);
    expect(screen.getByText("Refresh data")).toBeInTheDocument();
  });

  it("calls postRefresh and onRefresh on click", async () => {
    vi.spyOn(api, "postRefresh").mockResolvedValueOnce(undefined);
    const onRefresh = vi.fn();
    render(<RefreshButton onRefresh={onRefresh} />);

    fireEvent.click(screen.getByText("Refresh data"));

    await waitFor(() => {
      expect(api.postRefresh).toHaveBeenCalledOnce();
      expect(onRefresh).toHaveBeenCalledOnce();
    });
  });

  it("shows 'Refreshing…' while in flight", async () => {
    let resolve!: () => void;
    vi.spyOn(api, "postRefresh").mockReturnValue(
      new Promise<void>((res) => { resolve = res; })
    );
    render(<RefreshButton onRefresh={() => {}} />);

    fireEvent.click(screen.getByText("Refresh data"));
    expect(await screen.findByText("Refreshing…")).toBeInTheDocument();

    resolve();
    await waitFor(() => {
      expect(screen.getByText("Refresh data")).toBeInTheDocument();
    });
  });

  it("shows error message on failure", async () => {
    vi.spyOn(api, "postRefresh").mockRejectedValueOnce(new Error("network error"));
    render(<RefreshButton onRefresh={() => {}} />);

    fireEvent.click(screen.getByText("Refresh data"));

    await waitFor(() => {
      expect(screen.getByText(/network error/)).toBeInTheDocument();
    });
  });

  it("button is disabled while refreshing", async () => {
    let resolve!: () => void;
    vi.spyOn(api, "postRefresh").mockReturnValue(
      new Promise<void>((res) => { resolve = res; })
    );
    render(<RefreshButton onRefresh={() => {}} />);

    fireEvent.click(screen.getByText("Refresh data"));
    const btn = await screen.findByText("Refreshing…");
    expect(btn.closest("button")).toBeDisabled();

    resolve();
    await waitFor(() => {
      expect(screen.getByText("Refresh data").closest("button")).not.toBeDisabled();
    });
  });
});
