"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export function useTasks(params: {
  status?: string;
  page?: number;
  pageSize?: number;
}) {
  const pageSize = params.pageSize ?? 20;
  const page = params.page ?? 1;
  const offset = (page - 1) * pageSize;
  return useQuery({
    queryKey: ["tasks", params.status, page, pageSize],
    queryFn: () =>
      api.listTasks({ status: params.status, limit: pageSize, offset }),
  });
}

export function useTask(id: string) {
  return useQuery({
    queryKey: ["task", id],
    queryFn: () => api.getTask(id),
    enabled: !!id,
  });
}

export function useTaskRecords(id: string, page = 1, pageSize = 20) {
  const offset = (page - 1) * pageSize;
  return useQuery({
    queryKey: ["task-records", id, page, pageSize],
    queryFn: () => api.listRecords(id, pageSize, offset),
    enabled: !!id,
  });
}

export function useCreateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: api.createTask,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tasks"] }),
  });
}

export function useCancelTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancelTask(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: ["task", id] });
      qc.invalidateQueries({ queryKey: ["tasks"] });
    },
  });
}

export function useReactivateTask() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, execTime }: { id: string; execTime?: number }) =>
      api.reactivateTask(id, execTime),
    onSuccess: (_data, { id }) => {
      qc.invalidateQueries({ queryKey: ["task", id] });
      qc.invalidateQueries({ queryKey: ["tasks"] });
    },
  });
}
