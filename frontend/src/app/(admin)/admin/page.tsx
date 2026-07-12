import { HmiShell } from "@/components/HmiShell";

export default function AdminPage() {
  return (
    <HmiShell
      title="Суперадминка"
      subtitle="Операционная панель платформы"
      lampState="g"
    >
      <div className="hmi-card p-6">
        <p className="text-sm text-mut">
          Метрики здоровья платформы, онбординг вендоров и биллинг — в фазах 3 и 8.
        </p>
      </div>
    </HmiShell>
  );
}
