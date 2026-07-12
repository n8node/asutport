import type { ReactNode } from "react";
import { TopBar } from "@/components/TopBar";
import type { LampState } from "@/components/Lamp";

type HmiShellProps = {
  title: string;
  subtitle?: string;
  lampState?: LampState;
  children: ReactNode;
};

export function HmiShell({ title, subtitle, lampState = "g", children }: HmiShellProps) {
  return (
    <div className="min-h-screen bg-bg">
      <TopBar title={title} subtitle={subtitle} lampState={lampState} />
      <main className="mx-auto max-w-6xl px-6 py-8">{children}</main>
    </div>
  );
}
