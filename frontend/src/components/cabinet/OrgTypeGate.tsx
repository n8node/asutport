"use client";

import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import { fetchAccountProfile, vendorOrgLabel } from "@/lib/cabinet-routing";
import { DashboardEmpty } from "@/components/dashboard/Ui";

export function OrgTypeGate({
  allowed,
  children,
  title = "Раздел недоступен",
}: {
  allowed: string[];
  children: ReactNode;
  title?: string;
}) {
  const [orgType, setOrgType] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    void fetchAccountProfile()
      .then((profile) => setOrgType(profile?.org?.type ?? ""))
      .finally(() => setReady(true));
  }, []);

  if (!ready) {
    return <p className="text-sm text-[#6f6a62]">Загрузка…</p>;
  }

  if (!orgType || !allowed.includes(orgType)) {
    return (
      <DashboardEmpty title={title}>
        Этот раздел доступен для типа организации «{allowed.map(vendorOrgLabel).join("» или «")}».
        Ваша организация: {vendorOrgLabel(orgType || undefined)}.
      </DashboardEmpty>
    );
  }

  return children;
}
