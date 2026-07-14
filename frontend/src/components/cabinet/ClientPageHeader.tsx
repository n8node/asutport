import type { ReactNode } from "react";

export function ClientPageHeader({ title, subtitle }: { title: string; subtitle: ReactNode }) {
  return (
    <div className="mb-6">
      <h1 className="text-2xl font-medium text-[#18212f]">{title}</h1>
      <p className="mt-1 text-sm text-[#8a857d]">{subtitle}</p>
    </div>
  );
}
