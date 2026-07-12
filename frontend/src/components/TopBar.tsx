import type { LampState } from "./Lamp";

const labels: Record<LampState, string> = {
  g: "Система в норме",
  a: "Внимание",
  r: "Авария",
  off: "Отключено",
};

type TopBarProps = {
  title: string;
  lampState?: LampState;
  subtitle?: string;
};

export function TopBar({ title, lampState = "g", subtitle }: TopBarProps) {
  return (
    <header className="flex items-center justify-between border-b border-line bg-panel px-6 py-4">
      <div>
        <p className="font-logo text-[10px] font-medium uppercase tracking-[0.18em] text-dim">
          ASUTPORT
        </p>
        <h1 className="mt-1 text-lg font-semibold text-text">{title}</h1>
        {subtitle ? <p className="mt-1 text-sm text-mut">{subtitle}</p> : null}
      </div>
      <div className="flex items-center gap-3">
        <span className="font-mono text-xs text-dim">{labels[lampState]}</span>
        <span className={`lamp lamp-${lampState}`} aria-hidden="true" />
      </div>
    </header>
  );
}
