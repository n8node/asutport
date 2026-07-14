import { authFetch } from "@/lib/auth-session";

export type ClientOrgProfile = {
  id?: string;
  name: string;
  legal_name?: string;
  inn?: string;
  type?: string;
  review_status?: string;
  is_personal?: boolean;
};

type ClientMeResponse = {
  user?: { email?: string; full_name?: string };
  org?: ClientOrgProfile;
};

const ORG_FALLBACK_LABEL = "Кабинет эксплуатации";

export function orgDisplayName(org?: Partial<ClientOrgProfile> | null): string {
  const name = org?.name?.trim();
  const legalName = org?.legal_name?.trim();
  const inn = org?.inn?.trim();
  if (name && name !== inn) return name;
  if (legalName && legalName !== inn) return legalName;
  if (name) return name;
  if (legalName) return legalName;
  return ORG_FALLBACK_LABEL;
}

async function fetchClientMe(): Promise<ClientMeResponse | null> {
  const response = await authFetch("/api/v1/auth/me");
  const body = (await response.json()) as { data?: ClientMeResponse };
  if (!response.ok || !body.data) return null;
  return body.data;
}

export async function fetchClientOrgProfile(): Promise<ClientOrgProfile | null> {
  const me = await fetchClientMe();
  return me?.org ?? null;
}

export async function fetchClientMeProfile(): Promise<ClientMeResponse | null> {
  return fetchClientMe();
}

export type DashboardSummary = {
  installations_count: number;
  open_tickets_count: number;
  sla_active_count: number;
  coverage_percent: number;
  profile_complete: boolean;
  products_count: number;
  supply_records_count: number;
};

export type Installation = {
  id: string;
  name: string;
  site_address: string;
  criticality: "continuous" | "batch" | "auxiliary";
  snapshot_allowed: boolean;
  emergency_contact_name: string;
  emergency_contact_phone: string;
  environment: Record<string, string>;
};

export type InstallationProduct = {
  id: string;
  installation_id: string;
  manufacturer_name: string;
  product_name: string;
  kind: string;
  version: string;
  notes: string;
};

export type SupplyRecord = {
  id: string;
  installation_product_id: string;
  product_name?: string;
  serial_or_license: string;
  supplier_name: string;
  integrator_name: string;
  purchase_date?: string;
  warranty_until?: string;
  contract_ref: string;
  verify_status: string;
};

export type ClientTicket = {
  id: string;
  type: string;
  priority: string;
  status: string;
  subject: string;
  installation_id?: string;
  ball_owner_org_id?: string;
  ball_owner_org_name?: string;
  assigned_target_org_id?: string;
  assigned_target_org_name?: string;
  sla_reaction_deadline?: string;
  created_at: string;
  updated_at: string;
};

export function ballOwnerLabel(ticket: Pick<ClientTicket, "ball_owner_org_id" | "ball_owner_org_name">, clientOrgID?: string): string {
  if (!ticket.ball_owner_org_id) {
    return "Платформа ASUTPORT";
  }
  if (clientOrgID && ticket.ball_owner_org_id === clientOrgID) {
    return "Ваша организация";
  }
  if (ticket.ball_owner_org_name) {
    return ticket.ball_owner_org_name;
  }
  return "Контрагент";
}

type ApiList<T> = { data?: T[]; meta?: { total?: number }; error?: { message?: string } };
type ApiOne<T> = { data?: T; error?: { message?: string } };

export async function fetchDashboardSummary(): Promise<DashboardSummary | null> {
  const response = await authFetch("/api/v1/client/dashboard");
  const body = (await response.json()) as ApiOne<DashboardSummary>;
  if (!response.ok) return null;
  return body.data ?? null;
}

export async function fetchInstallations(): Promise<Installation[]> {
  const response = await authFetch("/api/v1/client/installations");
  const body = (await response.json()) as ApiList<Installation>;
  if (!response.ok) return [];
  return body.data ?? [];
}

export async function saveInstallation(
  payload: Partial<Installation> & { id?: string },
): Promise<{ ok: boolean; error?: string; data?: Installation }> {
  const method = payload.id ? "PATCH" : "POST";
  const url = payload.id
    ? `/api/v1/client/installations/${payload.id}`
    : "/api/v1/client/installations";
  const response = await authFetch(url, {
    method,
    body: JSON.stringify(payload),
  });
  const body = (await response.json()) as ApiOne<Installation>;
  if (!response.ok) {
    return { ok: false, error: body.error?.message || "Не удалось сохранить" };
  }
  return { ok: true, data: body.data };
}

export async function fetchProducts(installationID: string): Promise<InstallationProduct[]> {
  const response = await authFetch(`/api/v1/client/installations/${installationID}/products`);
  const body = (await response.json()) as ApiList<InstallationProduct>;
  if (!response.ok) return [];
  return body.data ?? [];
}

export async function saveProduct(
  installationID: string,
  payload: Partial<InstallationProduct> & { id?: string },
): Promise<{ ok: boolean; error?: string }> {
  const method = payload.id ? "PATCH" : "POST";
  const url = payload.id
    ? `/api/v1/client/products/${payload.id}`
    : `/api/v1/client/installations/${installationID}/products`;
  const response = await authFetch(url, { method, body: JSON.stringify(payload) });
  const body = (await response.json()) as ApiOne<InstallationProduct>;
  if (!response.ok) {
    return { ok: false, error: body.error?.message || "Не удалось сохранить" };
  }
  return { ok: true };
}

