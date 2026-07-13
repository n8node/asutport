import { AdminOrgs } from "@/components/AdminOrgs";
import { AdminShell } from "@/components/AdminShell";

export default function AdminClientsPage() {
  return (
    <AdminShell breadcrumb="Клиенты">
      <AdminOrgs kind="client_org" />
    </AdminShell>
  );
}
