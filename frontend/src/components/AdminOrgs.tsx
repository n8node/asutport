"use client";

import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { authFetch } from "@/lib/auth-session";

export type AdminOrgKind = "client_org" | "manufacturer" | "vendor" | "integrator";

type OrgOwner = {
  user_id: string;
  email: string;
  full_name: string;
  role: string;
};

type OrgMember = {
  user_id: string;
  email: string;
  full_name: string;
  role: string;
  is_active: boolean;
  created_at: string;
};

type OrgMetrics = {
  installations?: number | null;
  ticket_quota_used?: number | null;
  ticket_quota_limit?: number | null;
  open_tickets?: number | null;
  doc_sources?: number | null;
  products?: number | null;
  support_zone_loaded?: boolean;
  golden_set_ready?: boolean;
  entitlement_links?: number | null;
  fallback_events_30d?: number | null;
  application_tickets?: number | null;
  plan_name?: string;
  mrr_rub?: number | null;
};

type AdminOrg = {
  id: string;
  name: string;
  type: string;
  slug: string;
  is_active: boolean;
  legal_name: string;
  inn: string;
  website: string;
  contact_phone: string;
  review_comment: string;
  is_personal: boolean;
  review_status: string;
  reviewed_at: string | null;
  created_at: string;
  updated_at: string;
  member_count: number;
  owner?: OrgOwner;
  metrics: OrgMetrics;
  onboarding_stage: string;
  members?: OrgMember[];
};

type Filters = {
  search: string;
  review_status: string;
  is_active: string;
  is_personal: string;
};

const reviewLabels: Record<string, string> = {
  pending_email: "Ожидает email",
  pending_review: "На проверке",
  active: "Активна",
  rejected: "Отклонена",
  suspended: "Приостановлена",
};

const roleLabels: Record<string, string> = {
  owner: "Владелец",
  admin: "Админ",
  member: "Участник",
  support_engineer: "Инженер",
  superadmin: "Superadmin",
};

const stageLabels: Record<string, string> = {
  review: "Проверка заявки",
  onboarding: "Онбординг",
  golden: "Golden set",
  active: "Активен",
  pending_review: "На проверке",
  pending_email: "Ожидает email",
  rejected: "Отклонён",
  suspended: "Приостановлен",
};

type PageConfig = {
  kind: AdminOrgKind;
  title: string;
  subtitle: string;
};

const configs: Record<AdminOrgKind, PageConfig> = {
  client_org: {
    kind: "client_org",
    title: "Клиенты",
    subtitle: "Эксплуатация, установки, квоты тикетов и SLA",
  },
  manufacturer: {
    kind: "manufacturer",
    title: "Производители",
    subtitle: "Документация, golden set, зона поддержки, эскалации",
  },
  vendor: {
    kind: "vendor",
    title: "Поставщики",
    subtitle: "Entitlement, гарантия, fallback-отчёты (кабинет — после MVP)",
  },
  integrator: {
    kind: "integrator",
    title: "Интеграторы",
    subtitle: "Прикладная зона, проекты, услуги (кабинет — после MVP)",
  },
};

function phaseStub(label = "Фаза 3+") {
  return <span className="text-[#8a857d]">{label}</span>;
}

