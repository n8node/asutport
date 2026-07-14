"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import { DashboardPanel } from "@/components/dashboard/Ui";
import {
  fetchClientOrgProfile,
  fetchClientTickets,
  fetchDashboardSummary,
  orgDisplayName,
  type DashboardSummary,
} from "@/lib/client-dashboard";
import { SlaTimer } from "@/components/dashboard/SlaTimer";

export default function DashboardPage() {
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [companyName, setCompanyName] = useState(orgDisplayName());
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void fetchClientOrgProfile().then((org) => {
      if (org) setCompanyName(orgDisplayName(org));
    });
    void fetchDashboardSummary()
      .then(setSummary)
      .finally(() => setLoading(false));
    void fetchClientTickets().then((tickets) => {
      const open = tickets.filter((t) => !["resolved", "closed"].includes(t.status));
      if (open.length > 0) setSummary((prev) => (prev ? { ...prev, open_tickets_count: open.length } : prev));
    });
  }, []);

  const s = summary ?? {
    installations_count: 0,
    open_tickets_count: 0,
    sla_active_count: 0,
    coverage_percent: 0,
    profile_complete: false,
    products_count: 0,
    supply_records_count: 0,
  };

  return (
    <DashboardShell pageTitle="Сводка">
      <div className="mb-6">
        <h1 className="text-2xl font-medium tracking-tight text-[#18212f] sm:text-[26px]">{companyName}</h1>
        <p className="mt-1 text-sm text-[#8a857d]">Единое окно мультивендорной поддержки</p>
      </div>

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка сводки…</p> : null}

      <section className="mb-8">
        <h2 className="mb-3.5 text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">Показатели</h2>
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
          <SummaryCard label="Установки" value={String(s.installations_count)} note={s.installations_count ? "Площадки на учёте" : "Добавьте первую площадку"} tone="blue" progress={Math.min(100, s.installations_count * 40)} />
          <SummaryCard label="Открытые тикеты" value={String(s.open_tickets_count)} note={s.open_tickets_count ? "Требуют внимания" : "Новых обращений нет"} tone="green" progress={Math.min(100, s.open_tickets_count * 20)} />
          <SummaryCard label="SLA" value={s.sla_active_count ? String(s.sla_active_count) : "—"} note={s.sla_active_count ? "Активных таймеров" : "Появятся с тикетами"} tone="amber" progress={s.sla_active_count ? 60 : 0} />
          <SummaryCard label="Покрытие" value={`${s.coverage_percent}%`} note="Профиль, продукты, серийники" tone="purple" progress={s.coverage_percent} />
        </div>
      </section>

      <section className="mb-8">
        <h2 className="mb-3.5 text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">Быстрые действия</h2>
        <div className="flex flex-wrap gap-2">
          <ActionLink href="/app/dashboard/agent" primary>Описать проблему агенту</ActionLink>
          <ActionLink href="/app/dashboard/tickets">Создать тикет</ActionLink>
          <ActionLink href="/app/dashboard/installation">Профиль установки</ActionLink>
          <ActionLink href="/app/kb">База знаний</ActionLink>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-12">
        <section className="lg:col-span-7">
          <h2 className="mb-3.5 text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">Начните работу</h2>
          <div className="rounded-lg border border-[#dedbd3] bg-white p-5 sm:p-6">
            <ol className="space-y-5">
              <OnboardingStep done number="1" title="Аккаунт создан">Вы вошли в кабинет ASUTPORT.</OnboardingStep>
              <OnboardingStep done={s.profile_complete} number="2" title="Заполните профиль установки">
                Площадка, критичность производства, аварийный контакт, среда эксплуатации.
              </OnboardingStep>
              <OnboardingStep done={s.products_count > 0} number="3" title="Добавьте продукты и серийники">
                Оборудование на объекте и записи о поставках определяют маршрут эскалаций.
              </OnboardingStep>
              <OnboardingStep number="4" title="Задайте первый вопрос агенту">
                Агент ответит с цитатами из документации или предложит эскалацию.
              </OnboardingStep>
              <OnboardingStep number="5" title="Выберите тариф">
                SLA и квоты тикетов подключаются в разделе «Биллинг».
              </OnboardingStep>
            </ol>
          </div>
        </section>

        <aside className="flex flex-col gap-4 lg:col-span-5">
          <DashboardPanel title="Покрытие поддержки">
            <p className="text-[13px] leading-5 text-[#6f6a62]">
              Продуктов: <strong>{s.products_count}</strong> · Серийников: <strong>{s.supply_records_count}</strong>
            </p>
            <p className="mt-2 text-[12px] text-[#8a857d]">
              Полное покрытие возможно, когда производитель и поставщик подключены к платформе.
            </p>
          </DashboardPanel>

          <DashboardPanel title="SLA и мяч на стороне">
            <p className="text-[13px] leading-5 text-[#6f6a62]">
              {s.sla_active_count
                ? `Активных таймеров: ${s.sla_active_count}. Подробности — в разделе «SLA-таймеры».`
                : "Здесь появятся живые SLA-таймеры по открытым тикетам."}
            </p>
            {s.sla_active_count ? (
              <Link href="/app/dashboard/sla" className="mt-2 inline-block text-[12px] text-[#185fa5] underline">
                Открыть SLA-таймеры
              </Link>
            ) : null}
          </DashboardPanel>

          <RecentSlaPreview />
        </aside>
      </div>
    </DashboardShell>
  );
}

