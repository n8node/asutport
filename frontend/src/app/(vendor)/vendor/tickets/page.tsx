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
    <VendorShell activePath="/app/vendor/tickets" pageTitle="Очередь эскалаций">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Очередь эскалаций</h1>
        <p className="mt-1 text-sm text-[#8a857d]">Только обращения, маршрутизированные на вашу организацию.</p>
      </div>

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}
      {!loading && tickets.length === 0 ? (
        <div className="rounded-lg border border-[#dedbd3] bg-white p-6 text-sm text-[#6f6a62]">
          Эскалаций пока нет. Они появятся, когда клиент создаст обращение с типом «дефект», «гарантия» или «прикладной»
          и в профиле установки указан ваш производитель или поставщик.
        </div>
      ) : null}

      <div className="space-y-2">
        {tickets.map((t) => (
          <Link
            key={t.id}
            href={`/app/vendor/tickets/${t.id}`}
            className="block rounded-lg border border-[#dedbd3] bg-white px-4 py-3 transition-colors hover:bg-[#faf9f7]"
          >
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="font-medium text-[#18212f]">{t.subject}</div>
                <div className="mt-1 text-[12px] text-[#8a857d]">
                  {t.client_org_name || "Клиент"} · {VENDOR_TICKET_TYPE_LABELS[t.type] || t.type} ·{" "}
                  {VENDOR_PRIORITY_LABELS[t.priority] || t.priority}
                </div>
              </div>
              <div className="text-right">
                <span
                  className={`text-[12px] font-medium ${
                    t.status === "waiting_vendor" ? "text-[#ba7517]" : "text-[#5f6b7a]"
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
