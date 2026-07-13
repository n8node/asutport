import { AdminOrgs } from "@/components/AdminOrgs";
import { AdminShell } from "@/components/AdminShell";

export default function AdminIntegratorsPage() {
  return (
    <AdminShell breadcrumb="Интеграторы">
      <AdminOrgs kind="integrator" />
    </AdminShell>
  );
}
