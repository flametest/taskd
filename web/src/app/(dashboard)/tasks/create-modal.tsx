"use client";

import { useState } from "react";
import {
  Button,
  Input,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  Select,
  SelectItem,
} from "@heroui/react";
import { useCreateTask } from "@/hooks/use-tasks";
import type { CreateTaskReq, Protocol } from "@/types/task";

const PROTOCOLS: Protocol[] = ["http", "https", "grpc"];

export function CreateTaskModal({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) {
  const create = useCreateTask();
  const [name, setName] = useState("");
  const [refId, setRefId] = useState("");
  const [protocol, setProtocol] = useState<Protocol>("http");
  const [address, setAddress] = useState("");
  const [execTime, setExecTime] = useState("");
  const [maxRetries, setMaxRetries] = useState("5");
  const [cron, setCron] = useState("");

  const submit = () => {
    const body: CreateTaskReq["body"] = {
      name,
      ref_id: refId,
      protocol,
      address,
      params: {},
      exec_time: Math.floor(new Date(execTime).getTime() / 1000),
      max_retries: Number(maxRetries) || 5,
    };
    if (cron) body.cron = cron;
    create.mutate(
      { body },
      {
        onSuccess: () => {
          onClose();
          setName("");
          setRefId("");
          setAddress("");
          setExecTime("");
          setCron("");
        },
      },
    );
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <ModalContent>
        <ModalHeader>New Task</ModalHeader>
        <ModalBody className="flex flex-col gap-3">
          <Input label="Name" value={name} onValueChange={setName} />
          <Input label="Ref ID" value={refId} onValueChange={setRefId} />
          <Select
            label="Protocol"
            selectedKeys={[protocol]}
            onChange={(e) => setProtocol(e.target.value as Protocol)}
          >
            {PROTOCOLS.map((p) => (
              <SelectItem key={p}>{p}</SelectItem>
            ))}
          </Select>
          <Input
            label="Address"
            value={address}
            onValueChange={setAddress}
            placeholder="host:port/path or scheme://..."
          />
          <Input
            label="Exec Time"
            type="datetime-local"
            value={execTime}
            onValueChange={setExecTime}
          />
          <Input
            label="Max Retries"
            type="number"
            value={maxRetries}
            onValueChange={setMaxRetries}
          />
          <Input
            label="Cron (optional)"
            value={cron}
            onValueChange={setCron}
            placeholder="@every 5m or 0 9 * * *"
          />
        </ModalBody>
        <ModalFooter>
          <Button variant="flat" onClick={onClose}>
            Cancel
          </Button>
          <Button color="primary" onClick={submit} isLoading={create.isPending}>
            Create
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
