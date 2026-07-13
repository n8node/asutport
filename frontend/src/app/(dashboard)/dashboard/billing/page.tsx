import { DashboardShell } from "@/components/DashboardShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";

export default function BillingPage() {
  return (
    <DashboardShell activePath="/app/dashboard/billing" pageTitle="Биллинг">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Биллинг и тариф</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Подписка организации, квота тикетов и счета на юрлицо. Аварийные обращения принимаются всегда.
        </p>
      </div>

      <DashboardEmpty title="Тариф не подключён">
        Сейчас действует бесплатный уровень: агент и база знаний. Платные тарифы с SLA и приоритетом в очереди подключаются после выставления счёта.
      </DashboardEmpty>

      <div className="mt-6 grid gap-4 sm:grid-cols-3">
        <TariffCard name="Бесплатный" price="0 ₽" note="Агент, база знаний, тикеты без гарантии SLA" active />
        <TariffCard name="Входной" price="~25 000 ₽/мес" note="Приоритет в очереди, расширенная квота" />
        <TariffCard name="Priority" price="~60 000 ₽/мес" note="SLA реакции 8/5, расширенная поддержка" />
      </div>
    </DashboardShell>
  );
}

function TariffCard({ name, price, note, active = false }: { name: string; price: string; note: string; active?: boolean }) {
  return (
    <DashboardPanel title={name}>
      <div className="text-lg font-medium text-[#18212f]">{price}</div>
      <p className="mt-2 text-[12px] leading-5 text-[#6f6a62]">{note}</p>
      {active ? (
        <span className="mt-3 inline-flex rounded-full bg-[#e6f1fb] px-2 py-0.5 text-[10px] font-semibold text-[#185fa5]">
          Текущий
        </span>
      ) : (
        <span className="mt-3 inline-flex text-[11px] text-[#8a857d]">Подключение — через менеджера</span>
      )}
    </DashboardPanel>
  );
}
