"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { VendorShell } from "@/components/VendorShell";
import { fetchVendorDashboard, fetchVendorTickets } from "@/lib/vendor-dashboard";

export default function VendorPage() {
  const [openCount, setOpenCount] = useState<number | null>(null);
  const [waitingOnVendor, setWaitingOnVendor] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void Promise.all([fetchVendorDashboard(), fetchVendorTickets()])
      .then(([summary, tickets]) => {
        setOpenCount(summary?.open_escalations_count ?? 0);
        setWaitingOnVendor(tickets.filter((t) => t.status === "waiting_vendor").length);
      })
      .finally(() => setLoading(false));
  }, []);

  return (
    <VendorShell
      title="Обзор"
      subtitle="Эскалации от клиентов, назначенные вашей организации по профилю установки и типу вопроса."
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-[#2A3138] bg-[#1B2025] p-4">
          <div className="text-[10px] uppercase tracking-[0.14em] text-[#5F6C78]">Открытые эскалации</div>
          <div className="mt-2 font-[family-name:var(--font-jetbrains-mono)] text-3xl text-[#3FC8B7]">
            {loading ? "…" : openCount ?? 0}
          </div>
        </div>
        <div className="rounded-lg border border-[#2A3138] bg-[#1B2025] p-4">
          <div className="text-[10px] uppercase tracking-[0.14em] text-[#5F6C78]">Мяч у вас</div>
          <div className="mt-2 font-[family-name:var(--font-jetbrains-mono)] text-3xl text-[#F2A33C]">
            {loading ? "…" : waitingOnVendor}
          </div>
        </div>
      </div>

      <div className="mt-6">
        <Link
          href="/app/vendor/tickets"
          className="inline-flex rounded-lg bg-[#3FC8B7] px-4 py-2 text-sm font-medium text-[#0B2723]"
        >
          Перейти в очередь
        </Link>
      </div>
    </VendorShell>
  );
}
