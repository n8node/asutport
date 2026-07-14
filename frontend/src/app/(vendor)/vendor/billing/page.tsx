"use client";

import { VendorShell } from "@/components/VendorShell";
import { VendorPageHeader } from "@/components/cabinet/VendorPageHeader";
import { DashboardPanel } from "@/components/dashboard/Ui";
import { useEffect, useState } from "react";
import { fetchAccountProfile, vendorOrgLabel } from "@/lib/cabinet-routing";

export default function VendorBillingPage() {
  const [orgType, setOrgType] = useState("");

  useEffect(() => {
    void fetchAccountProfile().then((p) => setOrgType(p?.org?.type ?? ""));
  }, []);

  const isManufacturer = orgType === "manufacturer";

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

      <div className="grid gap-4 sm:grid-cols-2">
        {isManufacturer ? (
          <>
            <TariffCard name="Базовый" price="~100–120 тыс ₽/мес" note="Очередь эскалаций, агент по вашей документации, ежемесячный отчёт PDF" active />
            <TariffCard name="Расширенный" price="~300 тыс ₽/мес" note="Расширенная аналитика и брендированный раздел базы знаний" />
          </>
        ) : (
          <>
            <TariffCard name="Канал поддержки" price="~20–40 тыс ₽/мес" note="Эскалации по гарантии и отгрузкам ваших клиентов на платформе" active />
            <TariffCard name="Расширенный" price="по запросу" note="Отчёты фолбэков и сигналы апсейла — после MVP" />
          </>
        )}
      </div>

      <div className="mt-6">
        <DashboardPanel title="Оплата на MVP">
          <p className="text-[13px] leading-5 text-[#6f6a62]">
            Счёт на юрлицо выставляет менеджер платформы; оплата фиксируется вручную. Автоматический платёжный провайдер — после первых платных клиентов.
          </p>
        </DashboardPanel>
      </div>

      {orgType ? (
        <p className="mt-4 text-[12px] text-[#8a857d]">Тип организации: {vendorOrgLabel(orgType)}</p>
      ) : null}
    </VendorShell>
  );
}

function TariffCard({ name, price, note, active = false }: { name: string; price: string; note: string; active?: boolean }) {
  return (
    <DashboardPanel title={name}>
      <div className="text-lg font-medium text-[#18212f]">{price}</div>
      <p className="mt-2 text-[12px] leading-5 text-[#6f6a62]">{note}</p>
      {active ? (
        <span className="mt-3 inline-flex rounded-full bg-[#e6f1fb] px-2 py-0.5 text-[10px] font-semibold text-[#185fa5]">
          Пилот / по согласованию
        </span>
      ) : null}
    </DashboardPanel>
  );
}
