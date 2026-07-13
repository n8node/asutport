"use client";

import { useEffect, useState } from "react";

type OrgRequest = {
  id: string;
  name: string;
  type: string;
  inn: string;
  website: string;
  contact_phone: string;
  review_comment: string;
  review_status: string;
  created_at: string;
};

const typeLabels: Record<string, string> = {
  client_org: "Клиент",
  manufacturer: "Производитель",
  vendor: "Поставщик / вендор",
  integrator: "Интегратор",
};

export function AdminOrgRequests() {
  const [items, setItems] = useState<OrgRequest[]>([]);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("loading");
  const [message, setMessage] = useState("");

  async function load() {
    const token = sessionStorage.getItem("asutport_access_token");
    if (!token) {
      setStatus("error");
      setMessage("Нет access token. Войдите как суперадмин.");
      return;
    }

    try {
      setStatus("loading");
      const response = await fetch("/api/v1/admin/orgs?review_status=pending_review", {
        headers: { Authorization: `Bearer ${token}` },
      });
      const body = (await response.json()) as { data?: OrgRequest[]; error?: { message?: string } };
      if (!response.ok) {
        setStatus("error");
        setMessage(body.error?.message || "Не удалось загрузить заявки");
        return;
      }
      setItems(body.data || []);
      setStatus("idle");
      setMessage("");
    } catch {
      setStatus("error");
      setMessage("API заявок временно недоступен");
    }
  }

  async function updateReview(orgID: string, reviewStatus: "active" | "rejected") {
    const token = sessionStorage.getItem("asutport_access_token");
    if (!token) {
      setMessage("Нет access token. Войдите как суперадмин.");
      return;
    }
    const response = await fetch(`/api/v1/admin/orgs/${orgID}/review`, {
      method: "PATCH",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ status: reviewStatus }),
    });
    if (!response.ok) {
      setMessage("Не удалось обновить статус заявки");
      return;
    }
    await load();
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <section className="hmi-card p-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <p className="font-logo text-[10px] uppercase tracking-[0.18em] text-dim">
            Заявки организаций
          </p>
          <h2 className="mt-2 text-lg font-semibold text-text">Ожидают проверки</h2>
        </div>
        <button type="button" className="hmi-btn-secondary" onClick={() => void load()}>
          Обновить
        </button>
      </div>

      {message ? <p className="mt-4 text-sm text-lampAmber">{message}</p> : null}
      {status === "loading" ? <p className="mt-4 text-sm text-mut">Загружаем заявки...</p> : null}

      <div className="mt-5 divide-y divide-line">
        {items.map((item) => (
          <article key={item.id} className="py-4">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <p className="text-sm font-semibold text-text">{item.name}</p>
                <p className="mt-1 font-mono text-xs text-dim">
                  {typeLabels[item.type] || item.type} · ИНН {item.inn || "не указан"}
                </p>
                <p className="mt-2 text-sm text-mut">
                  {item.review_comment || "Комментарий к заявке не заполнен."}
                </p>
                <div className="mt-2 flex flex-wrap gap-3 font-mono text-xs text-dim">
                  {item.website ? <span>{item.website}</span> : null}
                  {item.contact_phone ? <span>{item.contact_phone}</span> : null}
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  className="hmi-btn-primary"
                  onClick={() => void updateReview(item.id, "active")}
                >
                  Активировать
                </button>
                <button
                  type="button"
                  className="hmi-btn-secondary"
                  onClick={() => void updateReview(item.id, "rejected")}
                >
                  Отклонить
                </button>
              </div>
            </div>
          </article>
        ))}
        {status === "idle" && items.length === 0 ? (
          <p className="py-4 text-sm text-mut">Новых заявок нет.</p>
        ) : null}
      </div>
    </section>
  );
}
