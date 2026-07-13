import { AdminOrgs } from "@/components/AdminOrgs";
import { AdminShell } from "@/components/AdminShell";

export default function AdminManufacturersPage() {
  return (
    <AdminShell breadcrumb="Производители">
      <AdminOrgs kind="manufacturer" />
    </AdminShell>
  );
}
