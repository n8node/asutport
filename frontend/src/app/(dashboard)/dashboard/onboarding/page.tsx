"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { DashboardShell } from "@/components/DashboardShell";
import { TicketThread } from "@/components/TicketThread";
import { authFetch } from "@/lib/auth-session";

type Ticket = {
  id: string;
  subject: string;
  status: string;
};

export default function DashboardOnboardingPage() {
  const [ticketID, setTicketID] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    void authFetch("/api/v1/tickets/onboarding")
      .then(async (response) => {
        const body = (await response.json()) as { data?: Ticket; error?: { message?: string } };
        if (!response.ok) {
          setError(body.error?.message || "Тикет проверки не найден");
          return;
        }
        setTicketID(body.data?.id || "");
      })
      .catch(() => setError("Сервис временно недоступен"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <DashboardShell activePath="/app/dashboard/onboarding" reviewBanner>
      <div className="mb-6">
        <h1 className="text-2xl font-medium tracking-tight text-[#18212f]">Проверка организации</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Приложите подтверждающие документы и переписывайтесь с платформой в этом тикете.
        </p>
      </div>

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка...</p> : null}
      {!loading && error ? (
        <div className="rounded-lg border border-[#f0b8b8] bg-[#fff5f5] px-4 py-3 text-sm text-[#b42318]">
          {error}
        </div>
      ) : null}
      {!loading && ticketID ? <TicketThread ticketID={ticketID} mode="client" /> : null}

      <p className="mt-6 text-sm">
        <Link href="/app/dashboard" className="text-[#185fa5] hover:underline">
          ← Вернуться в кабинет
        </Link>
      </p>
    </DashboardShell>
  );
}
