import { AdminShell } from "@/components/AdminShell";
import { AdminUsers } from "@/components/AdminUsers";

export default function AdminUsersPage() {
  return (
    <AdminShell breadcrumb="Пользователи">
      <div className="mx-auto max-w-[100%]">
        <AdminUsers />
      </div>
    </AdminShell>
  );
}
