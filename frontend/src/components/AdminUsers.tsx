"use client";

import type { ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";

import { authFetch } from "@/lib/auth-session";

type Membership = {
  org_id: string;
  org_name: string;
  org_type: string;
  org_slug: string;
  role: string;
  review_status: string;
  is_personal: boolean;
  org_is_active: boolean;
  inn: string;
  website: string;
  contact_phone: string;
  member_since: string;
};

type MessengerLink = {
  id: string;
  provider: "telegram" | "max";
  external_user_id: string;
  username: string;
  display_name: string;
  is_verified: boolean;
  notifications_enabled: boolean;
  linked_at: string | null;
  created_at: string;
};

type AdminUser = {
  id: string;
  email: string;
  full_name: string;
  is_active: boolean;
  access_level: "full" | "limited" | "none";
  created_at: string;
  updated_at: string;
  last_login_at: string | null;
  active_sessions: number;
  last_ip: string;
  last_user_agent: string;
  memberships: Membership[];
  messengers: MessengerLink[];
  sessions?: AdminSession[];
};

type AdminSession = {
  id: string;
  org_id: string;
  org_name: string;
  ip_address: string;
  user_agent: string;
  expires_at: string;
  revoked_at: string | null;
  created_at: string;
  is_active: boolean;
};

type Filters = {
  search: string;
  is_active: string;
  access: string;
  org_type: string;
  role: string;
  review_status: string;
  is_personal: string;
  has_active_sessions: string;
  last_login: string;
};

const defaultFilters: Filters = {
  search: "",
  is_active: "",
  access: "",
  org_type: "",
  role: "",
  review_status: "",
  is_personal: "",
  has_active_sessions: "",
  last_login: "",
};

const orgTypeLabels: Record<string, string> = {
  client_org: "Клиент",
  manufacturer: "Производитель",
  vendor: "Поставщик",
  integrator: "Интегратор",
  partner: "Партнёр",
};

const roleLabels: Record<string, string> = {
  owner: "Владелец",
  admin: "Админ",
  member: "Участник",
  support_engineer: "Инженер",
  superadmin: "Superadmin",
};

const reviewLabels: Record<string, string> = {
  pending_email: "Ожидает email",
  pending_review: "На проверке",
  active: "Активна",
  rejected: "Отклонена",
  suspended: "Приостановлена",
};

const accessLabels: Record<string, string> = {
  full: "Полный",
  limited: "Ограничен",
  none: "Нет входа",
};

export function AdminUsers() {
  const [items, setItems] = useState<AdminUser[]>([]);
  const [total, setTotal] = useState(0);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("loading");
  const [message, setMessage] = useState("");
  const [filters, setFilters] = useState<Filters>(defaultFilters);
  const [selected, setSelected] = useState<AdminUser | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [actionBusy, setActionBusy] = useState(false);

  const query = useMemo(() => {
    const params = new URLSearchParams();
    Object.entries(filters).forEach(([key, value]) => {
      if (value) {
        params.set(key, value);
      }
    });
    params.set("limit", "100");
    return params.toString();
  }, [filters]);

  const load = useCallback(async () => {
    try {
      setStatus("loading");
      const response = await authFetch(`/api/v1/admin/users?${query}`);
      const body = (await response.json()) as {
        data?: AdminUser[];
        meta?: { total?: number };
        error?: { message?: string };
      };
      if (!response.ok) {
        setStatus("error");
        setMessage(body.error?.message || "Не удалось загрузить пользователей");
        return;
      }
      setItems(body.data || []);
      setTotal(body.meta?.total || 0);
      setStatus("idle");
      setMessage("");
    } catch {
      setStatus("error");
      setMessage("API пользователей временно недоступен");
    }
  }, [query]);

  useEffect(() => {
    void load();
  }, [load]);

  async function openDetail(user: AdminUser) {
    setSelected(user);
    setDetailLoading(true);
    try {
      const response = await authFetch(`/api/v1/admin/users/${user.id}`);
      const body = (await response.json()) as { data?: AdminUser; error?: { message?: string } };
      if (response.ok && body.data) {
        setSelected(body.data);
      }
    } finally {
      setDetailLoading(false);
    }
  }

  async function patchActive(userID: string, isActive: boolean) {
    setActionBusy(true);
    try {
      const response = await authFetch(`/api/v1/admin/users/${userID}`, {
        method: "PATCH",
        body: JSON.stringify({ is_active: isActive }),
      });
      if (!response.ok) {
        setMessage("Не удалось обновить статус пользователя");
        return;
      }
      await load();
      if (selected?.id === userID) {
        await openDetail({ ...selected, is_active: isActive });
      }
    } finally {
      setActionBusy(false);
    }
  }

  async function revokeSessions(userID: string) {
    setActionBusy(true);
    try {
      const response = await authFetch(`/api/v1/admin/users/${userID}/revoke-sessions`, {
        method: "POST",
        body: "{}",
      });
      if (!response.ok) {
        setMessage("Не удалось отозвать сессии");
        return;
      }
      await load();
      if (selected?.id === userID) {
        const current = items.find((u) => u.id === userID) || selected;
        await openDetail(current);
      }
    } finally {
      setActionBusy(false);
    }
  }

  async function deleteUser(user: AdminUser) {
    if (isSuperadminUser(user)) {
      setMessage("Нельзя удалить учётную запись superadmin");
      return;
    }
    const confirmed = window.confirm(
      `Удалить пользователя ${user.email}?\n\nEmail освободится для повторной регистрации. Организации с одним участником будут удалены. Действие необратимо.`,
    );
    if (!confirmed) {
      return;
    }
    setActionBusy(true);
    try {
      const response = await authFetch(`/api/v1/admin/users/${user.id}`, { method: "DELETE" });
      const body = (await response.json()) as { error?: { message?: string } };
      if (!response.ok) {
        setMessage(body.error?.message || "Не удалось удалить пользователя");
        return;
      }
      setSelected(null);
      setMessage(`Пользователь ${user.email} удалён`);
      await load();
    } finally {
      setActionBusy(false);
    }
  }

  return (
    <div className="relative">
      <section className="overflow-hidden rounded-[12px] border border-[#dedbd3] bg-white">
        <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[#e5e1da] px-4 py-3">
          <div>
            <p className="text-[10px] font-medium uppercase tracking-[0.07em] text-[#8a857d]">Платформа</p>
            <h2 className="mt-1 text-[15px] font-semibold text-[#18212f]">Пользователи</h2>
            <p className="mt-1 text-[12px] text-[#8a857d]">Всего: {total}</p>
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
              placeholder="Поиск email или ФИО"
              value={filters.search}
              onChange={(e) => setFilters((f) => ({ ...f, search: e.target.value }))}
              className="h-[30px] min-w-[200px] rounded-lg border border-[#d7d2ca] px-2.5 text-[12px]"
            />
            <FilterSelect
              label="Аккаунт"
              value={filters.is_active}
              onChange={(v) => setFilters((f) => ({ ...f, is_active: v }))}
              options={[
                ["", "Все"],
                ["true", "Активен"],
                ["false", "Заблокирован"],
              ]}
            />
            <FilterSelect
              label="Доступ"
              value={filters.access}
              onChange={(v) => setFilters((f) => ({ ...f, access: v }))}
              options={[
                ["", "Все"],
                ["full", "Полный"],
                ["limited", "Ограничен"],
                ["none", "Нет входа"],
              ]}
            />
            <FilterSelect
              label="Тип орг."
              value={filters.org_type}
              onChange={(v) => setFilters((f) => ({ ...f, org_type: v }))}
              options={[
                ["", "Все"],
                ["client_org", "Клиент"],
                ["manufacturer", "Производитель"],
                ["vendor", "Поставщик"],
                ["integrator", "Интегратор"],
              ]}
            />
            <FilterSelect
              label="Роль"
              value={filters.role}
              onChange={(v) => setFilters((f) => ({ ...f, role: v }))}
              options={[
                ["", "Все"],
                ["superadmin", "Superadmin"],
                ["owner", "Владелец"],
                ["admin", "Админ"],
                ["member", "Участник"],
                ["support_engineer", "Инженер"],
              ]}
            />
            <FilterSelect
              label="Статус орг."
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
              label="Вход"
              value={filters.last_login}
              onChange={(v) => setFilters((f) => ({ ...f, last_login: v }))}
              options={[
                ["", "Все"],
                ["today", "Сегодня"],
                ["7d", "7 дней"],
                ["30d", "30 дней"],
                ["never", "Никогда"],
              ]}
            />
          </div>
        </div>

        {message ? <p className="px-4 pt-4 text-[13px] text-[#854f0b]">{message}</p> : null}
        {status === "loading" ? <p className="px-4 py-4 text-[13px] text-[#6f6a62]">Загружаем...</p> : null}

        <div className="overflow-x-auto">
          <table className="min-w-[1800px] w-full text-left text-[12px]">
            <thead>
              <tr className="border-b border-[#ebe7df] bg-[#faf9f7] text-[10px] uppercase tracking-[0.06em] text-[#8a857d]">
                <Th>Пользователь</Th>
                <Th>ID</Th>
                <Th>Статус аккаунта</Th>
                <Th>Доступ</Th>
                <Th>Организации</Th>
                <Th>Тип</Th>
                <Th>Роль</Th>
                <Th>Статус орг.</Th>
                <Th>Личный</Th>
                <Th>Зарегистрирован</Th>
                <Th>Последний вход</Th>
                <Th>Сессии</Th>
                <Th>IP</Th>
                <Th>User-Agent</Th>
                <Th>Telegram</Th>
                <Th>MAX</Th>
                <Th />
              </tr>
            </thead>
            <tbody>
              {items.map((user) => {
                const primary = user.memberships[0];
                const extraOrgs = user.memberships.length > 1 ? user.memberships.length - 1 : 0;
                const tg = messengerOf(user.messengers, "telegram");
                const max = messengerOf(user.messengers, "max");
                return (
                  <tr
                    key={user.id}
                    className="cursor-pointer border-b border-[#f0ede8] hover:bg-[#faf9f7]"
                    onClick={() => void openDetail(user)}
                  >
                    <Td>
                      <div className="font-medium text-[#18212f]">{user.full_name || "—"}</div>
                      <div className="text-[11px] text-[#8a857d]">{user.email}</div>
                    </Td>
                    <Td mono>{shortID(user.id)}</Td>
                    <Td>
                      <StatusPill tone={user.is_active ? "green" : "red"}>
                        {user.is_active ? "Активен" : "Заблокирован"}
                      </StatusPill>
                    </Td>
                    <Td>
                      <StatusPill tone={accessTone(user.access_level)}>{accessLabels[user.access_level]}</StatusPill>
                    </Td>
                    <Td>
                      {primary ? (
                        <>
                          <div>{primary.org_name}</div>
                          {extraOrgs > 0 ? <div className="text-[11px] text-[#8a857d]">+{extraOrgs}</div> : null}
                        </>
                      ) : (
                        "—"
                      )}
                    </Td>
                    <Td>{primary ? orgTypeLabels[primary.org_type] || primary.org_type : "—"}</Td>
                    <Td>
                      {primary ? (
                        <span className={primary.role === "superadmin" ? "font-semibold text-[#1d4ed8]" : ""}>
                          {roleLabels[primary.role] || primary.role}
                        </span>
                      ) : (
                        "—"
                      )}
                    </Td>
                    <Td>
                      {primary ? (
                        <StatusPill tone={reviewTone(primary.review_status)}>
                          {reviewLabels[primary.review_status] || primary.review_status}
                        </StatusPill>
                      ) : (
                        "—"
                      )}
                    </Td>
                    <Td>{primary ? (primary.is_personal ? "Да" : "Нет") : "—"}</Td>
                    <Td mono>{formatDate(user.created_at)}</Td>
                    <Td mono>{user.last_login_at ? formatDate(user.last_login_at) : "—"}</Td>
                    <Td mono>{user.active_sessions}</Td>
                    <Td mono>{user.last_ip || "—"}</Td>
                    <Td>
                      <span className="block max-w-[180px] truncate" title={user.last_user_agent}>
                        {shortUA(user.last_user_agent)}
                      </span>
                    </Td>
                    <Td>{messengerCell(tg)}</Td>
                    <Td>{messengerCell(max)}</Td>
                    <Td>
                      <button
                        type="button"
                        className="rounded border border-[#d7d2ca] px-2 py-1 text-[11px] hover:bg-[#ebe9e4]"
                        onClick={(e) => {
                          e.stopPropagation();
                          void openDetail(user);
                        }}
                      >
                        Открыть
                      </button>
                    </Td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          {status === "idle" && items.length === 0 ? (
            <p className="px-4 py-6 text-[13px] text-[#6f6a62]">Пользователи не найдены.</p>
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
          <aside className="fixed right-0 top-0 z-40 flex h-screen w-full max-w-[480px] flex-col border-l border-[#dedbd3] bg-white shadow-xl">
            <div className="flex items-start justify-between gap-3 border-b border-[#e5e1da] px-5 py-4">
              <div className="min-w-0">
                <p className="text-[10px] font-medium uppercase tracking-[0.07em] text-[#8a857d]">Пользователь</p>
                <h3 className="mt-1 truncate text-[16px] font-semibold text-[#18212f]">
                  {selected.full_name || selected.email}
                </h3>
                <p className="truncate text-[12px] text-[#8a857d]">{selected.email}</p>
                <p className="mt-1 font-mono text-[11px] text-[#8a857d]">{selected.id}</p>
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

              <DrawerSection title="Статусы">
                <div className="flex flex-wrap gap-2">
                  <StatusPill tone={selected.is_active ? "green" : "red"}>
                    {selected.is_active ? "Аккаунт активен" : "Аккаунт заблокирован"}
                  </StatusPill>
                  <StatusPill tone={accessTone(selected.access_level)}>
                    Доступ: {accessLabels[selected.access_level]}
                  </StatusPill>
                </div>
                <dl className="mt-3 grid grid-cols-2 gap-2 text-[12px]">
                  <DrawerField label="Зарегистрирован" value={formatDateTime(selected.created_at)} mono />
                  <DrawerField label="Обновлён" value={formatDateTime(selected.updated_at)} mono />
                  <DrawerField
                    label="Последний вход"
                    value={selected.last_login_at ? formatDateTime(selected.last_login_at) : "—"}
                    mono
                  />
                  <DrawerField label="Активных сессий" value={String(selected.active_sessions)} mono />
                  <DrawerField label="Последний IP" value={selected.last_ip || "—"} mono />
                </dl>
                {selected.last_user_agent ? (
                  <p className="mt-2 break-all font-mono text-[11px] text-[#5f6b7a]">{selected.last_user_agent}</p>
                ) : null}
              </DrawerSection>

              <DrawerSection title="Организации и роли">
                {selected.memberships.length === 0 ? (
                  <p className="text-[12px] text-[#6f6a62]">Нет членств в организациях.</p>
                ) : (
                  <div className="space-y-3">
                    {selected.memberships.map((m) => (
                      <div key={m.org_id} className="rounded-lg border border-[#ebe7df] p-3">
                        <div className="font-medium text-[#18212f]">{m.org_name}</div>
                        <div className="mt-1 flex flex-wrap gap-2">
                          <StatusPill tone="neutral">{orgTypeLabels[m.org_type] || m.org_type}</StatusPill>
                          <StatusPill tone={m.role === "superadmin" ? "blue" : "neutral"}>
                            {roleLabels[m.role] || m.role}
                          </StatusPill>
                          <StatusPill tone={reviewTone(m.review_status)}>
                            {reviewLabels[m.review_status] || m.review_status}
                          </StatusPill>
                          {m.is_personal ? <StatusPill tone="neutral">Личный</StatusPill> : null}
                        </div>
                        <dl className="mt-2 space-y-1 text-[11px] text-[#5f6b7a]">
                          <div>
                            <span className="text-[#8a857d]">ИНН: </span>
                            <span className="font-mono">{m.inn || "—"}</span>
                          </div>
                          <div>
                            <span className="text-[#8a857d]">Телефон: </span>
                            <span className="font-mono">{m.contact_phone || "—"}</span>
                          </div>
                          <div>
                            <span className="text-[#8a857d]">Сайт: </span>
                            <span>{m.website || "—"}</span>
                          </div>
                          <div>
                            <span className="text-[#8a857d]">В организации с: </span>
                            <span className="font-mono">{formatDate(m.member_since)}</span>
                          </div>
                        </dl>
                      </div>
                    ))}
                  </div>
                )}
              </DrawerSection>

              <DrawerSection title="Мессенджеры (уведомления)">
                <MessengerCard
                  provider="telegram"
                  title="Telegram"
                  link={messengerOf(selected.messengers, "telegram")}
                />
                <MessengerCard provider="max" title="MAX" link={messengerOf(selected.messengers, "max")} />
                <p className="mt-2 text-[11px] text-[#8a857d]">
                  Привязка настраивается пользователем в кабинете. Здесь видны только подтверждённые аккаунты без
                  токенов ботов.
                </p>
              </DrawerSection>

              <DrawerSection title="Сессии">
                {!selected.sessions || selected.sessions.length === 0 ? (
                  <p className="text-[12px] text-[#6f6a62]">Сессий нет.</p>
                ) : (
                  <div className="space-y-2">
                    {selected.sessions.map((s) => (
                      <div key={s.id} className="rounded-lg border border-[#ebe7df] px-3 py-2 text-[11px]">
                        <div className="flex items-center justify-between gap-2">
                          <span className="font-mono text-[#18212f]">{s.org_name || shortID(s.org_id)}</span>
                          <StatusPill tone={s.is_active ? "green" : "neutral"}>
                            {s.is_active ? "Активна" : "Завершена"}
                          </StatusPill>
                        </div>
                        <div className="mt-1 font-mono text-[#8a857d]">
                          {s.ip_address} · {formatDateTime(s.created_at)}
                        </div>
                        <div className="mt-1 truncate text-[#5f6b7a]" title={s.user_agent}>
                          {s.user_agent}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </DrawerSection>
            </div>

            <div className="border-t border-[#e5e1da] px-5 py-4">
              <div className="flex flex-wrap gap-2">
                {selected.is_active ? (
                  <button
                    type="button"
                    disabled={actionBusy}
                    className="rounded border border-[#e5484d] px-3 py-1.5 text-[12px] text-[#e5484d] hover:bg-red-50 disabled:opacity-50"
                    onClick={() => void patchActive(selected.id, false)}
                  >
                    Заблокировать
                  </button>
                ) : (
                  <button
                    type="button"
                    disabled={actionBusy}
                    className="rounded bg-[#1d4ed8] px-3 py-1.5 text-[12px] font-medium text-white hover:bg-[#1e40af] disabled:opacity-50"
                    onClick={() => void patchActive(selected.id, true)}
                  >
                    Разблокировать
                  </button>
                )}
                <button
                  type="button"
                  disabled={actionBusy || selected.active_sessions === 0}
                  className="rounded border border-[#d7d2ca] px-3 py-1.5 text-[12px] hover:bg-[#ebe9e4] disabled:opacity-50"
                  onClick={() => void revokeSessions(selected.id)}
                >
                  Отозвать сессии
                </button>
                <button
                  type="button"
                  disabled={actionBusy || isSuperadminUser(selected)}
                  title={isSuperadminUser(selected) ? "Нельзя удалить superadmin" : undefined}
                  className="rounded border border-[#e5484d] px-3 py-1.5 text-[12px] text-[#e5484d] hover:bg-red-50 disabled:opacity-50"
                  onClick={() => void deleteUser(selected)}
                >
                  Удалить
                </button>
                <button
                  type="button"
                  disabled
                  title="Будет доступно в фазе 8"
                  className="rounded border border-[#d7d2ca] px-3 py-1.5 text-[12px] text-[#8a857d] opacity-50"
                >
                  Impersonate (скоро)
                </button>
              </div>
            </div>
          </aside>
        </>
      ) : null}
    </div>
  );
}

function MessengerCard({
  provider,
  title,
  link,
}: {
  provider: "telegram" | "max";
  title: string;
  link?: MessengerLink;
}) {
  return (
    <div className="mb-2 rounded-lg border border-[#ebe7df] p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-[13px] font-medium text-[#18212f]">{title}</span>
        <StatusPill tone={link ? "green" : "neutral"}>{link ? "Привязан" : "Не привязан"}</StatusPill>
      </div>
      {link ? (
        <dl className="mt-2 space-y-1 text-[11px]">
          <DrawerField label="Username" value={link.username ? `@${link.username.replace(/^@/, "")}` : "—"} mono />
          <DrawerField label="Отображаемое имя" value={link.display_name || "—"} />
          <DrawerField label="ID в мессенджере" value={link.external_user_id || "—"} mono />
          <DrawerField
            label="Уведомления"
            value={link.notifications_enabled ? "Включены" : "Выключены"}
          />
          <DrawerField label="Верификация" value={link.is_verified ? "Подтверждён" : "Не подтверждён"} />
          <DrawerField
            label="Привязан"
            value={link.linked_at ? formatDateTime(link.linked_at) : formatDateTime(link.created_at)}
            mono
          />
        </dl>
      ) : (
        <p className="mt-2 text-[11px] text-[#8a857d]">
          Пользователь ещё не подключил {provider === "telegram" ? "Telegram" : "MAX"} для уведомлений.
        </p>
      )}
    </div>
  );
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
        className="h-[30px] rounded-lg border border-[#d7d2ca] bg-white px-2 text-[12px] text-[#18212f]"
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

function DrawerField({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-[#8a857d]">{label}</dt>
      <dd className={mono ? "font-mono text-[#18212f]" : "text-[#18212f]"}>{value}</dd>
    </div>
  );
}

function StatusPill({
  children,
  tone,
}: {
  children: ReactNode;
  tone: "green" | "amber" | "red" | "blue" | "neutral";
}) {
  const tones = {
    green: "border-[#4cc38a]/40 bg-[#4cc38a]/10 text-[#3b6d11]",
    amber: "border-[#f2a33c]/40 bg-[#f2a33c]/10 text-[#854f0b]",
    red: "border-[#e5484d]/40 bg-[#e5484d]/10 text-[#9f1239]",
    blue: "border-[#2563eb]/40 bg-[#2563eb]/10 text-[#1d4ed8]",
    neutral: "border-[#d7d2ca] bg-[#faf9f7] text-[#5f6b7a]",
  };
  return (
    <span className={`inline-flex rounded-full border px-2 py-0.5 text-[10px] font-medium ${tones[tone]}`}>
      {children}
    </span>
  );
}

function isSuperadminUser(user: AdminUser) {
  return user.memberships.some((m) => m.role === "superadmin");
}

function messengerOf(links: MessengerLink[], provider: "telegram" | "max") {
  return links.find((l) => l.provider === provider);
}

function messengerCell(link?: MessengerLink) {
  if (!link) {
    return <span className="text-[#8a857d]">—</span>;
  }
  const name = link.username ? `@${link.username.replace(/^@/, "")}` : link.display_name || "привязан";
  return (
    <span className="text-[#3b6d11]" title={link.external_user_id}>
      {name}
    </span>
  );
}

function shortID(id: string) {
  return id.length > 8 ? `${id.slice(0, 8)}…` : id;
}

function shortUA(ua: string) {
  if (!ua) return "—";
  return ua.length > 48 ? `${ua.slice(0, 45)}…` : ua;
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

function accessTone(level: string): "green" | "amber" | "red" {
  if (level === "full") return "green";
  if (level === "limited") return "amber";
  return "red";
}

function reviewTone(status: string): "green" | "amber" | "red" | "neutral" {
  if (status === "active") return "green";
  if (status === "pending_review" || status === "pending_email") return "amber";
  if (status === "rejected" || status === "suspended") return "red";
  return "neutral";
}