export function AdminOrgs({ kind }: { kind: AdminOrgKind }) {
  const config = configs[kind];
  const [items, setItems] = useState<AdminOrg[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("loading");
  const [message, setMessage] = useState("");
  const [filters, setFilters] = useState<Filters>({
    search: "",
    review_status: "",
    is_active: "",
    is_personal: "",
  });
  const [selected, setSelected] = useState<AdminOrg | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);

  const query = useMemo(() => {
    const params = new URLSearchParams({ type: kind, limit: "100" });
    if (filters.search) params.set("search", filters.search);
    if (filters.review_status) params.set("review_status", filters.review_status);
    if (filters.is_active) params.set("is_active", filters.is_active);
    if (filters.is_personal) params.set("is_personal", filters.is_personal);
    return params.toString();
  }, [kind, filters]);

  const load = useCallback(async () => {
    try {
      setStatus("loading");
      const response = await authFetch(`/api/v1/admin/orgs?${query}`);
      const body = (await response.json()) as {
        data?: AdminOrg[];
        meta?: { total?: number };
        error?: { message?: string };
      };
      if (!response.ok) {
        setStatus("error");
        setMessage(body.error?.message || "Не удалось загрузить организации");
        return;
      }
      setItems(body.data || []);
      setTotal(body.meta?.total || 0);
      setStatus("idle");
      setMessage("");
    } catch {
      setStatus("error");
      setMessage("API организаций временно недоступен");
    }
  }, [query]);

  useEffect(() => {
    void load();
  }, [load]);

  async function openDetail(org: AdminOrg) {
    setSelected(org);
    setDetailLoading(true);
    try {
      const response = await authFetch(`/api/v1/admin/orgs/${org.id}`);
      const body = (await response.json()) as { data?: AdminOrg };
      if (response.ok && body.data) setSelected(body.data);
    } finally {
      setDetailLoading(false);
    }
  }

  async function updateReview(orgID: string, reviewStatus: string) {
    setActionBusy(true);
    try {
      const response = await authFetch(`/api/v1/admin/orgs/${orgID}/review`, {
        method: "PATCH",
        body: JSON.stringify({ status: reviewStatus }),
      });
      const body = (await response.json()) as { data?: AdminOrg; error?: { message?: string } };
      if (!response.ok) {
        setMessage(body.error?.message || "Не удалось обновить статус");
        return;
      }
      await load();
      if (selected?.id === orgID && body.data) {
        setSelected(body.data);
      }
    } finally {
      setActionBusy(false);
    }
  }

  async function patchOrgActive(orgID: string, isActive: boolean) {
    setActionBusy(true);
    try {
      const response = await authFetch(`/api/v1/admin/orgs/${orgID}`, {
        method: "PATCH",
        body: JSON.stringify({ is_active: isActive }),
      });
      if (!response.ok) {
        setMessage("Не удалось обновить организацию");
        return;
      }
      await load();
      if (selected?.id === orgID) await openDetail({ ...selected, is_active: isActive });
    } finally {
      setActionBusy(false);
    }
  }

  return (
    <div className="relative">
      <section className="overflow-hidden rounded-[12px] border border-[#dedbd3] bg-white">
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[#e5e1da] px-4 py-3">
          <div>
            <p className="text-[10px] font-medium uppercase tracking-[0.07em] text-[#8a857d]">Организации</p>
            <h2 className="mt-1 text-[15px] font-semibold text-[#18212f]">{config.title}</h2>
            <p className="mt-1 text-[12px] text-[#8a857d]">
              {config.subtitle} · Всего: {total}
            </p>
          </div>
          <button
            type="button"
            className="rounded border border-[#d7d2ca] px-2.5 py-1 text-[11px] text-[#5f6b7a] hover:bg-[#ebe9e4]"
            onClick={() => void load()}
          >
            Обновить
          </button>
        </div>

        <div className="border-b border-[#ebe7df] px-4 py-3">
          <div className="flex flex-wrap gap-2">
            <input
              type="search"
              placeholder="Поиск: название, ИНН, slug"
              value={filters.search}
              onChange={(e) => setFilters((f) => ({ ...f, search: e.target.value }))}
              className="h-[30px] min-w-[220px] rounded-lg border border-[#d7d2ca] px-2.5 text-[12px]"
            />
            <FilterSelect
              label="Статус"
              value={filters.review_status}
              onChange={(v) => setFilters((f) => ({ ...f, review_status: v }))}
              options={[
                ["", "Все"],
                ["pending_review", "На проверке"],
                ["active", "Активна"],
                ["rejected", "Отклонена"],
                ["suspended", "Приостановлена"],
              ]}
            />
            <FilterSelect
              label="Включена"
              value={filters.is_active}
              onChange={(v) => setFilters((f) => ({ ...f, is_active: v }))}
              options={[
                ["", "Все"],
                ["true", "Да"],
                ["false", "Нет"],
              ]}
            />
            {kind === "client_org" ? (
              <FilterSelect
                label="Личный"
                value={filters.is_personal}
                onChange={(v) => setFilters((f) => ({ ...f, is_personal: v }))}
                options={[
                  ["", "Все"],
                  ["true", "Личный"],
                  ["false", "Юрлицо"],
                ]}
              />
            ) : null}
          </div>
        </div>

        {message ? <p className="px-4 pt-4 text-[13px] text-[#854f0b]">{message}</p> : null}
        {status === "loading" ? <p className="px-4 py-4 text-[13px] text-[#6f6a62]">Загружаем...</p> : null}

        <div className="overflow-x-auto">
          <table className="w-full min-w-[1400px] text-left text-[12px]">
            <thead>
              <tr className="border-b border-[#ebe7df] bg-[#faf9f7] text-[10px] uppercase tracking-[0.06em] text-[#8a857d]">
                <Th>Организация</Th>
                <Th>ID</Th>
                <Th>Юрлицо / ИНН</Th>
                <Th>Статус проверки</Th>
                <Th>Включена</Th>
                {kind === "client_org" ? <Th>Личный</Th> : null}
                <Th>Владелец</Th>
                <Th>Пользователей</Th>
                <Th>Контакты</Th>
                <Th>Зарегистрирована</Th>
                {extraColumns(kind)}
                <Th />
              </tr>
            </thead>
            <tbody>
              {items.map((org) => (
                <tr
                  key={org.id}
                  className="cursor-pointer border-b border-[#f0ede8] hover:bg-[#faf9f7]"
                  onClick={() => void openDetail(org)}
                >
                  <Td>
                    <div className="font-medium text-[#18212f]">{org.name}</div>
                    <div className="font-mono text-[11px] text-[#8a857d]">{org.slug}</div>
                  </Td>
                  <Td mono>{shortID(org.id)}</Td>
                  <Td>
                    <div>{org.legal_name || "—"}</div>
                    <div className="font-mono text-[11px] text-[#8a857d]">ИНН {org.inn || "—"}</div>
                  </Td>
                  <Td>
                    <StatusPill tone={reviewTone(org.review_status)}>
                      {reviewLabels[org.review_status] || org.review_status}
                    </StatusPill>
                  </Td>
                  <Td>
                    <StatusPill tone={org.is_active ? "green" : "red"}>
                      {org.is_active ? "Да" : "Нет"}
                    </StatusPill>
                  </Td>
                  {kind === "client_org" ? <Td>{org.is_personal ? "Да" : "Нет"}</Td> : null}
                  <Td>
                    {org.owner ? (
                      <>
                        <div>{org.owner.full_name || org.owner.email}</div>
                        <div className="text-[11px] text-[#8a857d]">{roleLabels[org.owner.role] || org.owner.role}</div>
                      </>
                    ) : (
                      "—"
                    )}
                  </Td>
                  <Td mono>{org.member_count}</Td>
                  <Td>
                    <div className="max-w-[140px] truncate">{org.contact_phone || "—"}</div>
                    <div className="max-w-[140px] truncate text-[11px] text-[#8a857d]">{org.website || ""}</div>
                  </Td>
                  <Td mono>{formatDate(org.created_at)}</Td>
                  {extraCells(kind, org)}
                  <Td>
                    <button
                      type="button"
                      className="rounded border border-[#d7d2ca] px-2 py-1 text-[11px] hover:bg-[#ebe9e4]"
                      onClick={(e) => {
                        e.stopPropagation();
                        void openDetail(org);
                      }}
                    >
                      Открыть
                    </button>
                  </Td>
                </tr>
              ))}
            </tbody>
          </table>
          {status === "idle" && items.length === 0 ? (
            <p className="px-4 py-6 text-[13px] text-[#6f6a62]">Организации не найдены.</p>
          ) : null}
        </div>
      </section>

      {selected ? (
        <>
          <button
            type="button"
            aria-label="Закрыть"
            className="fixed inset-0 z-30 bg-black/20"
            onClick={() => setSelected(null)}
          />
          <aside className="fixed right-0 top-0 z-40 flex h-screen w-full max-w-[520px] flex-col border-l border-[#dedbd3] bg-white shadow-xl">
            <div className="flex items-start justify-between gap-3 border-b border-[#e5e1da] px-5 py-4">
              <div className="min-w-0">
                <p className="text-[10px] font-medium uppercase tracking-[0.07em] text-[#8a857d]">{config.title}</p>
                <h3 className="mt-1 truncate text-[16px] font-semibold text-[#18212f]">{selected.name}</h3>
                <p className="font-mono text-[11px] text-[#8a857d]">{selected.slug}</p>
              </div>
              <button
                type="button"
                className="rounded border border-[#d7d2ca] px-2 py-1 text-[12px] hover:bg-[#ebe9e4]"
                onClick={() => setSelected(null)}
              >
                ✕
              </button>
            </div>

            <div className="flex-1 overflow-y-auto px-5 py-4">
              {detailLoading ? <p className="text-[13px] text-[#6f6a62]">Загружаем детали...</p> : null}

              <DrawerSection title="Реквизиты и статус">
                <div className="flex flex-wrap gap-2">
                  <StatusPill tone={reviewTone(selected.review_status)}>
                    {reviewLabels[selected.review_status] || selected.review_status}
                  </StatusPill>
                  <StatusPill tone={selected.is_active ? "green" : "red"}>
                    {selected.is_active ? "В маршрутизации" : "Отключена"}
                  </StatusPill>
                  <StatusPill tone="neutral">
                    {stageLabels[selected.onboarding_stage] || selected.onboarding_stage}
                  </StatusPill>
                </div>
                <dl className="mt-3 grid grid-cols-2 gap-2 text-[12px]">
                  <Field label="Юрлицо" value={selected.legal_name || "—"} />
                  <Field label="ИНН" value={selected.inn || "—"} mono />
                  <Field label="Телефон" value={selected.contact_phone || "—"} mono />
                  <Field label="Сайт" value={selected.website || "—"} />
                  <Field label="Создана" value={formatDateTime(selected.created_at)} mono />
                  <Field
                    label="Проверена"
                    value={selected.reviewed_at ? formatDateTime(selected.reviewed_at) : "—"}
                    mono
                  />
                </dl>
                {selected.review_comment ? (
                  <p className="mt-3 text-[12px] leading-5 text-[#5f6b7a]">{selected.review_comment}</p>
                ) : null}
              </DrawerSection>

              {selected.owner ? (
                <DrawerSection title="Владелец / контакт">
                  <div className="rounded-lg border border-[#ebe7df] p-3 text-[12px]">
                    <div className="font-medium">{selected.owner.full_name || selected.owner.email}</div>
                    <div className="text-[#8a857d]">{selected.owner.email}</div>
                    <div className="mt-1 text-[11px]">{roleLabels[selected.owner.role] || selected.owner.role}</div>
                  </div>
                </DrawerSection>
              ) : null}

              <DrawerSection title="Пользователи организации">
                {!selected.members || selected.members.length === 0 ? (
                  <p className="text-[12px] text-[#6f6a62]">Нет пользователей.</p>
                ) : (
                  <div className="space-y-2">
                    {selected.members.map((m) => (
                      <div key={m.user_id} className="rounded-lg border border-[#ebe7df] px-3 py-2 text-[11px]">
                        <div className="flex items-center justify-between gap-2">
                          <span className="font-medium">{m.full_name || m.email}</span>
                          <StatusPill tone={m.is_active ? "green" : "red"}>
                            {m.is_active ? "Активен" : "Заблокирован"}
                          </StatusPill>
                        </div>
                        <div className="text-[#8a857d]">{m.email}</div>
                        <div className="mt-1">{roleLabels[m.role] || m.role}</div>
                      </div>
                    ))}
                  </div>
                )}
              </DrawerSection>

              {drawerMetrics(kind, selected)}
            </div>

            <div className="border-t border-[#e5e1da] px-5 py-4">
              <div className="flex flex-wrap gap-2">
                {selected.review_status === "pending_review" ? (
                  <>
                    <button
                      type="button"
                      disabled={actionBusy}
                      className="rounded bg-[#1d4ed8] px-3 py-1.5 text-[12px] font-medium text-white hover:bg-[#1e40af] disabled:opacity-50"
                      onClick={() => void updateReview(selected.id, "active")}
                    >
                      Активировать
                    </button>
                    <button
                      type="button"
                      disabled={actionBusy}
                      className="rounded border border-[#d7d2ca] px-3 py-1.5 text-[12px] hover:bg-[#ebe9e4] disabled:opacity-50"
                      onClick={() => void updateReview(selected.id, "rejected")}
                    >
                      Отклонить
                    </button>
                  </>
                ) : null}
                {selected.is_active ? (
                  <button
                    type="button"
                    disabled={actionBusy}
                    className="rounded border border-[#e5484d] px-3 py-1.5 text-[12px] text-[#e5484d] hover:bg-red-50 disabled:opacity-50"
                    onClick={() => void patchOrgActive(selected.id, false)}
                  >
                    Отключить
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={actionBusy}
                    className="rounded border border-[#d7d2ca] px-3 py-1.5 text-[12px] hover:bg-[#ebe9e4] disabled:opacity-50"
                    onClick={() => void patchOrgActive(selected.id, true)}
                  >
                    Включить
                  </button>
                )}
                {selected.review_status === "active" ? (
                  <button
                    type="button"
                    disabled={actionBusy}
                    className="rounded border border-[#d7d2ca] px-3 py-1.5 text-[12px] hover:bg-[#ebe9e4] disabled:opacity-50"
                    onClick={() => void updateReview(selected.id, "suspended")}
                  >
                    Приостановить
                  </button>
                ) : null}
              </div>
            </div>
          </aside>
        </>
      ) : null}
    </div>
  );
}

function extraColumns(kind: AdminOrgKind) {
  switch (kind) {
    case "client_org":
      return (
        <>
          <Th>Тариф</Th>
          <Th>Установок</Th>
          <Th>Тикеты / квота</Th>
        </>
      );
    case "manufacturer":
      return (
        <>
          <Th>Онбординг</Th>
          <Th>Документы</Th>
          <Th>Зона поддержки</Th>
          <Th>Golden set</Th>
        </>
      );
    case "vendor":
      return (
        <>
          <Th>Entitlement</Th>
          <Th>Fallback 30д</Th>
          <Th>Тикеты warranty</Th>
        </>
      );
    case "integrator":
      return (
        <>
          <Th>Проектов</Th>
          <Th>Application</Th>
          <Th>Услуги</Th>
        </>
      );
  }
}

function extraCells(kind: AdminOrgKind, org: AdminOrg) {
  switch (kind) {
    case "client_org":
      return (
        <>
          <Td>{org.metrics.plan_name || phaseStub("Фаза 3")}</Td>
          <Td>{org.metrics.installations ?? phaseStub()}</Td>
          <Td>{phaseStub("Фаза 6")}</Td>
        </>
      );
    case "manufacturer":
      return (
        <>
          <Td>
            <StatusPill tone="neutral">{stageLabels[org.onboarding_stage] || org.onboarding_stage}</StatusPill>
          </Td>
          <Td>{org.metrics.doc_sources ?? phaseStub("Фаза 4")}</Td>
          <Td>
            <StatusPill tone={org.metrics.support_zone_loaded ? "green" : "amber"}>
              {org.metrics.support_zone_loaded ? "YAML" : "Нет"}
            </StatusPill>
          </Td>
          <Td>
            <StatusPill tone={org.metrics.golden_set_ready ? "green" : "amber"}>
              {org.metrics.golden_set_ready ? "Готов" : "Нет"}
            </StatusPill>
          </Td>
        </>
      );
    case "vendor":
      return (
        <>
          <Td>{org.metrics.entitlement_links ?? phaseStub("Фаза 6")}</Td>
          <Td>{org.metrics.fallback_events_30d ?? phaseStub("Фаза 6")}</Td>
          <Td>{org.metrics.open_tickets ?? phaseStub()}</Td>
        </>
      );
    case "integrator":
      return (
        <>
          <Td>{org.metrics.entitlement_links ?? phaseStub("Фаза 3")}</Td>
          <Td>{org.metrics.application_tickets ?? phaseStub("Фаза 6")}</Td>
          <Td>{phaseStub("Фаза 3")}</Td>
        </>
      );
  }
}

function drawerMetrics(kind: AdminOrgKind, org: AdminOrg) {
  switch (kind) {
    case "client_org":
      return (
        <DrawerSection title="Операционка клиента">
          <MetricGrid
            items={[
              ["Тариф", org.metrics.plan_name || "— (фаза 3)"],
              ["MRR", org.metrics.mrr_rub != null ? `${org.metrics.mrr_rub} ₽` : "— (фаза 3)"],
              ["Установок", fmtMetric(org.metrics.installations, "фаза 3")],
              ["Тикетов в периоде", fmtMetric(org.metrics.ticket_quota_used, "фаза 6")],
              ["Квота", fmtMetric(org.metrics.ticket_quota_limit, "фаза 3")],
              ["Открытых тикетов", fmtMetric(org.metrics.open_tickets, "фаза 6")],
            ]}
          />
        </DrawerSection>
      );
    case "manufacturer":
      return (
        <>
          <DrawerSection title="Онбординг производителя">
            <ol className="space-y-2 text-[12px] text-[#5f6b7a]">
              <li>1. Проверка заявки и ИНН</li>
              <li>2. Загрузка документации → S3 (фаза 4)</li>
              <li>3. Пайплайн parse/embed</li>
              <li>4. Импорт зоны поддержки (YAML)</li>
              <li>5. Golden set + «экзамен»</li>
            </ol>
          </DrawerSection>
          <DrawerSection title="Метрики вендора">
            <MetricGrid
              items={[
                ["Документов", fmtMetric(org.metrics.doc_sources, "фаза 4")],
                ["Продуктов", fmtMetric(org.metrics.products, "фаза 4")],
                ["Зона поддержки", org.metrics.support_zone_loaded ? "Импортирована" : "Не импортирована"],
                ["Golden set", org.metrics.golden_set_ready ? "Готов" : "Не готов"],
                ["Эскалаций", fmtMetric(org.metrics.open_tickets, "фаза 6")],
                ["Тариф", org.metrics.plan_name || "— (фаза 3)"],
              ]}
            />
          </DrawerSection>
        </>
      );
    case "vendor":
      return (
        <DrawerSection title="Поставщик (growth)">
          <p className="mb-3 text-[12px] text-[#5f6b7a]">
            Кабинет /partner не в MVP. Раздел для онбординга, entitlement-связей и fallback-отчётов.
          </p>
          <MetricGrid
            items={[
              ["Клиентов по entitlement", fmtMetric(org.metrics.entitlement_links, "фаза 6")],
              ["Fallback за 30д", fmtMetric(org.metrics.fallback_events_30d, "фаза 6")],
              ["Тикетов warranty", fmtMetric(org.metrics.open_tickets, "фаза 6")],
              ["Тариф", org.metrics.plan_name || "— (фаза 3)"],
            ]}
          />
        </DrawerSection>
      );
    case "integrator":
      return (
        <DrawerSection title="Интегратор (growth)">
          <p className="mb-3 text-[12px] text-[#5f6b7a]">
            Кабинет /integrator не в MVP. Реестр проектов, application-тикеты и услуги «паспорт проекта».
          </p>
          <MetricGrid
            items={[
              ["Проектов на платформе", fmtMetric(org.metrics.entitlement_links, "фаза 3")],
              ["Тикетов application", fmtMetric(org.metrics.application_tickets, "фаза 6")],
              ["Услуг (service_orders)", "— (фаза 3)"],
            ]}
          />
        </DrawerSection>
      );
  }
}

function MetricGrid({ items }: { items: [string, string][] }) {
  return (
    <dl className="grid grid-cols-2 gap-2 text-[12px]">
      {items.map(([k, v]) => (
        <div key={k}>
          <dt className="text-[#8a857d]">{k}</dt>
          <dd className="text-[#18212f]">{v}</dd>
        </div>
      ))}
    </dl>
  );
}

function fmtMetric(v: number | null | undefined, phase: string) {
  if (v != null) return String(v);
  return `— (${phase})`;
}

function FilterSelect({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: [string, string][];
}) {
  return (
    <label className="flex items-center gap-1.5 text-[11px] text-[#5f6b7a]">
      <span>{label}</span>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="h-[30px] rounded-lg border border-[#d7d2ca] bg-white px-2 text-[12px]"
      >
        {options.map(([v, t]) => (
          <option key={v || "all"} value={v}>
            {t}
          </option>
        ))}
      </select>
    </label>
  );
}

function Th({ children }: { children?: ReactNode }) {
  return <th className="px-3 py-2 font-medium">{children}</th>;
}

function Td({ children, mono }: { children: ReactNode; mono?: boolean }) {
  return <td className={`px-3 py-2.5 align-top ${mono ? "font-mono text-[11px]" : ""}`}>{children}</td>;
}

function DrawerSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="mb-5">
      <h4 className="mb-2 text-[11px] font-medium uppercase tracking-[0.07em] text-[#8a857d]">{title}</h4>
      {children}
    </section>
  );
}

