import { useState } from "react";
import { postRefresh } from "../api";

interface Props {
  onRefresh: () => void;
}

export function RefreshButton({ onRefresh }: Props) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    setLoading(true);
    setError(null);
    try {
      await postRefresh();
      onRefresh();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <button
        onClick={handleClick}
        disabled={loading}
        style={{
          background: loading ? "#334155" : "#3b82f6",
          color: "#fff",
          border: "none",
          borderRadius: 6,
          padding: "6px 14px",
          fontSize: 13,
          cursor: loading ? "not-allowed" : "pointer",
        }}
      >
        {loading ? "Refreshing…" : "Refresh data"}
      </button>
      {error && (
        <span style={{ color: "#f87171", fontSize: 12 }}>{error}</span>
      )}
    </div>
  );
}
