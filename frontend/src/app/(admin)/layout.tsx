import type { ReactNode } from "react";
import { CabinetGuard } from "@/components/CabinetGuard";

export default function AdminLayout({ children }: { children: ReactNode }) {
  return <CabinetGuard expected="admin">{children}</CabinetGuard>;
}
