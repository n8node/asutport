import { AdminOrgs } from "@/components/AdminOrgs";
import { AdminShell } from "@/components/AdminShell";

export default function AdminVendorsPage() {
  return (
    <AdminShell breadcrumb="Поставщики">
      <AdminOrgs kind="vendor" />
    </AdminShell>
  );
}
