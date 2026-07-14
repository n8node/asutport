"use client";

import { useEffect, useState } from "react";
import { AdminShell } from "@/components/AdminShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";
import { fetchAdminBillingOverview, fetchAdminPlans, formatRub, type AdminPlan } from "@/lib/billing";

export default function AdminBillingPage() {
  const [overview, setOverview] = useState<Awaited<ReturnType<typeof fetchAdminBillingOverview>>>(null);
  const [clientPlans, setClientPlans] = useState<AdminPlan[]>([]);
  const [vendorPlans, setVendorPlans] = useState<AdminPlan[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void Promise.all([
      fetchAdminBillingOverview(),
      fetchAdminPlans("client"),
      fetchAdminPlans("manufacturer"),
      fetchAdminPlans("partner"),
    ]).then(([ov, client, mfg, partner]) => {
      setOverview(ov);
      setClientPlans(client);
      setVendorPlans([...mfg, ...partner]);
    }).finally(() => setLoading(false));
  }, []);

  return (
    <AdminShell breadcrumb="Биллинг">
      <div className="mx-auto max-w-6xl">
        <div className="mb-6">
          <h1 className="text-2xl font-medium text-[#18212f]">Биллинг платформы</h1>
          <p className="mt-1 text-sm text-[#8a857d]">
            MRR по типам организаций, тарифы клиентов и производителей, ручные инвойсы на MVP.
          </p>
        </div>

        {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}

        <div id="revenue" className="mb-6 grid gap-4 sm:grid-cols-3">
          <MetricCard
            label="MRR (все роли)"
            value={overview ? formatRub(overview.mrr_total_rub) : "—"}
            note={`${overview?.active_subscriptions ?? 0} активных подписок`}
          />
          <MetricCard
            label="Клиенты"
            value={overview ? formatRub(overview.mrr_client_rub) : "—"}
            note="Подписки эксплуатации"
          />
          <MetricCard
            label="Производители + партнёры"
            value={overview ? formatRub(overview.mrr_manufacturer_rub + overview.mrr_partner_rub) : "—"}
            note="Подписки вендоров и канала"
          />
        </div>

        <div id="plans" className="mb-6 grid gap-4 lg:grid-cols-2">
          <DashboardPanel title="Тарифы клиентов">
            <PlanList plans={clientPlans} />
          </DashboardPanel>
          <DashboardPanel title="Тарифы производителей и партнёров">
            <PlanList plans={vendorPlans} />
          </DashboardPanel>
        </div>

        <div id="invoices">
          <DashboardEmpty title="Инвойсы и оплаты — ручной контур">
            Фиксация оплаты через API админки. Назначение тарифа организации — через subscription endpoint.
            Генерация PDF-счёта — следующий шаг после первых договоров.
          </DashboardEmpty>
        </div>
      </div>
    </AdminShell>
  );
}

function MetricCard({ label, value, note }: { label: string; value: string; note: string }) {
  return (
    <div className="rounded-lg border border-[#dedbd3] bg-white p-4">
      <div className="text-[10px] font-medium uppercase tracking-wide text-[#9a948c]">{label}</div>
      <div className="mt-1 text-2xl font-medium text-[#18212f]">{value}</div>
      <p className="mt-1 text-[12px] text-[#8a857d]">{note}</p>
    </div>
  );
}

function PlanList({ plans }: { plans: AdminPlan[] }) {
  if (plans.length === 0) {
    return <p className="text-[13px] text-[#8a857d]">Нет тарифов</p>;
  }
  return (
    <ul className="space-y-2 text-[13px] leading-5 text-[#5f6b7a]">
      {plans.map((p) => (
        <li key={p.id}>
          <span className="font-medium text-[#18212f]">{p.name}</span>
          {" — "}
          {formatRub(p.price_monthly_rub)}/мес
          {p.ticket_quota != null ? ` · квота ${p.ticket_quota} тикетов` : ""}
          {p.is_archived ? " (архив)" : ""}
        </li>
      ))}
    </ul>
  );
}
