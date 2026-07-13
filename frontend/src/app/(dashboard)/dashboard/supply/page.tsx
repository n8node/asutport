"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import {
  VERIFY_STATUS_LABELS,
  createSupplyRecord,
  deleteSupplyRecord,
  fetchInstallations,
  fetchProducts,
  fetchSupplyRecords,
  type InstallationProduct,
  type SupplyRecord,
} from "@/lib/client-dashboard";
import {
  DashboardEmpty,
  ErrorNote,
  FieldLabel,
  PrimaryButton,
  SecondaryButton,
  SelectInput,
  TextInput,
} from "@/components/dashboard/Ui";

export default function SupplyPage() {
  const [records, setRecords] = useState<SupplyRecord[]>([]);
  const [products, setProducts] = useState<InstallationProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    installation_product_id: "",
    serial_or_license: "",
    supplier_name: "",
    integrator_name: "",
    purchase_date: "",
    warranty_until: "",
    contract_ref: "",
  });

  async function reload() {
    const [r, installations] = await Promise.all([fetchSupplyRecords(), fetchInstallations()]);
    setRecords(r);
    const allProducts: InstallationProduct[] = [];
    for (const inst of installations) {
      const prods = await fetchProducts(inst.id);
      allProducts.push(...prods);
    }
    setProducts(allProducts);
    if (!form.installation_product_id && allProducts[0]?.id) {
      setForm((f) => ({ ...f, installation_product_id: allProducts[0].id }));
    }
  }

  useEffect(() => {
    void reload().finally(() => setLoading(false));
  }, []);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setError("");
    const result = await createSupplyRecord(form);
    if (!result.ok) {
      setError(result.error || "Ошибка");
      return;
    }
    setShowForm(false);
    await reload();
  }

  async function onDelete(id: string) {
    if (!confirm("Удалить запись?")) return;
    await deleteSupplyRecord(id);
    await reload();
  }

  return (
    <DashboardShell activePath="/app/dashboard/supply" pageTitle="Серийники и гарантия">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Серийники и гарантия</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Серийные номера, лицензии, поставщик и интегратор по каждому экземпляру. Эти данные определяют, куда уйдёт гарантийное или коммерческое обращение.
        </p>
      </div>

      {products.length === 0 && !loading ? (
        <DashboardEmpty title="Сначала добавьте продукты">
          <Link href="/app/dashboard/products" className="text-[#185fa5] underline">Перейти к продуктам и версиям</Link>
        </DashboardEmpty>
      ) : (
        <>
          <div className="mb-4">
            <PrimaryButton onClick={() => setShowForm((v) => !v)}>{showForm ? "Скрыть" : "Добавить запись"}</PrimaryButton>
          </div>

          {showForm ? (
            <form onSubmit={onSubmit} className="mb-6 max-w-2xl rounded-lg border border-[#dedbd3] bg-white p-5">
              {error ? <div className="mb-4"><ErrorNote>{error}</ErrorNote></div> : null}
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="sm:col-span-2">
                  <FieldLabel>Продукт на установке</FieldLabel>
                  <SelectInput
                    value={form.installation_product_id}
                    onChange={(e) => setForm({ ...form, installation_product_id: e.target.value })}
                    required
                  >
                    {products.map((p) => (
                      <option key={p.id} value={p.id}>{p.product_name} {p.version ? `v${p.version}` : ""}</option>
                    ))}
                  </SelectInput>
                </div>
                <div>
                  <FieldLabel>Серийный номер / лицензия</FieldLabel>
                  <TextInput value={form.serial_or_license} onChange={(e) => setForm({ ...form, serial_or_license: e.target.value })} required />
                </div>
                <div>
                  <FieldLabel>Номер договора</FieldLabel>
                  <TextInput value={form.contract_ref} onChange={(e) => setForm({ ...form, contract_ref: e.target.value })} />
                </div>
                <div>
                  <FieldLabel>Поставщик</FieldLabel>
                  <TextInput value={form.supplier_name} onChange={(e) => setForm({ ...form, supplier_name: e.target.value })} />
                </div>
                <div>
                  <FieldLabel>Интегратор проекта</FieldLabel>
                  <TextInput value={form.integrator_name} onChange={(e) => setForm({ ...form, integrator_name: e.target.value })} />
                </div>
                <div>
                  <FieldLabel>Дата поставки</FieldLabel>
                  <TextInput type="date" value={form.purchase_date} onChange={(e) => setForm({ ...form, purchase_date: e.target.value })} />
                </div>
                <div>
                  <FieldLabel>Гарантия до</FieldLabel>
                  <TextInput type="date" value={form.warranty_until} onChange={(e) => setForm({ ...form, warranty_until: e.target.value })} />
                </div>
              </div>
              <p className="mt-3 text-[12px] text-[#8a857d]">Новые записи помечаются как «данные клиента» до верификации производителем или поставщиком.</p>
              <div className="mt-4">
                <PrimaryButton type="submit">Сохранить</PrimaryButton>
              </div>
            </form>
          ) : null}
        </>
      )}

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}
      {!loading && records.length > 0 ? (
        <div className="space-y-2">
          {records.map((r) => (
            <div key={r.id} className="flex flex-wrap items-start justify-between gap-3 rounded-lg border border-[#dedbd3] bg-white px-4 py-3">
              <div>
                <div className="font-medium text-[#18212f]">{r.serial_or_license}</div>
                <div className="mt-1 text-[12px] text-[#8a857d]">
                  {r.product_name || "Продукт"} · {r.supplier_name || "поставщик не указан"}
                  {r.integrator_name ? ` · интегратор: ${r.integrator_name}` : ""}
                </div>
                <div className="mt-1 text-[11px] text-[#9a948c]">
                  {VERIFY_STATUS_LABELS[r.verify_status] || r.verify_status}
                  {r.warranty_until ? ` · гарантия до ${r.warranty_until}` : ""}
                </div>
              </div>
              <SecondaryButton onClick={() => onDelete(r.id)}>Удалить</SecondaryButton>
            </div>
          ))}
        </div>
      ) : null}
    </DashboardShell>
  );
}
