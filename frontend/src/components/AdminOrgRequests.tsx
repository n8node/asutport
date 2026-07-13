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
    <section className="overflow-hidden rounded-[12px] border border-[#dedbd3] bg-white">
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-[#e5e1da] px-4 py-3">
        <div>
          <p className="text-[10px] font-medium uppercase tracking-[0.07em] text-[#8a857d]">
            Заявки организаций
          </p>
          <h2 className="mt-1 text-[15px] font-semibold text-[#18212f]">Ожидают проверки</h2>
        </div>
        <button
          type="button"
          className="rounded border border-[#d7d2ca] px-2.5 py-1 text-[11px] text-[#5f6b7a] hover:bg-[#ebe9e4]"
          onClick={() => void load()}
        >
          Обновить
        </button>
      </div>

      <div className="px-4">
        {message ? <p className="mt-4 text-[13px] text-[#854f0b]">{message}</p> : null}
        {status === "loading" ? (
          <p className="mt-4 text-[13px] text-[#6f6a62]">Загружаем заявки...</p>
        ) : null}
      </div>

      <div className="divide-y divide-[#ebe7df] px-4">
        {items.map((item) => (
          <article key={item.id} className="py-4">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <p className="text-[13px] font-semibold text-[#18212f]">{item.name}</p>
                <p className="mt-1 font-mono text-[11px] text-[#8a857d]">
                  {typeLabels[item.type] || item.type} · ИНН {item.inn || "не указан"}
                </p>
                <p className="mt-2 max-w-2xl text-[13px] leading-5 text-[#5f6b7a]">
                  {item.review_comment || "Комментарий к заявке не заполнен."}
                </p>
                <div className="mt-2 flex flex-wrap gap-3 font-mono text-[11px] text-[#8a857d]">
                  {item.website ? <span>{item.website}</span> : null}
                  {item.contact_phone ? <span>{item.contact_phone}</span> : null}
                </div>
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  className="rounded bg-[#1d4ed8] px-3 py-1.5 text-[12px] font-medium text-white hover:bg-[#1e40af]"
                  onClick={() => void updateReview(item.id, "active")}
                >
                  Активировать
                </button>
                <button
                  type="button"
                  className="rounded border border-[#d7d2ca] px-3 py-1.5 text-[12px] font-medium text-[#5f6b7a] hover:bg-[#ebe9e4]"
                  onClick={() => void updateReview(item.id, "rejected")}
                >
                  Отклонить
                </button>
              </div>
            </div>
          </article>
        ))}
        {status === "idle" && items.length === 0 ? (
          <p className="py-4 text-[13px] text-[#6f6a62]">Новых заявок нет.</p>
        ) : null}
      </div>
    </section>
  );
}
