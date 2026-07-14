import type { ReactNode } from "react";
import { CabinetGuard } from "@/components/CabinetGuard";

export default function VendorLayout({ children }: { children: ReactNode }) {
  return <CabinetGuard expected="vendor">{children}</CabinetGuard>;
}
