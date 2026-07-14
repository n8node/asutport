import type { ReactNode } from "react";
import { CabinetGuard } from "@/components/CabinetGuard";

export default function DashboardLayout({ children }: { children: ReactNode }) {
  return <CabinetGuard expected="client">{children}</CabinetGuard>;
}
