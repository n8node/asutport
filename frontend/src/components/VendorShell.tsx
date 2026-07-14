"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { fetchAccountProfile, vendorOrgLabel } from "@/lib/cabinet-routing";
import { orgDisplayName } from "@/lib/client-dashboard";
import { fetchVendorDashboard } from "@/lib/vendor-dashboard";

type VendorShellProps = {
  children: ReactNode;
  activePath?: string;
  pageTitle?: string;
  reviewBanner?: boolean;
};

type NavItem = {
  label: string;
  href: string;
  icon: () => ReactNode;
  requiresActive?: boolean;
  badge?: string;
};

type NavSection = {
  title: string;
  items: NavItem[];
};

const base = "/app/vendor";
const onboardingPath = `${base}/onboarding`;

function buildNavSections(pendingReview: boolean, openEscalations: number): NavSection[] {
  const queueBadge = pendingReview ? undefined : openEscalations > 0 ? String(openEscalations) : undefined;
  return [
    {
      title: "Обзор",
      items: [{ label: "Сводка", href: base, icon: DashboardIcon }],
    },
    {
      title: "Поддержка",
      items: [
        {
          label: "Очередь эскалаций",
          href: `${base}/tickets`,
          icon: TicketIcon,
          badge: queueBadge,
          requiresActive: true,
        },
      ],
    },
    {
      title: "Организация",
      items: [
        ...(pendingReview
          ? [{ label: "Статус компании", href: onboardingPath, icon: StatusIcon, badge: "!" }]
          : []),
      ],
    },
  ];
}

