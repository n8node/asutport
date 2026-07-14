"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import { SlaTimer } from "@/components/dashboard/SlaTimer";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";
import {
  TICKET_PRIORITY_LABELS,
  TICKET_STATUS_LABELS,
  ballOwnerLabel,
  fetchClientMeProfile,
  fetchClientTickets,
  type ClientTicket,
} from "@/lib/client-dashboard";

export default function SlaPage() {
  const [tickets, setTickets] = useState<ClientTicket[]>([]);
  const [clientOrgID, setClientOrgID] = useState<string>();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void Promise.all([
      fetchClientTickets(),
      fetchClientMeProfile(),
    ])
      .then(([items, me]) => {
        setTickets(items.filter((t) => !["resolved", "closed"].includes(t.status)));
        setClientOrgID(me?.org?.id);
      })
      .finally(() => setLoading(false));
  }, []);

  const withSla = tickets.filter((t) => t.sla_reaction_deadline);

  return (
    <DashboardShell activePath="/app/dashboard/sla" pageTitle="SLA-таймеры">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">SLA-таймеры</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Обратный отсчёт до нормативного срока реакции. Таймеры считаются на сервере; здесь только отображение.
        </p>
      </div>

      <DashboardPanel title="Мяч на стороне">
        <p className="text-[13px] leading-5 text-[#6f6a62]">
          Показывает, чья очередь отвечать: ваша организация, платформа или производитель.
        </p>
      </DashboardPanel>

      <div className="mt-6">
        {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}
        {!loading && withSla.length === 0 ? (
          <DashboardEmpty title="Активных SLA-таймеров нет">
            Таймеры появятся после создания обращений с подключённым тарифом. Сейчас для новых тикетов действуют базовые нормативы реакции.
          </DashboardEmpty>
        ) : null}

        <div className="space-y-2">
          {withSla.map((t) => (
            <Link
              key={t.id}
              href={`/app/dashboard/tickets/${t.id}`}
              className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-[#dedbd3] bg-white px-4 py-3 hover:bg-[#faf9f7]"
            >
              <div>
                <div className="font-medium text-[#18212f]">{t.subject}</div>
                <div className="mt-1 text-[12px] text-[#8a857d]">
                  {TICKET_PRIORITY_LABELS[t.priority] || t.priority} · {TICKET_STATUS_LABELS[t.status] || t.status}
                </div>
                <div className="mt-1 text-[12px] text-[#6f6a62]">
                  Мяч: {ballOwnerLabel(t, clientOrgID)}
                  {t.assigned_target_org_name ? ` · Адресат: ${t.assigned_target_org_name}` : ""}
                </div>
              </div>
              <div className="text-right">
                <div className="text-[10px] uppercase tracking-wide text-[#9a948c]">До реакции</div>
                <SlaTimer deadline={t.sla_reaction_deadline} />
              </div>
            </Link>
          ))}
        </div>
      </div>
    </DashboardShell>
  );
}
