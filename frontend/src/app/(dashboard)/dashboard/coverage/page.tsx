"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import { ClientPageHeader } from "@/components/cabinet/ClientPageHeader";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";
import {
  fetchDashboardSummary,
  fetchInstallations,
  fetchClientTickets,
  type Installation,
  type InstallationProduct,
  type SupplyRecord,
} from "@/lib/client-dashboard";
import { authFetch } from "@/lib/auth-session";

type ProductRow = InstallationProduct & { installation_name?: string };

export default function CoveragePage() {
  const [summary, setSummary] = useState<{ coverage_percent: number; products_count: number; supply_records_count: number } | null>(null);
  const [products, setProducts] = useState<ProductRow[]>([]);
  const [supply, setSupply] = useState<SupplyRecord[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      const [s, installations, tickets] = await Promise.all([
        fetchDashboardSummary(),
        fetchInstallations(),
        fetchClientTickets(),
      ]);
      setSummary({
        coverage_percent: s?.coverage_percent ?? 0,
        products_count: s?.products_count ?? 0,
        supply_records_count: s?.supply_records_count ?? 0,
      });

      const allProducts: ProductRow[] = [];
      for (const inst of installations) {
        const response = await authFetch(`/api/v1/client/installations/${inst.id}/products`);
        const body = (await response.json()) as { data?: InstallationProduct[] };
        if (response.ok && body.data) {
          for (const p of body.data) {
            allProducts.push({ ...p, installation_name: inst.name });
          }
        }
      }
      setProducts(allProducts);

      const supplyRes = await authFetch("/api/v1/client/supply-records");
      const supplyBody = (await supplyRes.json()) as { data?: SupplyRecord[] };
      if (supplyRes.ok) setSupply(supplyBody.data ?? []);

      void tickets;
      setLoading(false);
    })();
  }, []);

  return (
    <DashboardShell activePath="/app/dashboard/coverage" pageTitle="Покрытие">
      <ClientPageHeader
        title="Покрытие поддержки"
        subtitle="Какие продукты на установке могут быть закрыты агентом и эскалацией на платформе, а где нужен фолбэк к контрагенту вне ASUTPORT."
      />

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}

      {!loading && summary ? (
        <div className="mb-6 grid gap-2.5 sm:grid-cols-3">
          <MetricCard label="Полнота профиля" value={`${summary.coverage_percent}%`} />
          <MetricCard label="Продуктов" value={String(summary.products_count)} />
          <MetricCard label="Записей поставок" value={String(summary.supply_records_count)} />
        </div>
      ) : null}

      {!loading && products.length === 0 ? (
        <DashboardEmpty title="Профиль установки пуст">
          Добавьте продукты в разделе{" "}
          <Link href="/app/dashboard/products" className="text-[#185fa5] underline">
            «Продукты и версии»
          </Link>{" "}
          — от этого зависит маршрутизация тикетов и ответы агента.
        </DashboardEmpty>
      ) : null}

      {!loading && products.length > 0 ? (
        <div className="space-y-3">
          {products.map((p) => {
            const records = supply.filter((r) => r.installation_product_id === p.id);
            return (
              <DashboardPanel key={p.id} title={`${p.product_name} · ${p.version || "версия не указана"}`}>
                <div className="text-[12px] text-[#8a857d]">
                  {p.installation_name} · {p.manufacturer_name || "Производитель не указан"}
                </div>
                <ul className="mt-3 space-y-2 text-[13px] leading-5 text-[#5f6b7a]">
                  <li>
                    <strong>Дефект / документация:</strong>{" "}
                    {p.manufacturer_name
                      ? `эскалация на «${p.manufacturer_name}», если производитель активен на платформе`
                      : "укажите производителя в профиле продукта"}
                  </li>
                  <li>
                    <strong>Гарантия:</strong>{" "}
                    {records[0]?.supplier_name
                      ? `маршрут на «${records[0].supplier_name}»`
                      : "добавьте поставщика в серийниках и гарантии"}
                  </li>
                  <li>
                    <strong>Прикладной код:</strong>{" "}
                    {records[0]?.integrator_name
                      ? `маршрут на «${records[0].integrator_name}»`
                      : "укажите интегратора в записи поставки"}
                  </li>
                </ul>
              </DashboardPanel>
            );
          })}
        </div>
      ) : null}

      <div className="mt-6">
        <DashboardPanel title="Если сторона не на платформе">
          <p className="text-[13px] leading-5 text-[#6f6a62]">
            ASUTPORT фиксирует фолбэк: клиент получает контакты и готовый текст обращения, платформа продолжает вести тикет.
            Подключение производителя или поставщика улучшает покрытие и SLA по договору.
          </p>
        </DashboardPanel>
      </div>
    </DashboardShell>
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-[#dedbd3] bg-white p-4">
      <div className="text-[10px] font-medium uppercase tracking-wide text-[#9a948c]">{label}</div>
      <div className="mt-1 text-2xl font-medium text-[#18212f]">{value}</div>
    </div>
  );
}
