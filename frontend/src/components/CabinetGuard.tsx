"use client";

import { useRouter } from "next/navigation";
import type { ReactNode } from "react";
import { useEffect, useState } from "react";
import {
  cabinetForProfile,
  fetchAccountProfile,
  homeRoute,
  type CabinetKind,
} from "@/lib/cabinet-routing";

export function CabinetGuard({
  expected,
  children,
}: {
  expected: CabinetKind;
  children: ReactNode;
}) {
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const token = sessionStorage.getItem("asutport_access_token");
    if (!token) {
      router.replace("/app/login");
      return;
    }

    void fetchAccountProfile()
      .then((profile) => {
        if (!profile?.org) {
          router.replace("/app/login");
          return;
        }
        const cabinet = cabinetForProfile(profile);
        if (cabinet !== expected) {
          router.replace(homeRoute(profile));
          return;
        }
        setReady(true);
      })
      .catch(() => router.replace("/app/login"));
  }, [expected, router]);

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#f3f2ef] text-sm text-[#6f6a62]">
        Загрузка кабинета…
      </div>
    );
  }

  return children;
}
