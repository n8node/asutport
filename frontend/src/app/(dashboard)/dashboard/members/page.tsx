import { DashboardShell } from "@/components/DashboardShell";
import { DashboardEmpty } from "@/components/dashboard/Ui";

export default function MembersPage() {
  return (
    <DashboardShell activePath="/app/dashboard/members" pageTitle="Сотрудники">
      <div className="mb-6">
        <h1 className="text-2xl font-medium text-[#18212f]">Сотрудники</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Приглашение коллег в организацию эксплуатации с ролями владельца, администратора и инженера.
        </p>
      </div>
      <DashboardEmpty title="Управление командой — в разработке">
        Пока доступен один аккаунт на организацию. Приглашения появятся вместе с ролями и правами доступа к тикетам установки.
      </DashboardEmpty>
    </DashboardShell>
  );
}
