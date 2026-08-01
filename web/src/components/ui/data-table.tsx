import React from "react";
import { cn, Spinner } from "@heroui/react";

export interface DataTableColumn<T> {
  key: string;
  label: string;
  width?: string;
  render?: (row: T) => React.ReactNode;
}

interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  data: T[];
  keyExtractor: (row: T) => string;
  title?: React.ReactNode;
  headerAction?: React.ReactNode;
  emptyMessage?: string;
  isLoading?: boolean;
  footer?: React.ReactNode;
  onRowClick?: (row: T) => void;
}

// DataTable is a lightweight table with a header bar (title + headerAction) and
// a footer slot (pagination), styled to match the laas-console list layout.
export function DataTable<T>({
  columns,
  data,
  keyExtractor,
  title,
  headerAction,
  emptyMessage = "No entries yet.",
  isLoading = false,
  footer,
  onRowClick,
}: DataTableProps<T>) {
  const showEmpty = !isLoading && data.length === 0;
  return (
    <div className="flex h-full flex-col overflow-hidden rounded-lg border border-default-200">
      {(title || headerAction) && (
        <div className="flex shrink-0 items-center justify-between border-b border-default-200 px-4 py-3">
          <div>{title}</div>
          <div className="flex min-w-0 flex-1 justify-end pr-22 ">{headerAction}</div>
        </div>
      )}
      <div className="min-h-0 flex-1 overflow-y-auto">
        {isLoading ? (
          <div className="flex h-full items-center justify-center">
            <Spinner color="primary" size="sm" />
          </div>
        ) : showEmpty ? (
          <div className="flex h-full items-center justify-center text-xs text-default-400">
            {emptyMessage}
          </div>
        ) : (
          <table className="w-full">
            <thead className="sticky top-0 z-10">
              <tr className="border-b border-default-200 bg-default-50">
                {columns.map((col) => (
                  <th
                    key={col.key}
                    className={cn(
                      "px-4 py-2 text-left text-xs font-medium text-default-500",
                      col.width,
                    )}
                  >
                    {col.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.map((row) => {
                const key = keyExtractor(row);
                return (
                  <tr
                    key={key}
                    className={cn(
                      "border-b border-default-100 hover:bg-default-50",
                      onRowClick && "cursor-pointer",
                    )}
                    onClick={onRowClick ? () => onRowClick(row) : undefined}
                  >
                    {columns.map((col) => (
                      <td key={col.key} className="px-4 py-2 text-sm">
                        {col.render
                          ? col.render(row)
                          : String(
                              (row as Record<string, unknown>)[col.key] ?? "",
                            )}
                      </td>
                    ))}
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
      {footer && (
        <div className="shrink-0 border-t border-default-200">{footer}</div>
      )}
    </div>
  );
}
