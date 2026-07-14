import { VendorShell } from "@/components/VendorShell";
import { VendorPageHeader } from "@/components/cabinet/VendorPageHeader";
import { DashboardEmpty } from "@/components/dashboard/Ui";

export default function VendorMembersPage() {
  return (
    <VendorShell activePath="/app/vendor/members" pageTitle="Сотрудники">
      <VendorPageHeader
        title="Сотрудники"
        subtitle="Команда организации с доступом к очереди эскалаций и настройкам кабинета."
      />
      <DashboardEmpty title="Управление командой — в разработке">
        Пока используется учётная запись владельца организации. Приглашения коллег и роли инженера поддержки появятся в следующей итерации.
      </DashboardEmpty>
    </VendorShell>
  );
}
