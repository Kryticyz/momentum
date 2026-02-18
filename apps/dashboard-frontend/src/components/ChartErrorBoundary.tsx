import { Component } from "react";
import type { ErrorInfo, ReactNode } from "react";
import { theme } from "../theme";

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  errorMessage: string | null;
}

export class ChartErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, errorMessage: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, errorMessage: error.message };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error("ChartErrorBoundary caught:", error, info.componentStack);
  }

  render(): ReactNode {
    if (this.state.hasError) {
      return (
        <div
          role="alert"
          style={{
            color: theme.colors.errorBoundaryText,
            textAlign: "center",
            padding: "32px 16px",
            fontSize: 13,
          }}
        >
          <p style={{ fontWeight: 600, marginBottom: 8 }}>
            Something went wrong rendering this chart.
          </p>
          {this.state.errorMessage && (
            <p style={{ color: theme.colors.muted, fontSize: 12 }}>
              {this.state.errorMessage}
            </p>
          )}
        </div>
      );
    }
    return this.props.children;
  }
}
