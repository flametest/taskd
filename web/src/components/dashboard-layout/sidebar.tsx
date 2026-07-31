"use client";

import { Icon } from "@iconify/react";
import Link from "next/link";
import { usePathname } from "next/navigation";

export type SidebarItem = {
  key: string;
  title: string;
  icon: string; // iconify name, e.g. "solar:list-bold"
  href: string;
};

export type SidebarSection = {
  key: string;
  title: string;
  items: SidebarItem[];
};

export function Sidebar({ sections }: { sections: SidebarSection[] }) {
  const pathname = usePathname();
  return (
    <aside className="h-full w-50 shrink-0 overflow-y-auto border-r border-default-200 bg-content1 p-4">
      <div className="mb-6 px-2 text-lg font-bold text-primary">taskd</div>
      <nav className="flex flex-col gap-6">
        {sections.map((section) => (
          <div key={section.key} className="flex flex-col gap-1">
            <div className="px-2 text-xs font-semibold tracking-wide text-default-400 uppercase">
              {section.title}
            </div>
            {section.items.map((item) => {
              const active = pathname === item.href || pathname.startsWith(item.href + "/");
              return (
                <Link
                  key={item.key}
                  href={item.href}
                  className={`flex items-center gap-2 rounded-lg px-2 py-2 text-sm ${
                    active ? "bg-primary text-white" : "text-default-600 hover:bg-default-100"
                  }`}
                >
                  <Icon icon={item.icon} width={18} />
                  {item.title}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>
    </aside>
  );
}