export async function deleteProduct(productID: string): Promise<boolean> {
  const response = await authFetch(`/api/v1/client/products/${productID}`, { method: "DELETE" });
  return response.ok;
}

export async function fetchSupplyRecords(): Promise<SupplyRecord[]> {
  const response = await authFetch("/api/v1/client/supply-records");
  const body = (await response.json()) as ApiList<SupplyRecord>;
  if (!response.ok) return [];
  return body.data ?? [];
}

export async function createSupplyRecord(payload: {
  installation_product_id: string;
  serial_or_license: string;
  supplier_name?: string;
  integrator_name?: string;
  purchase_date?: string;
  warranty_until?: string;
  contract_ref?: string;
}): Promise<{ ok: boolean; error?: string }> {
  const response = await authFetch("/api/v1/client/supply-records", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  const body = (await response.json()) as ApiOne<SupplyRecord>;
  if (!response.ok) {
    return { ok: false, error: body.error?.message || "Не удалось сохранить" };
  }
  return { ok: true };
}

export async function deleteSupplyRecord(recordID: string): Promise<boolean> {
  const response = await authFetch(`/api/v1/client/supply-records/${recordID}`, { method: "DELETE" });
  return response.ok;
}

export async function fetchClientTickets(): Promise<ClientTicket[]> {
  const response = await authFetch("/api/v1/client/tickets");
  const body = (await response.json()) as ApiList<ClientTicket>;
  if (!response.ok) return [];
  return body.data ?? [];
}

export async function createClientTicket(payload: {
  subject: string;
  type?: string;
  priority?: string;
  installation_id?: string;
  text?: string;
}): Promise<{ ok: boolean; error?: string; ticket?: ClientTicket }> {
  const response = await authFetch("/api/v1/client/tickets", {
    method: "POST",
    body: JSON.stringify(payload),
  });
  const body = (await response.json()) as ApiOne<ClientTicket>;
  if (!response.ok) {
    return { ok: false, error: body.error?.message || "Не удалось создать тикет" };
  }
  return { ok: true, ticket: body.data };
}

export type BillingSummary = {
  plan: {
    id: string;
    name: string;
    slug: string;
    price_monthly_rub: number;
    ticket_quota?: number;
    overage_price_rub: number;
  };
  subscription: {
    plan_name: string;
    plan_slug: string;
    status: string;
    current_period_start: string;
    current_period_end: string;
    price_monthly_rub: number;
  };
  tickets_used: number;
  ticket_quota?: number;
  overage_price_rub: number;
  period_start: string;
  period_end: string;
  recent_payments: Array<{
    id: string;
    type: string;
    amount_rub: number;
    status: string;
    note: string;
    created_at: string;
  }>;
  public_plans: Array<{
    id: string;
    name: string;
    slug: string;
    price_monthly_rub: number;
    ticket_quota?: number;
    overage_price_rub: number;
  }>;
};

export type TicketQuotaCheck = {
  allowed: boolean;
  is_overage: boolean;
  warning?: string;
  tickets_used: number;
  ticket_quota?: number;
  overage_price_rub: number;
  plan_name: string;
  priority: string;
};

export async function fetchClientBilling(): Promise<BillingSummary | null> {
  const response = await authFetch("/api/v1/client/billing");
  const body = (await response.json()) as ApiOne<BillingSummary>;
  if (!response.ok || !body.data) return null;
  return body.data;
}

export async function fetchTicketQuotaCheck(priority: string): Promise<TicketQuotaCheck | null> {
  const response = await authFetch(`/api/v1/client/billing/quota-check?priority=${encodeURIComponent(priority)}`);
  const body = (await response.json()) as ApiOne<TicketQuotaCheck>;
  if (!response.ok || !body.data) return null;
  return body.data;
}

export const CRITICALITY_LABELS: Record<Installation["criticality"], string> = {
  continuous: "Непрерывное производство",
  batch: "Периодические пуски",
  auxiliary: "Вспомогательные системы",
};

export const PRODUCT_KIND_LABELS: Record<string, string> = {
  plc: "ПЛК",
  scada: "SCADA",
  hmi: "Панель оператора",
  drive: "Привод",
  sensor: "Датчик / КИП",
  network: "Сетевое оборудование",
  other: "Другое",
};

export const TICKET_TYPE_LABELS: Record<string, string> = {
  typical: "Типовой вопрос",
  defect: "Дефект продукта",
  warranty: "Гарантия / поставка",
  application: "Прикладной код",
  cross_vendor: "Стыковой случай",
};

export const TICKET_PRIORITY_LABELS: Record<string, string> = {
  emergency: "Авария",
  degraded: "Деградация",
  question: "Вопрос",
};

export const TICKET_STATUS_LABELS: Record<string, string> = {
  open: "Открыт",
  waiting_client: "Ожидает клиента",
  waiting_platform: "Ожидает платформу",
  waiting_vendor: "Ожидает вендора",
  resolved: "Решён",
  closed: "Закрыт",
};

export const VERIFY_STATUS_LABELS: Record<string, string> = {
  client_claim: "Данные клиента",
  manufacturer_verified: "Подтверждено производителем",
  partner_verified: "Подтверждено поставщиком",
};
