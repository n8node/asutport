"use client";

import { useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import { DashboardEmpty, DashboardPanel, ErrorNote } from "@/components/dashboard/Ui";
import { fetchClientBilling, type BillingSummary } from "@/lib/client-dashboard";
import { formatRub } from "@/lib/billing";

export default function BillingPage() {
  const [summary, setSummary] = useState<BillingSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    void fetchClientBilling()
      .then((data) => {
        if (!data) setError("Не удалось загрузить данные биллинга");
        else setSummary(data);
      })
      .finally(() => setLoading(false));
  }, []);

  const activeSlug = summary?.subscription.plan_slug;

  return (
    <DashboardShell activePath="/app/dashboard/billing" pageTitle="Биллинг">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Биллинг и тариф</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Подписка организации, квота тикетов и счета на юрлицо. Аварийные обращения принимаются всегда.
        </p>
      </div>

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}
      {error ? <ErrorNote>{error}</ErrorNote> : null}

      {summary ? (
        <>
          <DashboardPanel title="Текущий период">
            <div className="grid gap-3 sm:grid-cols-3">
              <Stat label="Тариф" value={summary.plan.name} />
              <Stat
                label="Тикеты в периоде"
                value={
                  summary.ticket_quota != null
                    ? `${summary.tickets_used} / ${summary.ticket_quota}`
                    : `${summary.tickets_used} (без лимита)`
                }
              />
              <Stat label="Абонентская плата" value={formatRub(summary.plan.price_monthly_rub) + "/мес"} />
            </div>
            <p className="mt-3 text-[12px] text-[#8a857d]">
              Период: {formatDate(summary.period_start)} — {formatDate(summary.period_end)}.
              {summary.ticket_quota != null && summary.tickets_used >= summary.ticket_quota
                ? ` Сверхквота: ${formatRub(summary.overage_price_rub)} за тикет (кроме аварийных).`
                : null}
            </p>
          </DashboardPanel>

          <div className="mt-6 grid gap-4 sm:grid-cols-3">
            {summary.public_plans.map((plan) => (
              <TariffCard
                key={plan.id}
                name={plan.name}
                price={plan.price_monthly_rub === 0 ? "0 ₽" : `${formatRub(plan.price_monthly_rub)}/мес`}
                note={planNote(plan)}
                active={plan.slug === activeSlug}
              />
            ))}
          </div>

          {summary.recent_payments.length > 0 ? (
            <div className="mt-6">
              <DashboardPanel title="Последние платежи и начисления">
                <ul className="divide-y divide-[#eeeae4]">
                  {summary.recent_payments.map((p) => (
                    <li key={p.id} className="py-2 text-[13px] text-[#5f6b7a]">
                      <span className="font-medium text-[#18212f]">{formatRub(p.amount_rub)}</span>
                      {" · "}
                      {paymentTypeLabel(p.type)} · {paymentStatusLabel(p.status)}
                      {p.note ? ` — ${p.note}` : ""}
                    </li>
                  ))}
                </ul>
              </DashboardPanel>
            </div>
          ) : (
            <div className="mt-6">
              <DashboardEmpty title="Счета и оплаты">
                На MVP счёт выставляет менеджер платформы; оплата фиксируется вручную после поступления средств.
              </DashboardEmpty>
            </div>
          )}
        </>
      ) : null}
    </DashboardShell>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[10px] font-medium uppercase tracking-wide text-[#9a948c]">{label}</div>
      <div className="mt-1 font-medium text-[#18212f]">{value}</div>
    </div>
  );
}

function TariffCard({ name, price, note, active = false }: { name: string; price: string; note: string; active?: boolean }) {
  return (
    <DashboardPanel title={name}>
      <div className="text-lg font-medium text-[#18212f]">{price}</div>
      <p className="mt-2 text-[12px] leading-5 text-[#6f6a62]">{note}</p>
      {active ? (
        <span className="mt-3 inline-flex rounded-full bg-[#e6f1fb] px-2 py-0.5 text-[10px] font-semibold text-[#185fa5]">
          Текущий
        </span>
      ) : (
        <span className="mt-3 inline-flex text-[11px] text-[#8a857d]">Подключение — через менеджера</span>
      )}
    </DashboardPanel>
  );
}

function planNote(plan: { ticket_quota?: number; overage_price_rub: number; slug: string }): string {
  if (plan.slug === "free") return "Агент, база знаний, до 3 тикетов/мес без SLA";
  if (plan.ticket_quota != null) {
    return `Квота ${plan.ticket_quota} тикетов/мес, сверхквота ${formatRub(plan.overage_price_rub)}`;
  }
  return "Подписка на канал эскалаций";
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("ru-RU");
  } catch {
    return iso;
  }
}

function paymentTypeLabel(t: string): string {
  switch (t) {
    case "subscription":
      return "Подписка";
    case "overage":
      return "Сверхквота";
    case "service":
      return "Услуга";
    default:
      return t;
  }
}

function paymentStatusLabel(s: string): string {
  switch (s) {
    case "paid":
      return "оплачено";
    case "pending":
      return "ожидает оплаты";
    case "cancelled":
      return "отменено";
    default:
      return s;
  }
}
