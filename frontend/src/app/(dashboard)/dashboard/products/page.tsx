"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { DashboardShell } from "@/components/DashboardShell";
import {
  PRODUCT_KIND_LABELS,
  deleteProduct,
  fetchInstallations,
  fetchProducts,
  saveProduct,
  type Installation,
  type InstallationProduct,
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

export default function ProductsPage() {
  const [installations, setInstallations] = useState<Installation[]>([]);
  const [products, setProducts] = useState<InstallationProduct[]>([]);
  const [installationID, setInstallationID] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState({
    id: "",
    manufacturer_name: "",
    product_name: "",
    kind: "plc",
    version: "",
    notes: "",
  });

  async function reload(instID: string) {
    if (!instID) {
      setProducts([]);
      return;
    }
    setProducts(await fetchProducts(instID));
  }

  useEffect(() => {
    void fetchInstallations().then(async (list) => {
      setInstallations(list);
      const id = list[0]?.id || "";
      setInstallationID(id);
      await reload(id);
      setLoading(false);
    });
  }, []);

  useEffect(() => {
    if (installationID) void reload(installationID);
  }, [installationID]);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    if (!installationID) {
      setError("Сначала создайте профиль установки");
      return;
    }
    setError("");
    const result = await saveProduct(installationID, form);
    if (!result.ok) {
      setError(result.error || "Ошибка");
      return;
    }
    setShowForm(false);
    setForm({ id: "", manufacturer_name: "", product_name: "", kind: "plc", version: "", notes: "" });
    await reload(installationID);
  }

  async function onDelete(id: string) {
    if (!confirm("Удалить продукт и связанные серийники?")) return;
    await deleteProduct(id);
    await reload(installationID);
  }

  return (
    <DashboardShell activePath="/app/dashboard/products" pageTitle="Продукты и версии">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Продукты и версии</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Оборудование и ПО на вашем объекте — не каталог вендоров, а ваш «зоопарк» для агента и маршрутизации.
        </p>
      </div>

      {installations.length === 0 && !loading ? (
        <DashboardEmpty title="Нет профиля установки">
          <Link href="/app/dashboard/installation" className="text-[#185fa5] underline">Создайте профиль установки</Link>, затем добавьте продукты.
        </DashboardEmpty>
      ) : null}

      {installations.length > 0 ? (
        <>
          <div className="mb-4 flex flex-wrap items-center gap-3">
            {installations.length > 1 ? (
              <SelectInput value={installationID} onChange={(e) => setInstallationID(e.target.value)} className="max-w-xs">
                {installations.map((i) => (
                  <option key={i.id} value={i.id}>{i.name}</option>
                ))}
              </SelectInput>
            ) : null}
            <PrimaryButton onClick={() => setShowForm((v) => !v)}>{showForm ? "Скрыть" : "Добавить продукт"}</PrimaryButton>
          </div>

          {showForm ? (
            <form onSubmit={onSubmit} className="mb-6 rounded-lg border border-[#dedbd3] bg-white p-5">
              {error ? <div className="mb-4"><ErrorNote>{error}</ErrorNote></div> : null}
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <FieldLabel>Производитель</FieldLabel>
                  <TextInput value={form.manufacturer_name} onChange={(e) => setForm({ ...form, manufacturer_name: e.target.value })} />
                </div>
                <div>
                  <FieldLabel>Название продукта</FieldLabel>
                  <TextInput value={form.product_name} onChange={(e) => setForm({ ...form, product_name: e.target.value })} required />
                </div>
                <div>
                  <FieldLabel>Тип</FieldLabel>
                  <SelectInput value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}>
                    {Object.entries(PRODUCT_KIND_LABELS).map(([k, v]) => (
                      <option key={k} value={k}>{v}</option>
                    ))}
                  </SelectInput>
                </div>
                <div>
                  <FieldLabel>Версия</FieldLabel>
                  <TextInput value={form.version} onChange={(e) => setForm({ ...form, version: e.target.value })} placeholder="2.3.1" />
                </div>
              </div>
              <div className="mt-4">
                <PrimaryButton type="submit">Сохранить</PrimaryButton>
              </div>
            </form>
          ) : null}

          {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}
          {!loading && products.length === 0 ? (
            <DashboardEmpty title="Продукты не добавлены">Укажите ПЛК, SCADA, приводы и версии — без этого агент не сможет фильтровать документацию.</DashboardEmpty>
          ) : null}

          <div className="space-y-2">
            {products.map((p) => (
              <div key={p.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-[#dedbd3] bg-white px-4 py-3">
                <div>
                  <div className="font-medium text-[#18212f]">{p.product_name}</div>
                  <div className="mt-1 text-[12px] text-[#8a857d]">
                    {p.manufacturer_name || "—"} · {PRODUCT_KIND_LABELS[p.kind] || p.kind} · v{p.version || "?"}
                  </div>
                </div>
                <SecondaryButton onClick={() => onDelete(p.id)}>Удалить</SecondaryButton>
              </div>
            ))}
          </div>
        </>
      ) : null}
    </DashboardShell>
  );
}
