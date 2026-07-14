"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

const NAV = [
  { href: "/app/vendor", label: "Обзор" },
  { href: "/app/vendor/tickets", label: "Очередь эскалаций" },
];

export function VendorShell({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
}) {
  const pathname = usePathname();

  return (
    <div className="min-h-screen bg-[#131619] text-[#E6EAEE]">
      <header className="border-b border-[#2A3138] bg-[#1B2025]">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-3">
          <div>
            <p className="font-[family-name:var(--font-unbounded)] text-[10px] uppercase tracking-[0.18em] text-[#3FC8B7]">
              ASUTPORT
            </p>
            <p className="text-sm text-[#93A0AC]">Кабинет производителя</p>
          </div>
          <Link href="/login" className="text-sm text-[#93A0AC] hover:text-[#3FC8B7]">
            Выход
          </Link>
        </div>
      </header>

      <div className="mx-auto flex max-w-6xl gap-6 px-4 py-6">
        <nav className="hidden w-48 shrink-0 flex-col gap-1 sm:flex">
          {NAV.map((item) => {
            const active =
              item.href === "/app/vendor"
                ? pathname === "/app/vendor" || pathname === "/vendor"
                : pathname.startsWith(item.href) || pathname.startsWith(item.href.replace("/app", ""));
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`rounded-lg px-3 py-2 text-sm ${
                  active
                    ? "bg-[#21272D] text-[#3FC8B7]"
                    : "text-[#93A0AC] hover:bg-[#21272D] hover:text-[#E6EAEE]"
                }`}
              >
                {item.label}
              </Link>
            );
          })}
        </nav>

        <main className="min-w-0 flex-1">
          <div className="mb-6">
            <h1 className="text-xl font-semibold">{title}</h1>
            {subtitle ? <p className="mt-1 text-sm text-[#93A0AC]">{subtitle}</p> : null}
          </div>
          {children}
        </main>
      </div>
    </div>
  );
}
