import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { ChartErrorBoundary } from "../components/ChartErrorBoundary";

function ThrowingChild(): JSX.Element {
  throw new Error("test explosion");
}

function GoodChild(): JSX.Element {
  return <p>Chart rendered OK</p>;
}

describe("ChartErrorBoundary", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders children when no error occurs", () => {
    render(
      <ChartErrorBoundary>
        <GoodChild />
      </ChartErrorBoundary>
    );
    expect(screen.getByText("Chart rendered OK")).toBeInTheDocument();
  });

  it("renders fallback UI when a child throws", () => {
    render(
      <ChartErrorBoundary>
        <ThrowingChild />
      </ChartErrorBoundary>
    );
    expect(
      screen.getByText("Something went wrong rendering this chart.")
    ).toBeInTheDocument();
    expect(screen.getByText("test explosion")).toBeInTheDocument();
  });

  it("renders the fallback with role=alert for accessibility", () => {
    render(
      <ChartErrorBoundary>
        <ThrowingChild />
      </ChartErrorBoundary>
    );
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("does not affect sibling boundaries", () => {
    render(
      <div>
        <ChartErrorBoundary>
          <ThrowingChild />
        </ChartErrorBoundary>
        <ChartErrorBoundary>
          <GoodChild />
        </ChartErrorBoundary>
      </div>
    );
    expect(
      screen.getByText("Something went wrong rendering this chart.")
    ).toBeInTheDocument();
    expect(screen.getByText("Chart rendered OK")).toBeInTheDocument();
  });
});
