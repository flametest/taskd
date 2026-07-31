"use client";

import { useParams } from "next/navigation";
import { Accordion, AccordionItem, Button, Card, CardBody, Chip } from "@heroui/react";
import {
  useCancelTask,
  useReactivateTask,
  useTask,
  useTaskRecords,
} from "@/hooks/use-tasks";

export default function TaskDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: task, isLoading } = useTask(id);
  const { data: recordsData } = useTaskRecords(id);
  const cancel = useCancelTask();
  const reactivate = useReactivateTask();

  if (isLoading) return <div className="p-6">Loading...</div>;
  if (!task) return <div className="p-6">Task not found</div>;

  const canCancel = task.status === "scheduled" || task.status === "claimed";
  const canReactivate = task.status === "dead";

  return (
    <div className="flex flex-col gap-4 p-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">{task.name}</h1>
        <div className="flex gap-2">
          {canCancel && (
            <Button
              color="danger"
              variant="flat"
              onClick={() => cancel.mutate(id)}
              isLoading={cancel.isPending}
            >
              Cancel
            </Button>
          )}
          {canReactivate && (
            <Button
              color="primary"
              onClick={() => reactivate.mutate({ id })}
              isLoading={reactivate.isPending}
            >
              Reactivate
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardBody className="flex flex-col gap-2">
          <Row label="ID" value={task.id} mono />
          <Row label="Status" value={<Chip size="sm" variant="flat">{task.status}</Chip>} />
          <Row label="Protocol" value={task.protocol} />
          <Row label="Address" value={task.address} />
          <Row label="Ref ID" value={task.ref_id} />
          <Row label="Attempts" value={`${task.attempts} / ${task.max_retries}`} />
          <Row label="Cron" value={task.cron || "-"} mono />
          <Row
            label="Exec Time"
            value={task.exec_time ? new Date(task.exec_time).toLocaleString() : "-"}
          />
          <Row
            label="Locked Until"
            value={task.locked_until ? new Date(task.locked_until).toLocaleString() : "-"}
          />
          {task.last_error && <Row label="Last Error" value={task.last_error} />}
          <Row
            label="Params"
            value={
              <pre className="text-xs">{JSON.stringify(task.params, null, 2)}</pre>
            }
          />
        </CardBody>
      </Card>

      <h2 className="text-lg font-semibold">Execution Records</h2>
      <Accordion selectionMode="multiple" variant="splitted">
        {(recordsData?.records ?? []).map((r) => (
          <AccordionItem
            key={r.id}
            aria-label={`record-${r.id}`}
            title={
              <div className="flex items-center gap-2">
                <Chip
                  size="sm"
                  color={r.result === "success" ? "success" : "danger"}
                  variant="flat"
                >
                  {r.result}
                </Chip>
                <span className="text-default-500">attempt #{r.attempt}</span>
                <span className="text-default-400 text-xs">
                  {new Date(r.started_at).toLocaleString()}
                </span>
                <span className="text-default-400 text-xs">{r.duration_ms}ms</span>
              </div>
            }
          >
            <div className="flex flex-col gap-1 text-xs">
              <div>
                <span className="text-default-400">Started: </span>
                {new Date(r.started_at).toLocaleString()}
              </div>
              <div>
                <span className="text-default-400">Finished: </span>
                {new Date(r.finished_at).toLocaleString()}
              </div>
              <div>
                <span className="text-default-400">Duration: </span>
                {r.duration_ms}ms
              </div>
              <div>
                <span className="text-default-400">Instance: </span>
                <span className="font-mono">{r.instance_id}</span>
              </div>
              <div>
                <span className="text-default-400">Protocol: </span>
                {r.protocol}
              </div>
              {r.error_message && (
                <div className="text-danger">
                  <span>Error: </span>
                  {r.error_message}
                </div>
              )}
              {r.response != null && (
                <div>
                  <span className="text-default-400">Response: </span>
                  <pre className="mt-1 overflow-x-auto rounded bg-default-100 p-2">
                    {JSON.stringify(r.response, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </AccordionItem>
        ))}
      </Accordion>
      {recordsData && recordsData.records.length === 0 && (
        <span className="text-default-400 text-sm">No executions yet</span>
      )}
    </div>
  );
}

function Row({
  label,
  value,
  mono,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div className="flex gap-3">
      <span className="w-32 shrink-0 text-default-400 text-sm">{label}</span>
      <span className={`text-sm ${mono ? "font-mono" : ""}`}>{value}</span>
    </div>
  );
}
