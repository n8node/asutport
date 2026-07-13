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
          setError(body.error?.message || "Заявка на подключение не найдена");
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
        <h1 className="text-2xl font-medium tracking-tight text-[#18212f]">Статус компании</h1>
        <p className="mt-1 text-sm text-[#8a857d]">
          Заявка на подключение организации к платформе. Приложите документы и переписывайтесь с ASUTPORT в блоке ниже.
        </p>
      </div>

      <div className="mb-6 rounded-lg border border-[#e8d9b3] bg-[#f6f0df] px-4 py-3 text-[13px] text-[#6d4a1f]">
        <span className="flex items-center gap-2 font-semibold">
          <span className="h-2 w-2 rounded-full bg-[#ba7517]" />
          На проверке платформой
        </span>
        <p className="mt-2 pl-4 text-[12px] leading-5 text-[#9f7a3b]">
          Обычно проверка занимает 1–2 рабочих дня. Загрузите выписку ЕГРЮЛ, доверенность или иной документ,
          подтверждающий полномочия представителя.
        </p>
      </div>

      {loading ? <p className="text-sm text-[#6f6a62]">Загрузка...</p> : null}
      {!loading && error ? (
        <div className="rounded-lg border border-[#f0b8b8] bg-[#fff5f5] px-4 py-3 text-sm text-[#b42318]">
          {error}
        </div>
      ) : null}
      {!loading && ticketID ? <TicketThread ticketID={ticketID} mode="client" context="onboarding" /> : null}

      <p className="mt-6 text-sm">
        <Link href="/app/dashboard" className="text-[#185fa5] hover:underline">
          ← Вернуться в кабинет
        </Link>
      </p>
    </DashboardShell>
  );
}
