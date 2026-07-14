import { AdminShell } from "@/components/AdminShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";

export default function AdminBillingPage() {
  return (
    <AdminShell breadcrumb="Биллинг">
      <div className="mx-auto max-w-6xl">
        <div className="mb-6">
          <h1 className="text-2xl font-medium text-[#18212f]">Биллинг платформы</h1>
          <p className="mt-1 text-sm text-[#8a857d]">
            MRR по типам организаций, тарифы клиентов и производителей, ручные инвойсы на MVP.
          </p>
        </div>

        <div id="revenue" className="mb-6 grid gap-4 sm:grid-cols-3">
          <MetricCard label="MRR (все роли)" value="0 ₽" note="После первых оплат" />
          <MetricCard label="Клиенты" value="0 ₽" note="Подписки эксплуатации" />
          <MetricCard label="Производители" value="0 ₽" note="Подписки вендоров" />
        </div>

        <div id="plans" className="mb-6 grid gap-4 lg:grid-cols-2">
          <DashboardPanel title="Тарифы клиентов">
            <ul className="space-y-2 text-[13px] leading-5 text-[#5f6b7a]">
              <li>Free — агент, KB, тикеты без SLA</li>
              <li>Входной ~25 тыс ₽/мес — квота тикетов</li>
              <li>Priority ~60 тыс ₽/мес — SLA 8/5</li>
            </ul>
          </DashboardPanel>
          <DashboardPanel title="Тарифы производителей">
            <ul className="space-y-2 text-[13px] leading-5 text-[#5f6b7a]">
              <li>Базовый ~100–120 тыс ₽/мес</li>
              <li>Расширенный ~300 тыс ₽/мес</li>
            </ul>
          </DashboardPanel>
        </div>

        <div id="invoices">
          <DashboardEmpty title="Инвойсы и оплаты — ручной контур">
            Генерация PDF-счёта, фиксация оплаты суперадмином, квоты тикетов и overage — следующий шаг после первых договоров.
          </DashboardEmpty>
        </div>
      </div>
    </AdminShell>
  );
}

function MetricCard({ label, value, note }: { label: string; value: string; note: string }) {
  return (
    <div className="rounded-lg border border-[#dedbd3] bg-white p-4">
      <div className="text-[10px] font-medium uppercase tracking-wide text-[#9a948c]">{label}</div>
      <div className="mt-1 text-2xl font-medium text-[#18212f]">{value}</div>
      <p className="mt-1 text-[12px] text-[#8a857d]">{note}</p>
    </div>
  );
}