function RecentSlaPreview() {
  const [deadline, setDeadline] = useState<string | undefined>();
  useEffect(() => {
    void fetchClientTickets().then((items) => {
      const open = items.find((t) => t.sla_reaction_deadline && !["resolved", "closed"].includes(t.status));
      setDeadline(open?.sla_reaction_deadline);
    });
  }, []);
  if (!deadline) return null;
  return (
    <DashboardPanel title="Ближайший дедлайн">
      <SlaTimer deadline={deadline} />
    </DashboardPanel>
  );
}

function SummaryCard({ label, value, note, tone, progress }: { label: string; value: string; note: string; tone: "blue" | "green" | "amber" | "purple"; progress: number }) {
  const colors = { blue: "#185fa5", green: "#3b6d11", amber: "#854f0b", purple: "#534ab7" };
  return (
    <div className="relative overflow-hidden rounded-lg border border-[#dedbd3] bg-white p-4">
      <div className="text-[10px] font-medium uppercase tracking-wide text-[#9a948c]">{label}</div>
      <div className="mt-1 text-2xl font-medium tracking-tight text-[#18212f]">{value}</div>
      <p className="mt-1 text-[12px] text-[#8a857d]">{note}</p>
      <div className="mt-2 h-0.5 overflow-hidden rounded-sm bg-[#ebe9e4]">
        <div className="h-full rounded-sm" style={{ width: `${progress}%`, backgroundColor: colors[tone] }} />
      </div>
    </div>
  );
}

function ActionLink({ children, href, primary = false }: { children: React.ReactNode; href: string; primary?: boolean }) {
  return (
    <Link href={href} className={primary ? "rounded-full bg-[#18212f] px-4 py-2 text-[12px] font-medium text-white hover:opacity-90" : "rounded-full border border-[#d7d2ca] px-4 py-2 text-[12px] font-medium text-[#18212f] hover:bg-[#ebe9e4]"}>
      {children}
    </Link>
  );
}

function OnboardingStep({ number, title, children, done = false }: { number: string; title: string; children: React.ReactNode; done?: boolean }) {
  return (
    <li className="flex gap-3">
      <div className={done ? "flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#3b6d11] text-white" : "flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-[#d7d2ca] bg-[#ebe9e4] text-[11px] font-medium text-[#18212f]"}>
        {done ? "✓" : number}
      </div>
      <div>
        <div className="text-[13px] font-medium text-[#18212f]">{title}</div>
        <p className="mt-0.5 text-[12px] text-[#5f6b7a]">{children}</p>
      </div>
    </li>
  );
}
