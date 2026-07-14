"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { authFetch } from "@/lib/auth-session";

type Ticket = {
  id: string;
  subject: string;
  status: string;
  client_org_name?: string;
  client_org_type?: string;
  client_org_inn?: string;
  client_review_status?: string;
  updated_at?: string;
  attachments?: TicketAttachment[];
};

type TicketAttachment = {
  id: string;
  filename: string;
  content_type?: string;
  size_bytes?: number;
  created_at?: string;
};

type TicketEvent = {
  id: string;
  kind: string;
  payload?: {
    text?: string;
    filename?: string;
    attachment_id?: string;
    rationale?: string;
    target_org_name?: string;
    needed_role?: string;
    missing_org_name?: string;
    message?: string;
    note?: string;
  };
  actor_name?: string;
  actor_email?: string;
  is_platform?: boolean;
  created_at: string;
};

type TicketThreadProps = {
  ticketID: string;
  mode: "client" | "admin" | "vendor";
  context?: "onboarding" | "support";
  onTicketUpdate?: (ticket: Ticket) => void;
};

export function TicketThread({ ticketID, mode, context = "support", onTicketUpdate }: TicketThreadProps) {
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [events, setEvents] = useState<TicketEvent[]>([]);
  const [text, setText] = useState("");
  const [status, setStatus] = useState<"loading" | "idle" | "error">("loading");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [rejectRationale, setRejectRationale] = useState("");
  const [resolveNote, setResolveNote] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    setStatus("loading");
    try {
      const [ticketRes, eventsRes] = await Promise.all([
        authFetch(`/api/v1/tickets/${ticketID}`),
        authFetch(`/api/v1/tickets/${ticketID}/events`),
      ]);
      const ticketBody = (await ticketRes.json()) as { data?: Ticket; error?: { message?: string } };
      const eventsBody = (await eventsRes.json()) as { data?: TicketEvent[]; error?: { message?: string } };
      if (!ticketRes.ok || !eventsRes.ok) {
        setStatus("error");
        setError(ticketBody.error?.message || eventsBody.error?.message || "Не удалось загрузить тикет");
        return;
      }
      setTicket(ticketBody.data || null);
      setEvents(eventsBody.data || []);
      if (ticketBody.data) {
        onTicketUpdate?.(ticketBody.data);
      }
      setStatus("idle");
      setError("");
    } catch {
      setStatus("error");
      setError("Сервис тикетов временно недоступен");
    }
  }, [ticketID, onTicketUpdate]);

  useEffect(() => {
    void load();
  }, [load]);

  async function sendMessage() {
    const body = text.trim();
    if (!body || busy) {
      return;
    }
    setBusy(true);
    setError("");
    try {
      const response = await authFetch(`/api/v1/tickets/${ticketID}/messages`, {
        method: "POST",
        body: JSON.stringify({ text: body }),
      });
      const payload = (await response.json()) as {
        data?: { event?: TicketEvent; ticket?: Ticket };
        error?: { message?: string };
      };
      if (!response.ok) {
        setError(payload.error?.message || "Не удалось отправить сообщение");
        return;
      }
      setText("");
      if (payload.data?.ticket) {
        setTicket(payload.data.ticket);
        onTicketUpdate?.(payload.data.ticket);
      }
      if (payload.data?.event) {
        setEvents((prev) => [...prev, payload.data!.event!]);
      } else {
        await load();
      }
    } finally {
      setBusy(false);
    }
  }

  async function uploadFile(file: File) {
    if (busy) {
      return;
    }
    setBusy(true);
    setError("");
    try {
      const contentBase64 = await fileToBase64(file);
      const response = await authFetch(`/api/v1/tickets/${ticketID}/attachments/upload`, {
        method: "POST",
        body: JSON.stringify({
          filename: file.name,
          content_type: inferContentType(file),
          content_base64: contentBase64,
        }),
      });
      const body = (await response.json()) as {
        data?: { event?: TicketEvent; ticket?: Ticket; attachments?: TicketAttachment[] };
        error?: { message?: string; code?: string };
      };
      if (!response.ok) {
        const details = [body.error?.message, body.error?.code].filter(Boolean).join(" · ");
        setError(details || "Не удалось прикрепить файл");
        return;
      }
      if (body.data?.ticket) {
        setTicket(body.data.ticket);
        onTicketUpdate?.(body.data.ticket);
      }
      if (body.data?.event) {
        setEvents((prev) => [...prev, body.data!.event!]);
      } else {
        await load();
      }
    } catch {
      setError("Не удалось прочитать файл на устройстве");
    } finally {
      setBusy(false);
      if (fileRef.current) {
        fileRef.current.value = "";
      }
    }
  }

  async function reviewOrg(action: "approve" | "reject") {
    if (busy) {
      return;
    }
    if (action === "reject" && !rejectRationale.trim()) {
      setError("Укажите причину отклонения");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const response = await authFetch(`/api/v1/admin/tickets/${ticketID}/${action}-org`, {
        method: "POST",
        body: JSON.stringify({ rationale: rejectRationale.trim() }),
      });
      const body = (await response.json()) as { data?: Ticket; error?: { message?: string } };
      if (!response.ok) {
        setError(body.error?.message || "Не удалось обновить статус организации");
        return;
      }
      if (body.data) {
        setTicket(body.data);
        onTicketUpdate?.(body.data);
      }
      await load();
    } finally {
      setBusy(false);
    }
  }

  async function resolveTicket() {
    if (busy) {
      return;
    }
    setBusy(true);
    setError("");
    try {
      const response = await authFetch(`/api/v1/tickets/${ticketID}/resolve`, {
        method: "POST",
        body: JSON.stringify({ note: resolveNote.trim() }),
      });
      const body = (await response.json()) as { data?: Ticket; error?: { message?: string } };
      if (!response.ok) {
        setError(body.error?.message || "Не удалось закрыть обращение");
        return;
      }
      if (body.data) {
        setTicket(body.data);
        onTicketUpdate?.(body.data);
      }
      await load();
    } finally {
      setBusy(false);
    }
  }

  async function openAttachment(attachmentID: string) {
    const response = await authFetch(`/api/v1/tickets/${ticketID}/attachments/${attachmentID}/url`);
    const body = (await response.json()) as { data?: { url?: string }; error?: { message?: string } };
    if (!response.ok || !body.data?.url) {
      setError(body.error?.message || "Не удалось открыть файл");
      return;
    }
    window.open(body.data.url, "_blank", "noopener,noreferrer");
  }

  const isOnboarding = context === "onboarding" && mode === "client";
  const isVendor = mode === "vendor";

  if (status === "loading") {
    return (
      <p className={`text-sm ${isVendor ? "text-[#93A0AC]" : "text-[#6f6a62]"}`}>
        {isOnboarding ? "Загружаем переписку..." : "Загружаем тикет..."}
      </p>
    );
  }
  if (status === "error") {
    return <p className="text-sm text-[#E5484D]">{error || "Ошибка загрузки"}</p>;
  }

  const closed = ticket?.status === "closed" || ticket?.status === "resolved";
  const panel = isVendor ? "rounded-lg border border-[#2A3138] bg-[#1B2025]" : "rounded-lg border border-[#dedbd3] bg-white";
  const muted = isVendor ? "text-[#93A0AC]" : "text-[#6f6a62]";
  const textMain = isVendor ? "text-[#E6EAEE]" : "text-[#18212f]";
  const eventBox = isVendor ? "rounded-lg border border-[#2A3138] bg-[#21272D]" : "rounded-lg border border-[#ebe9e4] bg-[#faf9f7]";
  const inputClass = isVendor
    ? "w-full rounded-lg border border-[#38414A] bg-[#131619] px-3 py-2 text-[13px] text-[#E6EAEE] outline-none focus:border-[#3FC8B7]"
    : "w-full rounded-lg border border-[#d7d2ca] px-3 py-2 text-[13px] outline-none focus:border-[#185fa5]";

  return (
    <div className="space-y-4">
      <header className={`${panel} p-4`}>
        <h2 className={`text-lg font-semibold ${textMain}`}>
          {isOnboarding ? "Переписка с платформой" : ticket?.subject}
        </h2>
        <div className={`mt-2 flex flex-wrap gap-3 text-[12px] ${muted}`}>
          <span>
            {isOnboarding ? "Заявка" : "Статус"}: {isOnboarding ? onboardingStatusLabel(ticket) : statusLabel(ticket?.status)}
          </span>
          {mode === "vendor" && ticket?.client_org_name ? <span>Клиент: {ticket.client_org_name}</span> : null}
          {mode === "client" && ticket?.client_org_name ? <span>{ticket.client_org_name}</span> : null}
          {mode === "admin" && ticket?.client_org_inn ? <span>ИНН: {ticket.client_org_inn}</span> : null}
          {!isOnboarding && ticket?.client_review_status ? (
            <span>Проверка org: {ticket.client_review_status}</span>
          ) : null}
          {ticket?.attachments && ticket.attachments.length > 0 ? (
            <span>Вложений: {ticket.attachments.length}</span>
          ) : null}
        </div>
        {ticket?.attachments && ticket.attachments.length > 0 ? (
          <div className="mt-3 flex flex-wrap gap-2">
            {ticket.attachments.map((att) => (
              <button
                key={att.id}
                type="button"
                className="rounded border border-[#d7d2ca] px-2.5 py-1 text-[12px] text-[#185fa5] hover:bg-[#ebe9e4]"
                onClick={() => void openAttachment(att.id)}
              >
                📎 {att.filename}
              </button>
            ))}
          </div>
        ) : null}
      </header>

      <section className={panel}>
        <div className="max-h-[480px] space-y-3 overflow-y-auto p-4">
          {events.map((event) => (
            <article key={event.id} className={`${eventBox} px-3 py-2.5`}>
              <div className={`mb-1 flex flex-wrap items-center gap-2 text-[11px] ${isVendor ? "text-[#5F6C78]" : "text-[#8a857d]"}`}>
                <span>{event.is_platform ? "Платформа ASUTPORT" : event.actor_name || event.actor_email || "Участник"}</span>
                <span>·</span>
                <span>{formatDate(event.created_at)}</span>
              </div>
              {event.kind === "message" ? (
                <p className={`whitespace-pre-wrap text-[13px] leading-6 ${textMain}`}>{event.payload?.text}</p>
              ) : null}
              {event.kind === "attachment_added" ? (
                <button
                  type="button"
                  className={`text-[13px] font-medium hover:underline ${isVendor ? "text-[#3FC8B7]" : "text-[#185fa5]"}`}
                  onClick={() => void openAttachment(event.payload?.attachment_id || "")}
                >
                  📎 {event.payload?.filename || "Вложение"}
                </button>
              ) : null}
              {event.kind === "org_approved" || event.kind === "org_rejected" ? (
                <p className={`text-[13px] leading-6 ${textMain}`}>
                  {event.kind === "org_approved" ? "Организация активирована." : "Организация отклонена."}
                  {event.payload?.rationale ? ` ${event.payload.rationale}` : ""}
                </p>
              ) : null}
              {event.kind === "escalated" ? (
                <p className={`text-[13px] leading-6 ${textMain}`}>
                  Эскалация производителю: {event.payload?.target_org_name || "контрагент"}.
                </p>
              ) : null}
              {event.kind === "fallback" ? (
                <p className={`text-[13px] leading-6 ${textMain}`}>
                  {event.payload?.message ||
                    `Сторона «${event.payload?.missing_org_name || "контрагент"}» не подключена к платформе.`}
                </p>
              ) : null}
              {event.kind === "resolved" ? (
                <p className={`text-[13px] leading-6 ${textMain}`}>
                  Обращение решено.{event.payload?.note ? ` ${event.payload.note}` : ""}
                </p>
              ) : null}
            </article>
          ))}
        </div>

        {!closed ? (
          <div className={`border-t p-4 ${isVendor ? "border-[#2A3138]" : "border-[#ebe9e4]"}`}>
            <textarea
              value={text}
              onChange={(e) => setText(e.target.value)}
              rows={3}
              placeholder="Напишите сообщение..."
              className={inputClass}
            />
            <div className="mt-3 flex flex-wrap items-center gap-2">
              <button
                type="button"
                disabled={busy || !text.trim()}
                onClick={() => void sendMessage()}
                className={
                  isVendor
                    ? "rounded-lg bg-[#3FC8B7] px-4 py-2 text-[12px] font-medium text-[#0B2723] disabled:opacity-50"
                    : "rounded-lg bg-[#18212f] px-4 py-2 text-[12px] font-medium text-white disabled:opacity-50"
                }
              >
                {busy ? "Отправка..." : "Отправить"}
              </button>
              <label
                className={
                  isVendor
                    ? "cursor-pointer rounded-lg border border-[#38414A] px-4 py-2 text-[12px] text-[#E6EAEE] hover:bg-[#21272D]"
                    : "cursor-pointer rounded-lg border border-[#d7d2ca] px-4 py-2 text-[12px] hover:bg-[#ebe9e4]"
                }
              >
                Прикрепить файл
                <input
                  ref={fileRef}
                  type="file"
                  accept=".pdf,.png,.jpg,.jpeg,image/png,image/jpeg,application/pdf"
                  className="hidden"
                  onChange={(e) => {
                    const file = e.target.files?.[0];
                    if (file) {
                      void uploadFile(file);
                    }
                  }}
                />
              </label>
              <span className={`text-[11px] ${muted}`}>PDF, PNG, JPEG до 20 МБ</span>
            </div>
          </div>
        ) : (
          <div className={`border-t p-4 text-[13px] ${muted} ${isVendor ? "border-[#2A3138]" : "border-[#ebe9e4]"}`}>
            {isOnboarding ? "Заявка на подключение закрыта." : ticket?.status === "resolved" ? "Обращение решено." : "Тикет закрыт."}
          </div>
        )}
      </section>

      {mode === "vendor" && !closed ? (
        <section className={`${panel} p-4`}>
          <h2 className={`text-[12px] font-medium uppercase tracking-[0.08em] ${isVendor ? "text-[#5F6C78]" : "text-[#8a857d]"}`}>
            Закрыть обращение
          </h2>
          <textarea
            value={resolveNote}
            onChange={(e) => setResolveNote(e.target.value)}
            rows={2}
            placeholder="Комментарий к решению (необязательно)"
            className={`mt-3 ${inputClass}`}
          />
          <div className="mt-3">
            <button
              type="button"
              disabled={busy}
              onClick={() => void resolveTicket()}
              className="rounded-lg border border-[#4CC38A] px-4 py-2 text-[12px] font-medium text-[#4CC38A] disabled:opacity-50"
            >
              Отметить решённым
            </button>
          </div>
        </section>
      ) : null}

      {mode === "admin" && ticket?.client_review_status === "pending_review" && !closed ? (
        <section className="rounded-lg border border-[#dedbd3] bg-white p-4">
          <h2 className="text-[12px] font-medium uppercase tracking-[0.08em] text-[#8a857d]">
            Решение по организации
          </h2>
          <textarea
            value={rejectRationale}
            onChange={(e) => setRejectRationale(e.target.value)}
            rows={2}
            placeholder="Комментарий (обязателен при отклонении)"
            className="mt-3 w-full rounded-lg border border-[#d7d2ca] px-3 py-2 text-[13px] outline-none focus:border-[#185fa5]"
          />
          <div className="mt-3 flex flex-wrap gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() => void reviewOrg("approve")}
              className="rounded-lg bg-[#1d4ed8] px-4 py-2 text-[12px] font-medium text-white disabled:opacity-50"
            >
              Активировать организацию
            </button>
            <button
              type="button"
              disabled={busy}
              onClick={() => void reviewOrg("reject")}
              className="rounded-lg border border-[#d7d2ca] px-4 py-2 text-[12px] disabled:opacity-50"
            >
              Отклонить
            </button>
          </div>
        </section>
      ) : null}

      {error ? <p className="text-sm text-[#b42318]">{error}</p> : null}
    </div>
  );
}

