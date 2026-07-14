"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";

import { authFetch } from "@/lib/auth-session";

type AdminShellProps = {
  children: ReactNode;
  breadcrumb?: string;
};

type MeResponse = {
  data?: {
    user?: {
      email?: string;
      full_name?: string;
    };
  };
};

type AdminNavItem = {
  label: string;
  href: string;
  icon: (props?: { className?: string }) => ReactNode;
  active?: boolean;
  badge?: string;
  tone?: "red" | "yellow" | "neutral";
};

type AdminNavSection = {
  title: string;
  items: AdminNavItem[];
};

const navSections: AdminNavSection[] = [
  {
    title: "Обзор",
    items: [
      { label: "Сводка", href: "/app/admin", icon: DashboardIcon },
      { label: "Алерты", href: "/app/admin#alerts", icon: BellIcon, badge: "—", tone: "red" },
    ],
  },
  {
    title: "Организации",
    items: [
      { label: "Заявки", href: "/app/admin#org-requests", icon: InboxIcon, badge: "new" },
      { label: "Пользователи", href: "/app/admin/users", icon: UserIcon },
      { label: "Клиенты", href: "/app/admin/clients", icon: UsersIcon },
      { label: "Производители", href: "/app/admin/manufacturers", icon: FactoryIcon },
      { label: "Поставщики", href: "/app/admin/vendors", icon: BriefcaseIcon },
      { label: "Интеграторы", href: "/app/admin/integrators", icon: WrenchIcon },
    ],
  },
  {
    title: "Биллинг",
    items: [
      { label: "Выручка и MRR", href: "/app/admin/billing", icon: ChartIcon },
      { label: "Тарифы", href: "/app/admin/billing#plans", icon: CardIcon },
      { label: "Инвойсы", href: "/app/admin/billing#invoices", icon: FileIcon },
    ],
  },
  {
    title: "Платформа",
    items: [
      { label: "Тикеты onboarding", href: "/app/admin/tickets", icon: TicketIcon, badge: "!" },
      { label: "Документы", href: "/app/admin#docs", icon: FileIcon },
      { label: "Workers", href: "/app/admin#workers", icon: CpuIcon, badge: "—", tone: "yellow" },
      { label: "Расходы ИИ", href: "/app/admin#llm", icon: SparkIcon },
      { label: "Object storage (S3)", href: "/app/admin#s3", icon: CloudIcon },
      { label: "Email / SMTP", href: "/app/admin#smtp", icon: MailIcon },
    ],
  },
  {
    title: "Конфигурация",
    items: [
      { label: "Feature flags", href: "/app/admin#flags", icon: FlagIcon },
      { label: "Audit log", href: "/app/admin#audit", icon: AuditIcon },
      { label: "System health", href: "/app/admin#health", icon: HealthIcon },
      { label: "Аналитика", href: "/app/admin#analytics", icon: ChartIcon },
    ],
  },
];

