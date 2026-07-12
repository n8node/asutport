import { HmiShell } from "@/components/HmiShell";

export default function VendorPage() {
  return (
    <HmiShell
      title="Кабинет производителя"
      subtitle="Очередь эскалаций — заглушка"
      lampState="a"
    >
      <div className="hmi-card p-6">
        <p className="text-sm text-mut">
          Здесь будет очередь эскалаций, зона поддержки и аналитика вендора.
        </p>
      </div>
    </HmiShell>
  );
}
