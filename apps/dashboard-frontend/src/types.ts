/**
 * Re-export API types generated from the OpenAPI spec.
 *
 * The generated file is the single source of truth for API shapes.
 * Run `bun run generate:types` to regenerate after spec changes.
 */
export type { components } from "./generated/api-types";

import type { components } from "./generated/api-types";

// Convenience aliases used throughout the frontend.
export type ProjectStat = components["schemas"]["ProjectStat"];
export type DayStat = components["schemas"]["DayStat"];
export type WeekStat = components["schemas"]["WeekStat"];
export type TimeEntry = components["schemas"]["TimeEntry"];
export type ResponseMeta = components["schemas"]["ResponseMeta"];
export type ApiEnvelope = components["schemas"]["APIEnvelope"];

/** Wrapper used by fetchJSON to type the full envelope. */
export interface ApiResponse<T> {
  data: T;
  meta: ResponseMeta | null;
}

/** Frontend-only type for date range filter state. */
export interface DateRange {
  from: string;
  to: string;
}
