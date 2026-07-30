import { DashboardLayout } from "@/components/dashboard-layout/dashboard";
import type { SidebarSection } from "@/components/dashboard-layout/sidebar";

const sections: SidebarSection[] = [
  {
    key: "tasks",
    title: "TASKS",
    items: [
      { key: "tasks", title: "Tasks", icon: "solar:list-bold", href: "/tasks" },
    ],
  },
];

export default function Layout({ children }: { children: React.ReactNode }) {
  return <DashboardLayout sections={sections}>{children}</DashboardLayout>;
}
