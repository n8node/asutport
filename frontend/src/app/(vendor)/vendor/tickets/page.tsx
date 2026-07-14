"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { SlaTimer } from "@/components/dashboard/SlaTimer";
import { VendorShell } from "@/components/VendorShell";
import {
  VENDOR_PRIORITY_LABELS,
  VENDOR_TICKET_TYPE_LABELS,
  fetchVendorTickets,
  vendorStatusLabel,
  type VendorTicket,
} from "@/lib/vendor-dashboard";

export default function VendorTicketsPage() {
  const [tickets, setTickets] = useState<VendorTicket[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void fetchVendorTickets()
      .then(setTickets)
      .finally(() => setLoading(false));
  }, []);

  return (
    <VendorShell title="Очередь эскалаций" subtitle="Только обращения, маршрутизированные на вашу организацию.">
      {loading ? <p className="text-sm text-[#93A0AC]">Загрузка…</p> : null}
      {!loading && tickets.length === 0 ? (
        <div className="rounded-lg border border-[#2A3138] bg-[#1B2025] p-6 text-sm text-[#93A0AC]">
          Эскалаций пока нет. Они появятся, когда клиент создаст обращение с типом «дефект», «гарантия» или «прикладной»
          и в профиле установки указан ваш производитель или поставщик.
        </div>
      ) : null}

      <div className="space-y-2">
        {tickets.map((t) => (
          <Link
            key={t.id}
            href={`/app/vendor/tickets/${t.id}`}
            className="block rounded-lg border border-[#2A3138] bg-[#1B2025] px-4 py-3 transition-colors hover:bg-[#21272D]"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="font-medium text-[#E6EAEE]">{t.subject}</div>
                <div className="mt-1 text-[12px] text-[#93A0AC]">
                  {t.client_org_name || "Клиент"} · {VENDOR_TICKET_TYPE_LABELS[t.type] || t.type} ·{" "}
                  {VENDOR_PRIORITY_LABELS[t.priority] || t.priority}
                </div>
              </div>
              <div className="text-right">
                <span
                  className={`text-[12px] font-medium ${
                    t.status === "waiting_vendor" ? "text-[#F2A33C]" : "text-[#93A0AC]"
                  }`}
                >
                  {vendorStatusLabel(t.status)}
                </span>
                {t.sla_reaction_deadline ? (
                  <div className="mt-1">
                    <SlaTimer deadline={t.sla_reaction_deadline} />
                  </div>
                ) : null}
              </div>
            </div>
          </Link>
        ))}
      </div>
    </VendorShell>
  );
}
