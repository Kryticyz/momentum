export interface ProjectStat {
  project: string;
  minutes: number;
  hours: number;
}

export interface DayStat {
  date: string;
  minutes: number;
  hours: number;
}

export interface WeekStat {
  weekStart: string;
  minutes: number;
  hours: number;
}

export interface DateRange {
  from: string;
  to: string;
}

export interface ResponseMeta {
  count: number;
  lastLoaded: string;
}

export interface ApiResponse<T> {
  data: T;
  meta: ResponseMeta | null;
}
