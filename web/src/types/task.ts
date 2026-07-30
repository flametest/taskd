// Mirrors the taskd backend DTOs (pkg/dto). Field names are snake_case to match
// the JSON tags produced by the Go side.
export type Protocol = "http" | "https" | "grpc";

export type Status =
  | "scheduled"
  | "claimed"
  | "running"
  | "succeeded"
  | "failed"
  | "dead"
  | "canceled";

export type ExecutionResult = "success" | "failure";

export type Task = {
  id: string;
  name: string;
  ref_id: string;
  protocol: Protocol;
  address: string;
  params: Record<string, unknown>;
  exec_time: string | null;
  status: Status;
  attempts: number;
  max_retries: number;
  last_error: string;
  locked_until: string | null;
  cron: string;
};

export type TaskRecord = {
  id: string;
  task_id: string;
  attempt: number;
  result: ExecutionResult;
  protocol: Protocol;
  instance_id: string;
  error_message: string;
  started_at: string;
  finished_at: string;
  duration_ms: number;
  response: unknown;
  created_at: string;
};

export type ListTasksResp = {
  tasks: Task[];
  total: number;
  limit: number;
  offset: number;
};

export type ListTaskRecordsResp = {
  records: TaskRecord[];
  limit: number;
  offset: number;
};

export type CreateTaskBody = {
  name: string;
  ref_id: string;
  protocol: Protocol;
  address: string;
  params: Record<string, unknown>;
  exec_time: number; // unix seconds
  max_retries: number;
  cron?: string;
};

export type CreateTaskReq = { body: CreateTaskBody };
