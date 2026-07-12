import { HmiShell } from "@/components/HmiShell";
import { Lamp } from "@/components/Lamp";

export default function DashboardPage() {
  return (
    <HmiShell
      title="Кабинет клиента"
      subtitle="Фаза 1 — скелет интерфейса"
      lampState="g"
    >
      <div className="grid gap-6 md:grid-cols-2">
        <section className="hmi-card p-6">
          <p className="font-logo text-[10px] uppercase tracking-[0.18em] text-dim">
            Статус платформы
          </p>
          <div className="mt-4 flex items-center gap-4">
            <Lamp state="g" />
            <span className="font-mono text-sm text-mut">v0.1.0 · dev</span>
          </div>
        </section>
        <section className="hmi-card p-6">
          <p className="font-logo text-[10px] uppercase tracking-[0.18em] text-dim">
            Следующий шаг
          </p>
          <p className="mt-4 text-sm text-mut">
            Профиль установки, чат с ИИ-агентом и тикеты появятся в следующих фазах.
          </p>
          <button type="button" className="hmi-btn-primary mt-6">
            Открыть чат (скоро)
          </button>
        </section>
      </div>
    </HmiShell>
  );
}
