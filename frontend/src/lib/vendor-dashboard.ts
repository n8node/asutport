import { authFetch } from "@/lib/auth-session";

export type VendorTicket = {
  id: string;
  type: string;
  priority: string;
  status: string;
  subject: string;
  installation_id?: string;
  client_org_id?: string;
  client_org_name?: string;
  ball_owner_org_id?: string;
  ball_owner_org_name?: string;
  assigned_target_org_id?: string;
  assigned_target_org_name?: string;
  sla_reaction_deadline?: string;
  created_at: string;
  updated_at: string;
};

export type VendorDashboardSummary = {
  open_escalations_count: number;
};

type ApiList<T> = { data?: T[]; meta?: { total?: number }; error?: { message?: string } };
type ApiOne<T> = { data?: T; error?: { message?: string } };

export async function fetchVendorDashboard(): Promise<VendorDashboardSummary | null> {
  const response = await authFetch("/api/v1/vendor/dashboard");
  const body = (await response.json()) as ApiOne<VendorDashboardSummary>;
  if (!response.ok) return null;
  return body.data ?? null;
}

export async function fetchVendorTickets(): Promise<VendorTicket[]> {
  const response = await authFetch("/api/v1/vendor/tickets");
  const body = (await response.json()) as ApiList<VendorTicket>;
  if (!response.ok) return [];
  return body.data ?? [];
}

export async function fetchVendorTicket(ticketID: string): Promise<VendorTicket | null> {
  const response = await authFetch(`/api/v1/tickets/${ticketID}`);
  const body = (await response.json()) as ApiOne<VendorTicket>;
  if (!response.ok) return null;
  return body.data ?? null;
}

export async function resolveVendorTicket(ticketID: string, note?: string): Promise<boolean> {
  const response = await authFetch(`/api/v1/tickets/${ticketID}/resolve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ note: note ?? "" }),
  });
  return response.ok;
}

export const VENDOR_TICKET_TYPE_LABELS: Record<string, string> = {
  typical: "Типовой",
  defect: "Дефект",
  warranty: "Гарантия",
  application: "Прикладной",
  cross_vendor: "Стыковой",
};

export const VENDOR_PRIORITY_LABELS: Record<string, string> = {
  emergency: "Авария",
  degraded: "Деградация",
  question: "Вопрос",
};

export function vendorStatusLabel(status?: string): string {
  switch (status) {
    case "waiting_client":
      return "Ожидает клиента";
    case "waiting_platform":
      return "Ожидает платформу";
    case "waiting_vendor":
      return "Мяч у вас";
    case "resolved":
      return "Решён";
    case "closed":
      return "Закрыт";
    default:
      return status || "—";
  }
}
