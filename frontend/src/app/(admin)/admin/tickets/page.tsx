"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { AdminShell } from "@/components/AdminShell";
import { authFetch } from "@/lib/auth-session";

type TicketRow = {
  id: string;
  subject: string;
  status: string;
  client_org_name?: string;
  client_org_type?: string;
  client_org_inn?: string;
  client_review_status?: string;
  updated_at?: string;
};

export default function AdminTicketsPage() {
  const [items, setItems] = useState<TicketRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    void authFetch("/api/v1/admin/tickets/onboarding?review_status=pending_review")
      .then(async (response) => {
        const body = (await response.json()) as { data?: TicketRow[]; error?: { message?: string } };
        if (!response.ok) {
          setError(body.error?.message || "Не удалось загрузить тикеты");
          return;
        }
        setItems(body.data || []);
      })
      .catch(() => setError("API временно недоступен"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <AdminShell breadcrumb="Тикеты onboarding">
      <div className="mx-auto max-w-5xl">
        <div className="mb-6">
          <h1 className="text-2xl font-medium text-[#18212f]">Тикеты проверки организаций</h1>
          <p className="mt-1 text-sm text-[#8a857d]">
            Переписка с заявителями и проверка документов перед активацией.
          </p>
        </div>

        {loading ? <p className="text-sm text-[#6f6a62]">Загрузка...</p> : null}
        {error ? <p className="text-sm text-[#b42318]">{error}</p> : null}

        <div className="overflow-hidden rounded-[12px] border border-[#dedbd3] bg-white">
          <div className="divide-y divide-[#ebe9e4]">
            {items.map((item) => (
              <Link
                key={item.id}
                href={`/app/admin/tickets/${item.id}`}
                className="block px-4 py-4 hover:bg-[#faf9f7]"
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="text-[14px] font-semibold text-[#18212f]">{item.client_org_name}</p>
                    <p className="mt-1 text-[12px] text-[#6f6a62]">{item.subject}</p>
                    <p className="mt-1 font-mono text-[11px] text-[#8a857d]">
                      {orgTypeLabel(item.client_org_type)} · ИНН {item.client_org_inn || "—"} · {statusLabel(item.status)}
                    </p>
                  </div>
                  <span className="text-[11px] text-[#8a857d]">{formatDate(item.updated_at)}</span>
                </div>
              </Link>
            ))}
            {!loading && items.length === 0 ? (
              <p className="px-4 py-6 text-sm text-[#6f6a62]">Нет тикетов на проверке.</p>
            ) : null}
          </div>
        </div>
      </div>
    </AdminShell>
  );
}

function orgTypeLabel(type?: string) {
  switch (type) {
    case "manufacturer":
      return "Производитель";
    case "vendor":
      return "Поставщик";
    case "integrator":
      return "Интегратор";
    case "client_org":
      return "Клиент";
    default:
      return type || "—";
  }
}

function statusLabel(status?: string) {
  switch (status) {
    case "waiting_platform":
      return "ожидает платформу";
    case "waiting_client":
      return "ожидает клиента";
    case "closed":
      return "закрыт";
    default:
      return status || "—";
  }
}

function formatDate(value?: string) {
  if (!value) {
    return "";
  }
  try {
    return new Date(value).toLocaleString("ru-RU");
  } catch {
    return value;
  }
}
