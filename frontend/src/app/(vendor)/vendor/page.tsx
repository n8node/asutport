"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { VendorShell } from "@/components/VendorShell";
import { fetchAccountProfile } from "@/lib/cabinet-routing";
import { fetchVendorDashboard, fetchVendorTickets } from "@/lib/vendor-dashboard";

export default function VendorPage() {
  const router = useRouter();
  const [openCount, setOpenCount] = useState<number | null>(null);
  const [waitingOnVendor, setWaitingOnVendor] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void fetchAccountProfile().then((profile) => {
      if (profile?.org?.review_status === "pending_review") {
        router.replace("/app/vendor/onboarding");
      }
    });
  }, [router]);

  useEffect(() => {
    void Promise.all([fetchVendorDashboard(), fetchVendorTickets()])
      .then(([summary, tickets]) => {
        setOpenCount(summary?.open_escalations_count ?? 0);
        setWaitingOnVendor(tickets.filter((t) => t.status === "waiting_vendor").length);
      })
      .catch(() => {
        setOpenCount(0);
        setWaitingOnVendor(0);
      })
      .finally(() => setLoading(false));
  }, []);

  return (
    <VendorShell activePath="/app/vendor" pageTitle="Сводка">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Сводка</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Эскалации от клиентов, назначенные вашей организации по профилю установки и типу вопроса.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-[#dedbd3] bg-white p-4">
          <div className="text-[10px] font-medium uppercase tracking-[0.08em] text-[#9a948c]">Открытые эскалации</div>
          <div className="mt-2 font-mono text-3xl font-medium text-[#18212f]">
            {loading ? "…" : openCount ?? 0}
          </div>
        </div>
        <div className="rounded-lg border border-[#dedbd3] bg-white p-4">
          <div className="text-[10px] font-medium uppercase tracking-[0.08em] text-[#9a948c]">Мяч у вас</div>
          <div className="mt-2 font-mono text-3xl font-medium text-[#ba7517]">
            {loading ? "…" : waitingOnVendor}
          </div>
        </div>
      </div>

      <div className="mt-6">
        <Link
          href="/app/vendor/tickets"
          className="inline-flex rounded-lg bg-[#18212f] px-4 py-2 text-sm font-medium text-white hover:opacity-90"
        >
          Перейти в очередь
        </Link>
      </div>
    </VendorShell>
  );
}
