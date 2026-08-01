"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Button,
  Chip,
  Input,
  Pagination,
  Select,
  SelectItem,
} from "@heroui/react";
import { Icon } from "@iconify/react";
import { useTasks } from "@/hooks/use-tasks";
import { DataTable, type DataTableColumn } from "@/components/ui/data-table";
import type { Task } from "@/types/task";
import { CreateTaskModal } from "./create-modal";

const STATUS_OPTIONS = [
  { label: "All", value: "" },
  { label: "Scheduled", value: "scheduled" },
  { label: "Claimed", value: "claimed" },
  { label: "Succeeded", value: "succeeded" },
  { label: "Dead", value: "dead" },
  { label: "Canceled", value: "canceled" },
];

const STATUS_COLOR: Record<string, "default" | "primary" | "success" | "warning" | "danger"> = {
  scheduled: "primary",
  claimed: "warning",
  running: "warning",
  succeeded: "success",
  failed: "danger",
  dead: "danger",
  canceled: "default",
};

const TIME_PRESETS = [
  { key: "1h", label: "1H" },
  { key: "today", label: "Today" },
  { key: "custom", label: "Custom" },
] as const;

type TimePreset = (typeof TIME_PRESETS)[number]["key"];

function getPresetTimeRange(preset: "1h" | "today"): { start: Date; end: Date } {
  const now = new Date();
  switch (preset) {
    case "1h":
      return { start: new Date(now.getTime() - 60 * 60 * 1000), end: now };
    case "today": {
      const start = new Date(now);
      start.setHours(0, 0, 0, 0);
      return { start, end: now };
    }
  }
}

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100];

const COLUMNS: DataTableColumn<Task>[] = [
  { key: "id", label: "ID", render: (t) => <span className="font-mono text-xs">{t.id.slice(0, 8)}</span> },
  { key: "name", label: "NAME" },
  { key: "ref_id", label: "REF ID", render: (t) => <span className="font-mono text-xs">{t.ref_id}</span> },
  { key: "protocol", label: "PROTOCOL" },
  {
    key: "status",
    label: "STATUS",
    render: (t) => (
      <Chip size="sm" color={STATUS_COLOR[t.status]} variant="flat">
        {t.status}
      </Chip>
    ),
  },
  { key: "attempts", label: "ATTEMPTS", render: (t) => `${t.attempts}/${t.max_retries}` },
  {
    key: "exec_time",
    label: "NEXT EXEC",
    render: (t) => (
      <span className="text-xs">
        {t.exec_time ? new Date(t.exec_time).toLocaleString() : "-"}
      </span>
    ),
  },
  { key: "cron", label: "CRON", render: (t) => <span className="font-mono text-xs">{t.cron || "-"}</span> },
  {
    key: "created_at",
    label: "CREATED",
    render: (t) => (
      <span className="text-xs">
        {t.created_at ? new Date(t.created_at).toLocaleString() : "-"}
      </span>
    ),
  },
];

export default function TasksPage() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [timePreset, setTimePreset] = useState<TimePreset | "">("");
  const [customStart, setCustomStart] = useState("");
  const [customEnd, setCustomEnd] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [createOpen, setCreateOpen] = useState(false);

  // Memoized so fromRfc/toRfc stay stable between renders -- getPresetTimeRange
  // calls new Date(), which would otherwise change every render and make the
  // react-query queryKey churn (infinite refetch).
  const { fromRfc, toRfc } = useMemo(() => {
    if (timePreset === "1h" || timePreset === "today") {
      const { start, end } = getPresetTimeRange(timePreset);
      return { fromRfc: start.toISOString(), toRfc: end.toISOString() };
    }
    if (timePreset === "custom") {
      return {
        fromRfc: customStart ? new Date(customStart).toISOString() : undefined,
        toRfc: customEnd ? new Date(customEnd).toISOString() : undefined,
      };
    }
    return { fromRfc: undefined, toRfc: undefined };
  }, [timePreset, customStart, customEnd]);

  const { data, isLoading } = useTasks({
    search: search || undefined,
    status: status || undefined,
    createdFrom: fromRfc,
    createdTo: toRfc,
    page,
    pageSize,
  });
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="flex flex-col gap-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Tasks</h1>
        <Button color="primary" onClick={() => setCreateOpen(true)}>
          <Icon icon="solar:add-circle-bold" width={18} /> New Task
        </Button>
      </div>
      <CreateTaskModal isOpen={createOpen} onClose={() => setCreateOpen(false)} />

      <DataTable
        columns={COLUMNS}
        data={data?.tasks ?? []}
        keyExtractor={(t) => t.id}
        isLoading={isLoading}
        emptyMessage="No tasks"
        onRowClick={(t) => router.push(`/tasks/${t.id}`)}
        title={
          <span className="text-sm font-medium">
            {total} {total === 1 ? "task" : "tasks"}
          </span>
        }
        headerAction={
          <div className="flex flex-nowrap items-center gap-2">
            <div className="flex shrink-0 gap-1">
              {TIME_PRESETS.map((preset) => (
                <Button
                  key={preset.key}
                  size="sm"
                  variant={timePreset === preset.key ? "solid" : "flat"}
                  color={timePreset === preset.key ? "primary" : "default"}
                  onPress={() => {
                    setTimePreset(timePreset === preset.key ? "" : preset.key);
                    setPage(1);
                  }}
                >
                  {preset.label}
                </Button>
              ))}
            </div>
            {timePreset === "custom" && (
              <>
                <input
                  type="datetime-local"
                  className="h-8 rounded-lg border border-default-200 bg-default-100 px-2 text-xs outline-none focus:border-primary"
                  value={customStart}
                  onChange={(e) => {
                    setCustomStart(e.target.value);
                    setPage(1);
                  }}
                />
                <span className="text-xs text-default-400">-</span>
                <input
                  type="datetime-local"
                  className="h-8 rounded-lg border border-default-200 bg-default-100 px-2 text-xs outline-none focus:border-primary"
                  value={customEnd}
                  onChange={(e) => {
                    setCustomEnd(e.target.value);
                    setPage(1);
                  }}
                />
              </>
            )}
            <Select
              className="max-w-[130px] shrink-0"
              size="sm"
              selectedKeys={[status]}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
            >
              {STATUS_OPTIONS.map((o) => (
                <SelectItem key={o.value}>{o.label}</SelectItem>
              ))}
            </Select>
            <Input
              className="w-80 shrink-0"
              placeholder="Search name or ref_id..."
              size="sm"
              value={search}
              onValueChange={(v) => {
                setSearch(v);
                setPage(1);
              }}
            />
          </div>
        }
        footer={
          <div className="flex items-center justify-between px-4 py-2">
            <div className="flex items-center gap-2 text-xs text-default-500">
              <Select
                size="sm"
                className="w-20"
                aria-label="Page size"
                selectedKeys={[String(pageSize)]}
                onSelectionChange={(keys) => {
                  const val = Number(Array.from(keys)[0]);
                  if (val) {
                    setPageSize(val);
                    setPage(1);
                  }
                }}
              >
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <SelectItem key={String(size)}>{String(size)}</SelectItem>
                ))}
              </Select>
              <span>per page</span>
              <span className="text-default-400">|</span>
              <span>
                Page {page} of {totalPages}
              </span>
            </div>
            {totalPages > 1 && (
              <Pagination
                size="sm"
                total={totalPages}
                page={page}
                onChange={setPage}
                showControls
              />
            )}
          </div>
        }
      />
    </div>
  );
}
