import { authFetch } from "@/lib/auth-session";

export type AdminBillingOverview = {
  mrr_total_rub: number;
  mrr_client_rub: number;
  mrr_manufacturer_rub: number;
  mrr_partner_rub: number;
  active_subscriptions: number;
};

export type AdminPlan = {
  id: string;
  org_type: string;
  name: string;
  slug: string;
  price_monthly_rub: number;
  ticket_quota?: number;
  overage_price_rub: number;
  is_public: boolean;
  is_archived: boolean;
  sort_order: number;
};

type ApiOne<T> = { data?: T; error?: { message?: string } };
type ApiList<T> = { data?: T[]; error?: { message?: string } };

export function formatRub(amount: number): string {
  return new Intl.NumberFormat("ru-RU").format(amount) + " ₽";
}

export async function fetchAdminBillingOverview(): Promise<AdminBillingOverview | null> {
  const response = await authFetch("/api/v1/admin/billing/overview");
  const body = (await response.json()) as ApiOne<AdminBillingOverview>;
  if (!response.ok || !body.data) return null;
  return body.data;
}

export async function fetchAdminPlans(orgType?: string): Promise<AdminPlan[]> {
  const q = orgType ? `?org_type=${encodeURIComponent(orgType)}` : "";
  const response = await authFetch(`/api/v1/admin/plans${q}`);
  const body = (await response.json()) as ApiList<AdminPlan>;
  if (!response.ok || !body.data) return [];
  return body.data;
}

export async function fetchVendorBilling(): Promise<import("@/lib/client-dashboard").BillingSummary | null> {
  const response = await authFetch("/api/v1/vendor/billing");
  const body = (await response.json()) as ApiOne<import("@/lib/client-dashboard").BillingSummary>;
  if (!response.ok || !body.data) return null;
  return body.data;
}
