import { AdminShell } from "@/components/AdminShell";
import { AdminOrgRequests } from "@/components/AdminOrgRequests";
import { AdminSettingsPanels } from "@/components/AdminSettingsPanels";
import type { ReactNode } from "react";

export default function AdminPage() {
  return (
    <AdminShell>
      <div className="mx-auto max-w-6xl">
        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 lg:grid-cols-4">
          <KpiCard
            tone="blue"
            label="MRR (RUB)"
            value="0 ₽"
            sub="Ручные инвойсы появятся в фазе биллинга"
            barWidth={8}
          />
          <KpiCard
            tone="green"
            label="Активные организации"
            value="—"
            sub="Клиенты, производители и заявки"
            barWidth={12}
          />
          <KpiCard
            tone="amber"
            label="Заявки на проверке"
            value="—"
            sub="Производители, поставщики, интеграторы"
            barWidth={25}
          />
          <KpiCard
            tone="purple"
            label="SLA breach"
            value="—"
            sub="Тикеты появятся в фазе 6"
            barWidth={5}
          />
        </div>

        <div className="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
          <div id="org-requests">
            <AdminOrgRequests />
          </div>

          <div className="space-y-3">
            <Panel title="Operations">
              <div className="flex flex-col gap-2 text-[13px]">
                <a href="/app/db/" className="text-[#185fa5] underline">
                  Открыть Adminer
                </a>
                <a href="#audit" className="text-[#185fa5] underline">
                  Admin audit log (скоро)
                </a>
                <a href="#health" className="text-[#185fa5] underline">
                  System health
                </a>
              </div>
            </Panel>

            <Panel title="Порядок онбординга">
              <ol className="space-y-2 text-[13px] leading-5 text-[#5f6b7a]">
                <li>1. Организация регистрируется и попадает в pending review.</li>
                <li>2. Суперадмин связывается с контактом и проверяет ИНН.</li>
                <li>3. После активации организация участвует в маршрутизации.</li>
              </ol>
            </Panel>
          </div>
        </div>

        <div className="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-2">
          <Panel title="Usage by contour (30d)">
            <p className="text-[13px] text-[#6f6a62]">Нет данных за последние 30 дней.</p>
          </Panel>
          <Panel title="LLM">
            <p className="text-[13px] text-[#6f6a62]">
              Контроль расходов ИИ будет подключён после пайплайна документации.
            </p>
          </Panel>
        </div>

        <div className="mt-6">
          <AdminSettingsPanels />
        </div>
      </div>
    </AdminShell>
  );
}

function KpiCard({
  label,
  value,
  sub,
  tone,
  barWidth,
}: {
  label: string;
  value: string;
  sub: string;
  tone: "blue" | "green" | "amber" | "purple";
  barWidth: number;
}) {
  const colors = {
    blue: "#185fa5",
    green: "#3b6d11",
    amber: "#854f0b",
    purple: "#534ab7",
  };
  return (
    <div className="relative overflow-hidden rounded-[12px] border border-[#dedbd3] bg-white px-4 py-3.5">
      <div className="absolute left-0 right-0 top-0 h-[3px]" style={{ backgroundColor: colors[tone] }} />
      <div className="text-[10px] font-medium uppercase tracking-[0.07em] text-[#8a857d]">{label}</div>
      <div className="mt-1.5 text-2xl font-medium tracking-tight text-[#18212f]">{value}</div>
      <div className="mt-1.5 text-[10px] text-[#8a857d]">{sub}</div>
      <div className="mt-2 h-1 rounded-sm bg-[#ebe9e4]">
        <div className="h-full rounded-sm" style={{ width: `${barWidth}%`, backgroundColor: colors[tone] }} />
      </div>
    </div>
  );
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="overflow-hidden rounded-[12px] border border-[#dedbd3] bg-white">
      <div className="border-b border-[#e5e1da] px-4 py-3">
        <h2 className="text-[12px] font-medium text-[#5f6b7a]">{title}</h2>
      </div>
      <div className="px-4 py-4">{children}</div>
    </section>
  );
}
