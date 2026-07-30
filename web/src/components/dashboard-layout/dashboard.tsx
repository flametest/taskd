import type { SidebarSection } from "./sidebar";
import { Sidebar } from "./sidebar";

export function DashboardLayout({
  sections,
  children,
}: {
  sections: SidebarSection[];
  children: React.ReactNode;
}) {
  return (
    <>
      <Sidebar sections={sections} />
      <main className="h-full flex-1 overflow-y-auto">{children}</main>
    </>
  );
}
