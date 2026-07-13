"use client";

import { useEffect, useState } from "react";

type SlaTimerProps = {
  deadline?: string;
  className?: string;
};

export function SlaTimer({ deadline, className = "" }: SlaTimerProps) {
  const [label, setLabel] = useState("—");
  const [tone, setTone] = useState<"ok" | "warn" | "over">("ok");

  useEffect(() => {
    if (!deadline) {
      setLabel("Без SLA");
      setTone("ok");
      return;
    }
    const target = deadline;
    function tick() {
      const end = new Date(target).getTime();
      const diff = end - Date.now();
      if (diff <= 0) {
        setLabel("Просрочено");
        setTone("over");
        return;
      }
      const totalSec = Math.floor(diff / 1000);
      const h = Math.floor(totalSec / 3600);
      const m = Math.floor((totalSec % 3600) / 60);
      const s = totalSec % 60;
      setLabel(`${String(h).padStart(2, "0")}:${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`);
      setTone(diff < 30 * 60 * 1000 ? "warn" : "ok");
    }
    tick();
    const id = window.setInterval(tick, 1000);
    return () => window.clearInterval(id);
  }, [deadline]);

  const colors = {
    ok: "text-[#3b6d11]",
    warn: "text-[#854f0b]",
    over: "text-[#b42318]",
  };

  return (
    <span className={`font-mono text-[13px] font-medium tabular-nums ${colors[tone]} ${className}`}>
      {label}
    </span>
  );
}
