"use client";

import { FormEvent, useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import {
  CRITICALITY_LABELS,
  fetchInstallations,
  saveInstallation,
  type Installation,
} from "@/lib/client-dashboard";
import {
  DashboardEmpty,
  ErrorNote,
  FieldLabel,
  PrimaryButton,
  SelectInput,
  TextInput,
} from "@/components/dashboard/Ui";

export default function InstallationPage() {
  const [items, setItems] = useState<Installation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState({
    id: "",
    name: "",
    site_address: "",
    criticality: "batch" as Installation["criticality"],
    snapshot_allowed: false,
    emergency_contact_name: "",
    emergency_contact_phone: "",
    os: "",
    virtualization: "",
    protocols: "",
  });

  async function load() {
    const list = await fetchInstallations();
    setItems(list);
    if (list[0]) {
      const i = list[0];
      const env = i.environment || {};
      setForm({
        id: i.id,
        name: i.name,
        site_address: i.site_address,
        criticality: i.criticality,
        snapshot_allowed: i.snapshot_allowed,
        emergency_contact_name: i.emergency_contact_name,
        emergency_contact_phone: i.emergency_contact_phone,
        os: String(env.os || ""),
        virtualization: String(env.virtualization || ""),
        protocols: String(env.protocols || ""),
      });
    }
  }

  useEffect(() => {
    void load().finally(() => setLoading(false));
  }, []);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    const result = await saveInstallation({
      id: form.id || undefined,
      name: form.name,
      site_address: form.site_address,
      criticality: form.criticality,
      snapshot_allowed: form.snapshot_allowed,
      emergency_contact_name: form.emergency_contact_name,
      emergency_contact_phone: form.emergency_contact_phone,
      environment: {
        os: form.os,
        virtualization: form.virtualization,
        protocols: form.protocols,
      },
    });
    setSaving(false);
    if (!result.ok) {
      setError(result.error || "Ошибка сохранения");
      return;
    }
    await load();
  }

  return (
    <DashboardShell activePath="/app/dashboard/installation" pageTitle="Профиль установки">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Профиль установки</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Описание производственной площадки: критичность, среда, аварийный контакт. Агент использует эти данные при каждом обращении.
        </p>
      </div>

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}

      {!loading && items.length === 0 && !form.name ? (
        <DashboardEmpty title="Профиль ещё не заполнен">
          Укажите площадку и контакты — это основа маршрутизации и персонализации ответов агента.
        </DashboardEmpty>
      ) : null}

      <form onSubmit={onSubmit} className="max-w-2xl rounded-lg border border-[#dedbd3] bg-white p-5">
        {error ? <div className="mb-4"><ErrorNote>{error}</ErrorNote></div> : null}
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <FieldLabel>Название площадки / объекта</FieldLabel>
            <TextInput value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required />
          </div>
          <div className="sm:col-span-2">
            <FieldLabel>Адрес или условное обозначение</FieldLabel>
            <TextInput value={form.site_address} onChange={(e) => setForm({ ...form, site_address: e.target.value })} />
          </div>
          <div>
            <FieldLabel>Критичность производства</FieldLabel>
            <SelectInput value={form.criticality} onChange={(e) => setForm({ ...form, criticality: e.target.value as Installation["criticality"] })}>
              {Object.entries(CRITICALITY_LABELS).map(([k, v]) => (
                <option key={k} value={k}>{v}</option>
              ))}
            </SelectInput>
          </div>
          <div className="flex items-end">
            <label className="flex items-center gap-2 text-[13px] text-[#18212f]">
              <input
                type="checkbox"
                checked={form.snapshot_allowed}
                onChange={(e) => setForm({ ...form, snapshot_allowed: e.target.checked })}
              />
              Разрешить диагностические слепки
            </label>
          </div>
          <div>
            <FieldLabel>ОС / среда исполнения</FieldLabel>
            <TextInput value={form.os} onChange={(e) => setForm({ ...form, os: e.target.value })} placeholder="Windows Server 2019, Astra Linux…" />
          </div>
          <div>
            <FieldLabel>Виртуализация</FieldLabel>
            <TextInput value={form.virtualization} onChange={(e) => setForm({ ...form, virtualization: e.target.value })} placeholder="VMware, Proxmox, bare metal…" />
          </div>
          <div className="sm:col-span-2">
            <FieldLabel>Промышленные протоколы</FieldLabel>
            <TextInput value={form.protocols} onChange={(e) => setForm({ ...form, protocols: e.target.value })} placeholder="Modbus TCP, OPC UA, PROFINET…" />
          </div>
          <div>
            <FieldLabel>Аварийный контакт (ФИО)</FieldLabel>
            <TextInput value={form.emergency_contact_name} onChange={(e) => setForm({ ...form, emergency_contact_name: e.target.value })} />
          </div>
          <div>
            <FieldLabel>Телефон дежурного</FieldLabel>
            <TextInput value={form.emergency_contact_phone} onChange={(e) => setForm({ ...form, emergency_contact_phone: e.target.value })} />
          </div>
        </div>
        <div className="mt-5">
          <PrimaryButton type="submit" disabled={saving}>{saving ? "Сохранение…" : form.id ? "Сохранить" : "Создать профиль"}</PrimaryButton>
        </div>
      </form>
    </DashboardShell>
  );
}
