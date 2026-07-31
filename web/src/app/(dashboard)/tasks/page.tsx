"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Button,
  Chip,
  Input,
  Pagination,
  Select,
  SelectItem,
  Table,
  TableBody,
  TableCell,
  TableColumn,
  TableHeader,
  TableRow,
} from "@heroui/react";
import { Icon } from "@iconify/react";
import { useTasks } from "@/hooks/use-tasks";
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

const PAGE_SIZE = 20;

export default function TasksPage() {
  const router = useRouter();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("");
  const [createdFrom, setCreatedFrom] = useState("");
  const [createdTo, setCreatedTo] = useState("");
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const fromRfc = createdFrom ? new Date(createdFrom + "T00:00:00").toISOString() : undefined;
  const toRfc = createdTo ? new Date(createdTo + "T23:59:59").toISOString() : undefined;
  const { data, isLoading } = useTasks({
    search: search || undefined,
    status: status || undefined,
    createdFrom: fromRfc,
    createdTo: toRfc,
    page,
    pageSize: PAGE_SIZE,
  });
  const totalPages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1;

  return (
    <div className="flex flex-col gap-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Tasks</h1>
        <Button color="primary" onClick={() => setCreateOpen(true)}>
          <Icon icon="solar:add-circle-bold" width={18} /> New Task
        </Button>
      </div>
      <CreateTaskModal isOpen={createOpen} onClose={() => setCreateOpen(false)} />

      <div className="flex flex-wrap items-center gap-3">
        <Input
          className="max-w-xs"
          placeholder="Search name or ref_id..."
          size="sm"
          value={search}
          onValueChange={(v) => {
            setSearch(v);
            setPage(1);
          }}
        />
        <Select
          className="max-w-[160px]"
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
          type="date"
          className="max-w-[150px]"
          size="sm"
          value={createdFrom}
          onValueChange={(v) => {
            setCreatedFrom(v);
            setPage(1);
          }}
        />
        <span className="text-default-400 text-sm">to</span>
        <Input
          type="date"
          className="max-w-[150px]"
          size="sm"
          value={createdTo}
          onValueChange={(v) => {
            setCreatedTo(v);
            setPage(1);
          }}
        />
        {data && <span className="text-default-400 text-sm">{data.total} total</span>}
      </div>

      <Table
        aria-label="tasks"
        onRowAction={(key) => router.push(`/tasks/${String(key)}`)}
      >
        <TableHeader>
          <TableColumn>ID</TableColumn>
          <TableColumn>NAME</TableColumn>
          <TableColumn>REF ID</TableColumn>
          <TableColumn>PROTOCOL</TableColumn>
          <TableColumn>STATUS</TableColumn>
          <TableColumn>ATTEMPTS</TableColumn>
          <TableColumn>NEXT EXEC</TableColumn>
          <TableColumn>CRON</TableColumn>
          <TableColumn>CREATED</TableColumn>
        </TableHeader>
        <TableBody
          items={data?.tasks ?? []}
          isLoading={isLoading}
          emptyContent="No tasks"
        >
          {(task) => (
            <TableRow key={task.id} className="cursor-pointer">
              <TableCell className="font-mono text-xs">{task.id.slice(0, 8)}</TableCell>
              <TableCell>{task.name}</TableCell>
              <TableCell className="font-mono text-xs">{task.ref_id}</TableCell>
              <TableCell>{task.protocol}</TableCell>
              <TableCell>
                <Chip size="sm" color={STATUS_COLOR[task.status]} variant="flat">
                  {task.status}
                </Chip>
              </TableCell>
              <TableCell>{`${task.attempts}/${task.max_retries}`}</TableCell>
              <TableCell className="text-xs">
                {task.exec_time ? new Date(task.exec_time).toLocaleString() : "-"}
              </TableCell>
              <TableCell className="font-mono text-xs">{task.cron || "-"}</TableCell>
              <TableCell className="text-xs">
                {task.created_at ? new Date(task.created_at).toLocaleString() : "-"}
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      <div className="flex justify-center">
        <Pagination total={totalPages} page={page} onChange={setPage} />
      </div>
    </div>
  );
}
