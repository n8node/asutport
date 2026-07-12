export type LampState = "g" | "a" | "r" | "off";

type LampProps = {
  state: LampState;
  label?: string;
  className?: string;
};

const stateLabels: Record<LampState, string> = {
  g: "Норма",
  a: "Внимание",
  r: "Авария",
  off: "Выкл.",
};

export function Lamp({ state, label, className = "" }: LampProps) {
  const text = label ?? stateLabels[state];
  return (
    <span className={`inline-flex items-center gap-2 ${className}`}>
      <span className={`lamp lamp-${state}`} role="img" aria-label={text} />
      <span className="font-mono text-xs text-mut">{text}</span>
    </span>
  );
}
