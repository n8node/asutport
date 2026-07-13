import { HmiShell } from "@/components/HmiShell";
import { AdminOrgRequests } from "@/components/AdminOrgRequests";

export default function AdminPage() {
  return (
    <HmiShell
      title="Суперадминка"
      subtitle="Операционная панель платформы"
      lampState="g"
    >
      <div className="grid gap-6">
        <AdminOrgRequests />
        <div className="hmi-card p-6">
        <p className="text-sm text-mut">
          Метрики здоровья платформы, онбординг вендоров и биллинг — в фазах 3 и 8.
        </p>
      </div>
      </div>
    </HmiShell>
  );
}