function onboardingStatusLabel(ticket?: Ticket | null) {
  if (ticket?.client_review_status === "pending_review") {
    return "на проверке";
  }
  if (ticket?.client_review_status === "rejected") {
    return "отклонена";
  }
  if (ticket?.status === "closed") {
    return "завершена";
  }
  return statusLabel(ticket?.status);
}

function statusLabel(status?: string) {
  switch (status) {
    case "waiting_client":
      return "Ожидает клиента";
    case "waiting_platform":
      return "Ожидает платформу";
    case "waiting_vendor":
      return "Ожидает вендора";
    case "resolved":
      return "Решён";
    case "closed":
      return "Закрыт";
    default:
      return status || "—";
  }
}

function formatDate(value: string) {
  try {
    return new Date(value).toLocaleString("ru-RU");
  } catch {
    return value;
  }
}

function inferContentType(file: File) {
  if (file.type) {
    return file.type;
  }
  const name = file.name.toLowerCase();
  if (name.endsWith(".pdf")) {
    return "application/pdf";
  }
  if (name.endsWith(".png")) {
    return "image/png";
  }
  if (name.endsWith(".jpg") || name.endsWith(".jpeg")) {
    return "image/jpeg";
  }
  return "";
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result;
      if (typeof result !== "string") {
        reject(new Error("read failed"));
        return;
      }
      const comma = result.indexOf(",");
      resolve(comma >= 0 ? result.slice(comma + 1) : result);
    };
    reader.onerror = () => reject(reader.error ?? new Error("read failed"));
    reader.readAsDataURL(file);
  });
}
