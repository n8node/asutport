import Link from "next/link";
import { DashboardShell } from "@/components/DashboardShell";
import type { ReactNode } from "react";

export default function DashboardPage() {
  return (
    <DashboardShell>
      <div className="mb-6">
        <h1 className="text-2xl font-medium tracking-tight text-[#18212f] sm:text-[26px]">
          Кабинет клиента
        </h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Единое окно поддержки: профиль установки, тикеты, агент и SLA.
        </p>
      </div>

      <section className="mb-8">
        <h2 className="mb-3.5 text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">
          Сводка
        </h2>
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
          <SummaryCard
            label="Установки"
            value="0"
            note="Добавьте первую производственную площадку"
            tone="blue"
            progress={8}
          />
          <SummaryCard
            label="Открытые тикеты"
            value="0"
            note="Новых обращений нет"
            tone="green"
            progress={4}
          />
          <SummaryCard
            label="SLA"
            value="—"
            note="Активируется после тарифа и тикетов"
            tone="amber"
            progress={0}
          />
          <SummaryCard
            label="Покрытие"
            value="0%"
            note="Заполните продукты и версии"
            tone="purple"
            progress={0}
          />
        </div>
      </section>

      <section className="mb-8">
        <h2 className="mb-3.5 text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">
          Быстрые действия
        </h2>
        <div className="flex flex-wrap gap-2">
          <ActionLink href="#agent" primary>Описать проблему агенту</ActionLink>
          <ActionLink href="#tickets">Создать тикет</ActionLink>
          <ActionLink href="#installation">Добавить установку</ActionLink>
          <ActionLink href="/app/kb">Открыть базу знаний</ActionLink>
        </div>
      </section>

      <div className="grid gap-6 lg:grid-cols-12">
        <section className="lg:col-span-7">
          <h2 className="mb-3.5 text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">
            Начните работу
          </h2>
          <div className="rounded-lg border border-[#dedbd3] bg-white p-5 sm:p-6">
            <ol className="space-y-5">
              <OnboardingStep done number="1" title="Аккаунт создан">
                Вы вошли в личный кабинет ASUTPORT.
              </OnboardingStep>
              <OnboardingStep number="2" title="Заполните профиль установки">
                Укажите площадку, продукты, версии, ОС/виртуализацию и критичность производства.
              </OnboardingStep>
              <OnboardingStep number="3" title="Добавьте продукты и entitlement">
                Серийники, лицензии, поставщики и интеграторы определяют будущий маршрут эскалаций.
              </OnboardingStep>
              <OnboardingStep number="4" title="Задайте первый вопрос агенту">
                Агент ответит только с цитатами из документации или предложит эскалацию.
              </OnboardingStep>
              <OnboardingStep number="5" title="Выберите тариф">
                SLA и квоты тикетов подключаются после биллинга.
              </OnboardingStep>
            </ol>
          </div>
        </section>

        <aside className="flex flex-col gap-4 lg:col-span-5">
          <Panel title="Статус платформы">
            <div className="flex items-center justify-between gap-4">
              <span className="text-[13px] text-[#5f6b7a]">API / Postgres</span>
              <span className="inline-flex items-center gap-1.5 text-[12px] font-medium text-[#3b6d11]">
                <span className="h-2 w-2 rounded-full bg-[#3b6d11]" />
                Норма
              </span>
            </div>
            <p className="mt-3 text-[12px] text-[#8a857d]">
              S3 на production может быть degraded до завершения настройки бакета.
            </p>
          </Panel>

          <Panel title="Последние события">
            <p className="text-[13px] text-[#6f6a62]">Пока нет тикетов и событий установки.</p>
          </Panel>

          <Panel title="SLA и мяч на стороне">
            <p className="text-[13px] leading-5 text-[#6f6a62]">
              Здесь появятся живые SLA-таймеры и текущий ответственный по каждому тикету.
            </p>
          </Panel>
        </aside>
      </div>
    </DashboardShell>
  );
}

function SummaryCard({
  label,
  value,
  note,
  tone,
  progress,
}: {
  label: string;
  value: string;
  note: string;
  tone: "blue" | "green" | "amber" | "purple";
  progress: number;
}) {
  const colors = {
    blue: "#185fa5",
    green: "#3b6d11",
    amber: "#854f0b",
    purple: "#534ab7",
  };
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

function ActionLink({
  children,
  href,
  primary = false,
}: {
  children: ReactNode;
  href: string;
  primary?: boolean;
}) {
  return (
    <Link
      href={href}
      className={
        primary
          ? "rounded-full bg-[#18212f] px-4 py-2 text-[12px] font-medium text-white hover:opacity-90"
          : "rounded-full border border-[#d7d2ca] px-4 py-2 text-[12px] font-medium text-[#18212f] hover:bg-[#ebe9e4]"
      }
    >
      {children}
    </Link>
  );
}

function OnboardingStep({
  number,
  title,
  children,
  done = false,
}: {
  number: string;
  title: string;
  children: ReactNode;
  done?: boolean;
}) {
  return (
    <li className="flex gap-3">
      <div
        className={
          done
            ? "flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#3b6d11] text-white"
            : "flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-[#d7d2ca] bg-[#ebe9e4] text-[11px] font-medium text-[#18212f]"
        }
      >
        {done ? "✓" : number}
      </div>
      <div>
        <div className="text-[13px] font-medium text-[#18212f]">{title}</div>
        <p className="mt-0.5 text-[12px] text-[#5f6b7a]">{children}</p>
      </div>
    </li>
  );
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-lg border border-[#dedbd3] bg-white p-4">
      <h2 className="mb-3 text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">
        {title}
      </h2>
      {children}
    </section>
  );
}
