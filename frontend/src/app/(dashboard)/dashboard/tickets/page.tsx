"use client";

import Link from "next/link";
import { FormEvent, useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import { SlaTimer } from "@/components/dashboard/SlaTimer";
import {
  DashboardEmpty,
  DashboardPanel,
  ErrorNote,
  FieldLabel,
  PrimaryButton,
  SelectInput,
  TextArea,
  TextInput,
} from "@/components/dashboard/Ui";
import {
  TICKET_PRIORITY_LABELS,
  TICKET_STATUS_LABELS,
  TICKET_TYPE_LABELS,
  createClientTicket,
  fetchClientTickets,
  fetchInstallations,
  fetchTicketQuotaCheck,
  type ClientTicket,
  type Installation,
} from "@/lib/client-dashboard";

export default function TicketsPage() {
  const [tickets, setTickets] = useState<ClientTicket[]>([]);
  const [installations, setInstallations] = useState<Installation[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const [subject, setSubject] = useState("");
  const [text, setText] = useState("");
  const [type, setType] = useState("typical");
  const [priority, setPriority] = useState("question");
  const [installationID, setInstallationID] = useState("");
  const [quotaWarning, setQuotaWarning] = useState("");

  async function reload() {
    const [t, i] = await Promise.all([fetchClientTickets(), fetchInstallations()]);
    setTickets(t);
    setInstallations(i);
    if (!installationID && i[0]?.id) setInstallationID(i[0].id);
  }

  useEffect(() => {
    void reload().finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!showForm) return;
    void fetchTicketQuotaCheck(priority).then((check) => {
      setQuotaWarning(check?.warning ?? "");
    });
  }, [priority, showForm]);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    const result = await createClientTicket({
      subject,
      text,
      type,
      priority,
      installation_id: installationID || undefined,
    });
    setSubmitting(false);
    if (!result.ok) {
      setError(result.error || "Ошибка");
      return;
    }
    setShowForm(false);
    setSubject("");
    setText("");
    await reload();
  }

  return (
    <DashboardShell activePath="/app/dashboard/tickets" pageTitle="Тикеты">
      <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-medium text-[#18212f]">Тикеты</h1>
          <p className="mt-1 text-sm text-[#8a857d]">Единое окно: адресат определяется платформой по типу вопроса и данным об установке.</p>
        </div>
        <PrimaryButton onClick={() => setShowForm((v) => !v)}>{showForm ? "Скрыть форму" : "Новое обращение"}</PrimaryButton>
      </div>

      {showForm ? (
        <form onSubmit={onSubmit} className="mb-6 rounded-lg border border-[#dedbd3] bg-white p-5">
          <h2 className="mb-4 text-[14px] font-medium text-[#18212f]">Новое обращение</h2>
          {error ? <div className="mb-4"><ErrorNote>{error}</ErrorNote></div> : null}
          {quotaWarning ? (
            <div className="mb-4 rounded-lg border border-[#f5d9a8] bg-[#fff8eb] px-3 py-2 text-[13px] leading-5 text-[#8a5a00]">
              {quotaWarning}
            </div>
          ) : null}
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <FieldLabel>Тема</FieldLabel>
              <TextInput value={subject} onChange={(e) => setSubject(e.target.value)} required maxLength={200} />
            </div>
            <div>
              <FieldLabel>Тип вопроса</FieldLabel>
              <SelectInput value={type} onChange={(e) => setType(e.target.value)}>
                {Object.entries(TICKET_TYPE_LABELS).map(([k, v]) => (
                  <option key={k} value={k}>{v}</option>
                ))}
              </SelectInput>
            </div>
            <div>
              <FieldLabel>Приоритет</FieldLabel>
              <SelectInput value={priority} onChange={(e) => setPriority(e.target.value)}>
                {Object.entries(TICKET_PRIORITY_LABELS).map(([k, v]) => (
                  <option key={k} value={k}>{v}</option>
                ))}
              </SelectInput>
            </div>
            {installations.length ? (
              <div className="sm:col-span-2">
                <FieldLabel>Установка</FieldLabel>
                <SelectInput value={installationID} onChange={(e) => setInstallationID(e.target.value)}>
                  {installations.map((i) => (
                    <option key={i.id} value={i.id}>{i.name || "Без названия"}</option>
                  ))}
                </SelectInput>
              </div>
            ) : null}
            <div className="sm:col-span-2">
              <FieldLabel>Описание</FieldLabel>
              <TextArea rows={4} value={text} onChange={(e) => setText(e.target.value)} placeholder="Симптомы, версии, что меняли перед инцидентом…" />
            </div>
          </div>
          <div className="mt-4">
            <PrimaryButton type="submit" disabled={submitting}>{submitting ? "Отправка…" : "Создать тикет"}</PrimaryButton>
          </div>
        </form>
      ) : null}

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка…</p> : null}
      {!loading && tickets.length === 0 ? (
        <DashboardEmpty title="Обращений пока нет">Создайте первый тикет — платформа соберёт контекст и направит его нужной стороне.</DashboardEmpty>
      ) : null}

      <div className="space-y-2">
        {tickets.map((t) => (
          <Link
            key={t.id}
            href={`/app/dashboard/tickets/${t.id}`}
            className="block rounded-lg border border-[#dedbd3] bg-white px-4 py-3 transition-colors hover:bg-[#faf9f7]"
          >
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <div className="font-medium text-[#18212f]">{t.subject}</div>
                <div className="mt-1 text-[12px] text-[#8a857d]">
                  {TICKET_TYPE_LABELS[t.type] || t.type} · {TICKET_PRIORITY_LABELS[t.priority] || t.priority}
                </div>
              </div>
              <div className="text-right">
                <span className="text-[12px] font-medium text-[#5f6b7a]">{TICKET_STATUS_LABELS[t.status] || t.status}</span>
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
    </DashboardShell>
  );
}