function initials(email: string) {
  const local = email.split("@")[0] || "user";
  const parts = local.split(/[._-]+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0]?.[0] || ""}${parts[1]?.[0] || ""}`.toUpperCase();
  }
  return local.slice(0, 2).toUpperCase();
}

export function VendorShell({
  children,
  activePath = base,
  pageTitle = "Сводка",
  reviewBanner = false,
}: VendorShellProps) {
  const [email, setEmail] = useState("user@asutport.ru");
  const [orgName, setOrgName] = useState("Организация");
  const [orgBadge, setOrgBadge] = useState("Партнёр");
  const [reviewStatus, setReviewStatus] = useState("");
  const [openEscalations, setOpenEscalations] = useState(0);
  const avatar = useMemo(() => initials(email), [email]);
  const pendingReview = reviewStatus === "pending_review" || reviewBanner;
  const navSections = useMemo(
    () => buildNavSections(pendingReview, openEscalations),
    [pendingReview, openEscalations],
  );

  useEffect(() => {
    const token = sessionStorage.getItem("asutport_access_token");
    if (!token) return;

    void fetchAccountProfile()
      .then((me) => {
        if (!me) return;
        if (me.user?.email) setEmail(me.user.email);
        if (me.org) {
          setOrgName(orgDisplayName(me.org));
          setOrgBadge(vendorOrgLabel(me.org.type));
          if (me.org.review_status) setReviewStatus(me.org.review_status);
        }
      })
      .catch(() => undefined);

    if (!reviewBanner && reviewStatus !== "pending_review") {
      void fetchVendorDashboard()
        .then((summary) => {
          if (summary?.open_escalations_count != null) {
            setOpenEscalations(summary.open_escalations_count);
          }
        })
        .catch(() => undefined);
    }
  }, [reviewBanner, reviewStatus]);

  function signOut() {
    sessionStorage.removeItem("asutport_access_token");
    sessionStorage.removeItem("asutport_refresh_token");
    window.location.href = "/app/login";
  }

  return (
    <div className="min-h-screen bg-[#f3f2ef] text-[#18212f]">
      <aside className="fixed left-0 top-0 z-20 flex h-screen w-[220px] shrink-0 flex-col border-r border-[#dedbd3] bg-white">
        <div className="flex items-center gap-2 border-b border-[#e5e1da] px-5 pb-4 pt-5">
          <div className="grid h-7 w-7 shrink-0 place-items-center rounded-md bg-[#18212f]">
            <LogoIcon />
          </div>
          <div className="min-w-0">
            <span className="block truncate text-sm font-medium text-[#18212f]">ASUTPORT</span>
            <span className="mt-1 inline-flex rounded border border-[#d7d2ca] bg-[#ebe9e4] px-1.5 py-px text-[9px] font-semibold uppercase tracking-wide text-[#5f6b7a]">
              {orgBadge}
            </span>
          </div>
        </div>

        <nav className="flex-1 overflow-y-auto px-2 py-3 text-[13px]">
          {navSections.map((section) => (
            <div key={section.title}>
              {section.items.length > 0 ? (
                <>
                  <div className="px-3 pb-1 pt-3 text-[10px] font-medium uppercase tracking-[0.08em] text-[#9a948c]">
                    {section.title}
                  </div>
                  <ul className="space-y-px">
                    {section.items.map((item) => {
                      const Icon = item.icon;
                      const isActive = activePath === item.href;
                      const disabled = pendingReview && item.requiresActive;
                      const className = isActive
                        ? "flex items-center gap-2 rounded-md bg-[#ebe9e4] px-3 py-[7px] font-medium text-[#18212f]"
                        : disabled
                          ? "flex cursor-not-allowed items-center gap-2 rounded-md px-3 py-[7px] text-[#b5b0a8]"
                          : "flex items-center gap-2 rounded-md px-3 py-[7px] text-[#5f6b7a] transition-colors hover:bg-[#ebe9e4] hover:text-[#18212f]";

                      return (
                        <li key={item.label}>
                          {disabled ? (
                            <span className={className} aria-disabled="true">
                              <Icon />
                              <span className="flex-1 truncate">{item.label}</span>
                              {item.badge ? (
                                <span className="rounded-[10px] bg-[#ebe9e4] px-1.5 py-0.5 text-[10px] font-semibold text-[#8a857d]">
                                  {item.badge}
                                </span>
                              ) : null}
                            </span>
                          ) : (
                            <Link href={item.href} className={className}>
                              <Icon />
                              <span className="flex-1 truncate">{item.label}</span>
                              {item.badge ? (
                                <span className="rounded-[10px] bg-[#e6f1fb] px-1.5 py-0.5 text-[10px] font-semibold text-[#185fa5]">
                                  {item.badge}
                                </span>
                              ) : null}
                            </Link>
                          )}
                        </li>
                      );
                    })}
                  </ul>
                </>
              ) : null}
            </div>
          ))}
        </nav>

        <div className="border-t border-[#e5e1da] px-3 pb-3 pt-3">
          <div className="mb-2.5 flex items-center gap-2">
            <div className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-[#18212f] text-[10px] font-semibold text-white">
              {avatar}
            </div>
            <div className="min-w-0 flex-1">
              <div className="truncate text-[12px] font-medium text-[#18212f]">{email.split("@")[0]}</div>
              <div className="truncate text-[10px] text-[#8a857d]">{orgBadge}</div>
            </div>
          </div>

          <div className="rounded-[10px] border border-[#e8d9b3] bg-[#f6f0df] px-2.5 py-2 text-[11px]">
            {pendingReview ? (
              <>
                <span className="flex items-center gap-2 font-semibold text-[#6d4a1f]">
                  <span className="h-1.5 w-1.5 rounded-full bg-[#ba7517]" />
                  Организация на проверке
                </span>
                <Link href={onboardingPath} className="mt-1 block pl-3.5 text-[10px] text-[#9f7a3b] underline">
                  Открыть статус и переписку
                </Link>
              </>
            ) : (
              <>
                <span className="flex items-center gap-2 font-semibold text-[#6d4a1f]">
                  <span className="h-1.5 w-1.5 rounded-full bg-[#3b6d11]" />
                  Организация активна
                </span>
                <Link href={`${base}/tickets`} className="mt-1 block truncate pl-3.5 text-[10px] text-[#9f7a3b] underline">
                  Открыть очередь эскалаций
                </Link>
              </>
            )}
          </div>

          <div className="mt-2.5 flex items-center justify-between gap-2 border-t border-[#e5e1da] pt-2">
            <button type="button" onClick={signOut} className="text-[11px] text-[#8a857d] hover:text-[#18212f]">
              Выйти
            </button>
          </div>
        </div>
      </aside>

      <div className="min-h-screen pl-[220px]">
        <header className="sticky top-0 z-10 flex h-[52px] items-center justify-between border-b border-[#dedbd3] bg-[#f3f2ef] px-7">
          <div>
            <span className="text-[15px] font-medium text-[#18212f]">{orgName}</span>
            <span className="ml-2 text-[12px] text-[#8a857d]">/ {pageTitle}</span>
          </div>
        </header>

        <main className="max-w-6xl p-7 text-[13px]">
          {pendingReview && activePath === base ? (
            <div className="mb-6 rounded-lg border border-[#e8d9b3] bg-[#f6f0df] px-4 py-3 text-[13px] text-[#6d4a1f]">
              Организация ожидает проверки платформой.{" "}
              <Link href={onboardingPath} className="font-medium underline">
                Откройте «Статус компании»
              </Link>{" "}
              и приложите подтверждающие документы. Очередь эскалаций откроется после активации.
            </div>
          ) : null}
          {children}
        </main>
      </div>
    </div>
  );
}

function IconBase({ children }: { children: ReactNode }) {
  return (
    <svg className="h-3.5 w-3.5 shrink-0 opacity-70" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {children}
    </svg>
  );
}

function LogoIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <circle cx="4" cy="8" r="2" fill="white" opacity="0.9" />
      <circle cx="12" cy="4" r="1.5" fill="white" opacity="0.6" />
      <circle cx="12" cy="12" r="1.5" fill="white" opacity="0.6" />
      <line x1="6" y1="8" x2="10.5" y2="4.5" stroke="white" strokeWidth="0.8" opacity="0.5" />
      <line x1="6" y1="8" x2="10.5" y2="11.5" stroke="white" strokeWidth="0.8" opacity="0.5" />
    </svg>
  );
}
function DashboardIcon() { return <IconBase><path d="M4 5h7v6H4z" /><path d="M13 5h7v14h-7z" /><path d="M4 13h7v6H4z" /></IconBase>; }
function TicketIcon() { return <IconBase><path d="M4 7a2 2 0 0 1 2-2h12v4a2 2 0 0 0 0 4v4H6a2 2 0 0 1-2-2v-4a2 2 0 0 0 0-4Z" /></IconBase>; }
function StatusIcon() { return <IconBase><path d="M4 21h16" /><path d="M6 21V7l6-4 6 4v14" /><path d="M10 11h4" /><path d="M10 15h4" /></IconBase>; }
