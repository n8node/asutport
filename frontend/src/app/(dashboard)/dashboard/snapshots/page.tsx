import { DashboardShell } from "@/components/DashboardShell";
import { DashboardEmpty, DashboardPanel } from "@/components/dashboard/Ui";

export default function SnapshotsPage() {
  return (
    <DashboardShell activePath="/app/dashboard/snapshots" pageTitle="Слепки конфигурации">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Слепки конфигурации</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Диагностические снимки установки: логи, параметры, выгрузки проектов — только по вашему согласию.
        </p>
      </div>

      <DashboardEmpty title="Слепки пока не собирались">
        После подключения диагностической утилиты здесь появятся файлы, которые можно приложить к тикету одним действием.
        Разрешение на слепки включается в профиле установки.
      </DashboardEmpty>

      <div className="mt-6">
        <DashboardPanel title="Безопасность">
          <p className="text-[13px] leading-5 text-[#6f6a62]">
            Слепок не отправляется автоматически. Состав данных виден до отправки; для промышленных объектов это обязательное требование.
          </p>
        </DashboardPanel>
      </div>
    </DashboardShell>
  );
}