function initials(email: string) {
  const local = email.split("@")[0] || "admin";
  const parts = local.split(/[._-]+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0]?.[0] || ""}${parts[1]?.[0] || ""}`.toUpperCase();
  }
  return local.slice(0, 2).toUpperCase();
}

export function AdminShell({ children, breadcrumb = "Dashboard" }: AdminShellProps) {
  const pathname = usePathname();
  const [email, setEmail] = useState("admin@asutport.ru");
  const avatar = useMemo(() => initials(email), [email]);

  function isNavActive(href: string) {
    if (href.startsWith("#")) {
      return false;
    }
    const path = href.split("#")[0];
    const normalized = pathname?.replace(/^\/app/, "") || pathname;
    const target = path.replace(/^\/app/, "");
    if (target === "/admin") {
      return normalized === "/admin";
    }
    return normalized === target || pathname === path;
  }

  useEffect(() => {
    void authFetch("/api/v1/auth/me")
      .then((response) => (response.ok ? response.json() : null))
      .then((body: MeResponse | null) => {
        if (body?.data?.user?.email) {
          setEmail(body.data.user.email);
        }
      })
      .catch(() => undefined);
  }, []);

  function signOut() {
    sessionStorage.removeItem("asutport_access_token");
    sessionStorage.removeItem("asutport_refresh_token");
    window.location.href = "/app/login";
  }

  return (
    <div className="min-h-screen bg-[#f3f2ef] text-[#18212f]">
      <aside className="fixed left-0 top-0 z-20 flex h-screen w-[220px] flex-col bg-[#0f172a]">
        <div className="flex items-center gap-2 border-b border-white/[0.06] px-4 pb-3.5 pt-[18px]">
          <div className="grid h-7 w-7 shrink-0 place-items-center rounded-[7px] bg-[#2563eb]">
            <AdminLogoIcon />
          </div>
          <div className="min-w-0">
            <span className="block truncate text-[13px] font-semibold tracking-tight text-white">
              ASUTPORT
            </span>
            <span className="mt-px inline-flex rounded bg-[#2563eb] px-1.5 py-px text-[9px] font-semibold uppercase tracking-wider text-white">
              Superadmin
            </span>
          </div>
        </div>

        <nav className="flex-1 overflow-y-auto px-2 py-2.5">
          {navSections.map((section, index) => (
            <div key={section.title}>
              <div
                className={`px-2.5 pb-1.5 text-[10px] font-medium uppercase tracking-[0.09em] text-white/30 ${
                  index === 0 ? "pt-1" : "pt-3"
                }`}
              >
                {section.title}
              </div>
              <ul className="space-y-px">
                {section.items.map((item) => {
                  const Icon = item.icon;
                  const active = isNavActive(item.href);
                  return (
                    <li key={item.label}>
                      <Link
                        href={item.href}
                        className={
                          active
                            ? "flex items-center gap-2 rounded-lg bg-[#1d4ed8] px-2.5 py-[7px] text-[12px] font-medium text-white"
                            : "flex items-center gap-2 rounded-lg px-2.5 py-[7px] text-[12px] text-white/70 transition-colors hover:bg-[#1e293b] hover:text-white"
                        }
                      >
                        <Icon />
                        <span className="flex-1 truncate">{item.label}</span>
                        {item.badge ? (
                          <span
                            className={`rounded-full px-1.5 py-px text-[10px] font-medium ${
                              item.tone === "red"
                                ? "bg-red-600 text-white"
                                : item.tone === "yellow"
                                  ? "bg-amber-600 text-white"
                                  : "bg-white/10 text-white/70"
                            }`}
                          >
                            {item.badge}
                          </span>
                        ) : null}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          ))}

          <div className="mx-2.5 my-2 h-px bg-white/[0.06]" />
          <Link
            href="/app/dashboard"
            className="flex items-center gap-2 rounded-lg px-2.5 py-[7px] text-[12px] text-white/40 transition-colors hover:bg-[#1e293b] hover:text-white/70"
          >
            <ChevronLeftIcon />
            Выйти в приложение
          </Link>
        </nav>

        <div className="flex items-center gap-2 border-t border-white/[0.06] px-3.5 py-3">
          <div className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-[#2563eb] text-[10px] font-bold text-white">
            {avatar}
          </div>
          <div className="min-w-0 flex-1">
            <strong className="block truncate text-[12px] font-medium text-white">{email}</strong>
            <span className="text-[11px] text-white/60">Superadmin</span>
          </div>
          <button
            type="button"
            className="ml-auto text-white/35 hover:text-white/70"
            title="Выйти"
            onClick={signOut}
          >
            <LogoutIcon />
          </button>
        </div>
      </aside>

      <div className="ml-[220px] flex min-h-screen flex-col">
        <header className="sticky top-0 z-10 flex h-[50px] shrink-0 items-center gap-3 border-b border-[#e0ded8] bg-[#f3f2ef] px-6">
          <div className="flex items-center gap-1.5 text-[12px] text-[#7a746b]">
            <span>Admin</span>
            <span className="text-[#c7c2ba]">›</span>
            <span className="font-medium text-[#18212f]">{breadcrumb}</span>
          </div>
          <div className="ml-auto flex items-center gap-2">
            <div className="relative">
              <SearchIcon className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-[#8a857d]" />
              <input
                type="search"
                disabled
                placeholder="Search (скоро)"
                className="h-[30px] w-[220px] rounded-lg border border-[#d7d2ca] bg-[#ebe9e4] py-0 pl-8 pr-2.5 text-[12px] text-[#18212f] placeholder:text-[#8a857d] disabled:opacity-70"
              />
            </div>
            <Link
              href="#alerts"
              className="relative grid h-[30px] w-[30px] place-items-center rounded-lg border border-[#d7d2ca] text-[#6f6a62] hover:bg-[#ebe9e4]"
              aria-label="Alerts"
            >
              <BellIcon />
              <span className="absolute right-1 top-1 h-1.5 w-1.5 rounded-full bg-red-600 ring-2 ring-[#f3f2ef]" />
            </Link>
          </div>
        </header>

        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}

function AdminLogoIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <circle cx="4" cy="7" r="2" fill="white" opacity=".9" />
      <circle cx="11" cy="3.5" r="1.4" fill="white" opacity=".6" />
      <circle cx="11" cy="10.5" r="1.4" fill="white" opacity=".6" />
      <line x1="5.8" y1="6.2" x2="9.7" y2="4" stroke="white" strokeWidth=".9" opacity=".55" />
      <line x1="5.8" y1="7.8" x2="9.7" y2="10" stroke="white" strokeWidth=".9" opacity=".55" />
    </svg>
  );
}

function IconBase({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <svg
      className={`h-3.5 w-3.5 shrink-0 opacity-60 ${className}`}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {children}
    </svg>
  );
}

function DashboardIcon() {
  return <IconBase><path d="M4 5h7v6H4z" /><path d="M13 5h7v14h-7z" /><path d="M4 13h7v6H4z" /></IconBase>;
}
function BellIcon(props: { className?: string } = {}) {
  return <IconBase className={props.className}><path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 7h18s-3 0-3-7" /><path d="M10 19a2 2 0 0 0 4 0" /></IconBase>;
}
function InboxIcon() {
  return <IconBase><path d="M4 4h16v16H4z" /><path d="M4 13h4l2 3h4l2-3h4" /></IconBase>;
}
function UsersIcon() {
  return <IconBase><path d="M16 21v-2a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v2" /><circle cx="9.5" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></IconBase>;
}
function UserIcon() {
  return <IconBase><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" /><circle cx="12" cy="7" r="4" /></IconBase>;
}
function FactoryIcon() {
  return <IconBase><path d="M3 21h18" /><path d="M5 21V9l5 3V9l5 3V5h4v16" /><path d="M9 17h1" /><path d="M14 17h1" /></IconBase>;
}
function BriefcaseIcon() {
  return <IconBase><rect x="3" y="7" width="18" height="13" rx="2" /><path d="M8 7V5a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" /><path d="M3 12h18" /></IconBase>;
}
function WrenchIcon() {
  return <IconBase><path d="M14.7 6.3a4 4 0 0 0-5 5L3 18l3 3 6.7-6.7a4 4 0 0 0 5-5l-2.4 2.4-3-3z" /></IconBase>;
}
function ChartIcon() {
  return <IconBase><path d="M3 3v18h18" /><path d="m7 15 4-4 3 3 5-7" /></IconBase>;
}
function CardIcon() {
  return <IconBase><rect x="3" y="5" width="18" height="14" rx="2" /><path d="M3 10h18" /></IconBase>;
}
function FileIcon() {
  return <IconBase><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><path d="M14 2v6h6" /></IconBase>;
}
function TicketIcon() {
  return <IconBase><path d="M4 7a2 2 0 0 1 2-2h12v4a2 2 0 0 0 0 4v4H6a2 2 0 0 1-2-2v-4a2 2 0 0 0 0-4Z" /></IconBase>;
}
function CpuIcon() {
  return <IconBase><rect x="7" y="7" width="10" height="10" rx="1" /><path d="M9 1v3" /><path d="M15 1v3" /><path d="M9 20v3" /><path d="M15 20v3" /><path d="M20 9h3" /><path d="M20 15h3" /><path d="M1 9h3" /><path d="M1 15h3" /></IconBase>;
}
function SparkIcon() {
  return <IconBase><path d="m12 3 1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" /><path d="m5 3 .8 2.2L8 6l-2.2.8L5 9l-.8-2.2L2 6l2.2-.8z" /></IconBase>;
}
function CloudIcon() {
  return <IconBase><path d="M17.5 19H7a5 5 0 1 1 1.2-9.85A6 6 0 0 1 19 12.5a3.5 3.5 0 0 1-1.5 6.5Z" /></IconBase>;
}
function FlagIcon() {
  return <IconBase><path d="M5 22V4" /><path d="M5 4h12l-2 5 2 5H5" /></IconBase>;
}
function AuditIcon() {
  return <IconBase><path d="M9 11h6" /><path d="M9 15h6" /><path d="M5 3h14v18H5z" /></IconBase>;
}
function HealthIcon() {
  return <IconBase><path d="M22 12h-4l-3 7L9 5l-3 7H2" /></IconBase>;
}
function MailIcon() {
  return <IconBase><rect x="3" y="5" width="18" height="14" rx="2" /><path d="m3 7 9 6 9-6" /></IconBase>;
}
function ChevronLeftIcon() {
  return <IconBase><path d="m15 18-6-6 6-6" /></IconBase>;
}
function LogoutIcon() {
  return <IconBase><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" /><path d="m16 17 5-5-5-5" /><path d="M21 12H9" /></IconBase>;
}
function SearchIcon(props: { className?: string } = {}) {
  return <IconBase className={props.className}><circle cx="11" cy="11" r="7" /><path d="m20 20-3.5-3.5" /></IconBase>;
}
