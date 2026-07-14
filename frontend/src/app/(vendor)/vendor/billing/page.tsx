"use client";

import { VendorShell } from "@/components/VendorShell";
import { VendorPageHeader } from "@/components/cabinet/VendorPageHeader";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";
import { useEffect, useState } from "react";
import { fetchAccountProfile, vendorOrgLabel } from "@/lib/cabinet-routing";
import { fetchVendorBilling, formatRub } from "@/lib/billing";
import type { BillingSummary } from "@/lib/client-dashboard";

export default function VendorBillingPage() {
  const [orgType, setOrgType] = useState("");
  const [summary, setSummary] = useState<BillingSummary | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void Promise.all([fetchAccountProfile(), fetchVendorBilling()]).then(([p, billing]) => {
      setOrgType(p?.org?.type ?? "");
      setSummary(billing);
    }).finally(() => setLoading(false));
  }, []);

  const isManufacturer = orgType === "manufacturer";
  const activeSlug = summary?.subscription.plan_slug;

  return (
    <VendorShell activePath="/app/vendor/billing" pageTitle="Биллинг">
      <VendorPageHeader
        title="Биллинг и тариф"
        subtitle={
          isManufacturer
            ? "Подписка производителя на платформу: канал эскалаций, разгрузка инженеров, база знаний."
            : "Подписка поставщика или интегратора на видимость клиентов и эскалации по вашим отгрузкам."
        }
      />

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}

      {summary ? (
        <DashboardPanel title="Текущая подписка">
          <p className="text-[13px] text-[#5f6b7a]">
            Тариф «{summary.plan.name}» — {formatRub(summary.plan.price_monthly_rub)}/мес.
            Статус: {summary.subscription.status === "active" ? "активна" : summary.subscription.status}.
          </p>
        </DashboardPanel>
      ) : null}

      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        {(summary?.public_plans.length ? summary.public_plans : fallbackPlans(isManufacturer)).map((plan) => (
          <TariffCard
            key={plan.slug}
            name={plan.name}
            price={plan.price_monthly_rub === 0 ? "по согласованию" : `${formatRub(plan.price_monthly_rub)}/мес`}
            note={"ticket_quota" in plan && plan.ticket_quota != null ? `Квота ${plan.ticket_quota}` : planNote(plan.slug, isManufacturer)}
            active={plan.slug === activeSlug || (!summary && plan.slug === (isManufacturer ? "basic" : "channel"))}
          />
        ))}
      </div>

      <div className="mt-6">
        <DashboardPanel title="Оплата на MVP">
          <p className="text-[13px] leading-5 text-[#6f6a62]">
            Счёт на юрлицо выставляет менеджер платформы; оплата фиксируется вручную. Автоматический платёжный провайдер — после первых платных клиентов.
          </p>
        </DashboardPanel>
      </div>

      {!summary && !loading ? (
        <div className="mt-4">
          <DashboardEmpty title="Подписка не назначена">
            После активации организации будет назначен тариф по умолчанию.
          </DashboardEmpty>
        </div>
      ) : null}

      {orgType ? (
        <p className="mt-4 text-[12px] text-[#8a857d]">Тип организации: {vendorOrgLabel(orgType)}</p>
      ) : null}
    </VendorShell>
  );
}

function fallbackPlans(isManufacturer: boolean) {
  if (isManufacturer) {
    return [
      { slug: "basic", name: "Базовый", price_monthly_rub: 110000 },
      { slug: "extended", name: "Расширенный", price_monthly_rub: 300000 },
    ];
  }
  return [{ slug: "channel", name: "Канал поддержки", price_monthly_rub: 30000 }];
}

function planNote(slug: string, isManufacturer: boolean): string {
  if (isManufacturer && slug === "basic") return "Очередь эскалаций, агент по документации, отчёт PDF";
  if (isManufacturer && slug === "extended") return "Расширенная аналитика и брендированный раздел KB";
  return "Эскалации по гарантии и отгрузкам ваших клиентов";
}

function TariffCard({ name, price, note, active = false }: { name: string; price: string; note: string; active?: boolean }) {
  return (
    <DashboardPanel title={name}>
      <div className="text-lg font-medium text-[#18212f]">{price}</div>
      <p className="mt-2 text-[12px] leading-5 text-[#6f6a62]">{note}</p>
      {active ? (
        <span className="mt-3 inline-flex rounded-full bg-[#e6f1fb] px-2 py-0.5 text-[10px] font-semibold text-[#185fa5]">
          {price.includes("согласован") ? "Пилот / по согласованию" : "Текущий"}
        </span>
      ) : null}
    </DashboardPanel>
  );
}
