import { authFetch } from "@/lib/auth-session";

export type AccountOrg = {
  id?: string;
  name?: string;
  legal_name?: string;
  inn?: string;
  type?: string;
  role?: string;
  review_status?: string;
  is_personal?: boolean;
};

export type AccountProfile = {
  user?: { email?: string; full_name?: string };
  org?: AccountOrg;
};

export function isVendorOrgType(orgType?: string): boolean {
  return orgType === "manufacturer" || orgType === "vendor" || orgType === "integrator";
}

export function isClientOrgType(orgType?: string): boolean {
  return orgType === "client_org";
}

export function isSuperAdminRole(role?: string): boolean {
  return role === "superadmin";
}

export type CabinetKind = "client" | "vendor" | "admin";

export function cabinetForProfile(profile?: AccountProfile | null): CabinetKind {
  if (isSuperAdminRole(profile?.org?.role)) {
    return "admin";
  }
  if (isVendorOrgType(profile?.org?.type)) {
    return "vendor";
  }
  return "client";
}

export function homeRoute(profile?: Pick<AccountProfile, "org"> | null): string {
  const org = profile?.org;
  if (isSuperAdminRole(org?.role)) {
    return "/app/admin";
  }
  if (org?.review_status === "pending_review") {
    if (isVendorOrgType(org.type)) {
      return "/app/vendor/onboarding";
    }
    return "/app/dashboard/onboarding";
  }
  if (isVendorOrgType(org?.type)) {
    return "/app/vendor";
  }
  return "/app/dashboard";
}

export function homeRouteFromLogin(role?: string, orgType?: string, reviewStatus?: string): string {
  return homeRoute({
    org: { role, type: orgType, review_status: reviewStatus },
  });
}

export async function fetchAccountProfile(): Promise<AccountProfile | null> {
  const response = await authFetch("/api/v1/auth/me");
  const body = (await response.json()) as { data?: AccountProfile };
  if (!response.ok) return null;
  return body.data ?? null;
}

export function vendorOrgLabel(orgType?: string): string {
  switch (orgType) {
    case "manufacturer":
      return "Производитель";
    case "vendor":
      return "Поставщик";
    case "integrator":
      return "Интегратор";
    default:
      return "Партнёр";
  }
}

export function clientOrgLabel(isPersonal?: boolean): string {
  return isPersonal ? "Личный кабинет" : "Эксплуатация";
}