function Field({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-[#8a857d]">{label}</dt>
      <dd className={mono ? "font-mono" : ""}>{value}</dd>
    </div>
  );
}

function StatusPill({ children, tone }: { children: ReactNode; tone: "green" | "amber" | "red" | "neutral" }) {
  const tones = {
    green: "border-[#4cc38a]/40 bg-[#4cc38a]/10 text-[#3b6d11]",
    amber: "border-[#f2a33c]/40 bg-[#f2a33c]/10 text-[#854f0b]",
    red: "border-[#e5484d]/40 bg-[#e5484d]/10 text-[#9f1239]",
    neutral: "border-[#d7d2ca] bg-[#faf9f7] text-[#5f6b7a]",
  };
  return (
    <span className={`inline-flex rounded-full border px-2 py-0.5 text-[10px] font-medium ${tones[tone]}`}>
      {children}
    </span>
  );
}

function shortID(id: string) {
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

function formatDate(iso: string) {
  try {
    return new Intl.DateTimeFormat("ru-RU", { day: "2-digit", month: "2-digit", year: "2-digit" }).format(
      new Date(iso),
    );
  } catch {
    return iso;
  }
}

function formatDateTime(iso: string) {
  try {
    return new Intl.DateTimeFormat("ru-RU", {
      day: "2-digit",
      month: "2-digit",
      year: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    }).format(new Date(iso));
  } catch {
    return iso;
  }
}

function reviewTone(status: string): "green" | "amber" | "red" | "neutral" {
  if (status === "active") return "green";
  if (status === "pending_review" || status === "pending_email") return "amber";
  if (status === "rejected" || status === "suspended") return "red";
  return "neutral";
}
