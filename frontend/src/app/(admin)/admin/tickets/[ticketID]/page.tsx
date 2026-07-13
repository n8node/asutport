"use client";

import Link from "next/link";
import { use } from "react";
import { AdminShell } from "@/components/AdminShell";
import { TicketThread } from "@/components/TicketThread";

export default function AdminTicketDetailPage({
  params,
}: {
  params: Promise<{ ticketID: string }>;
}) {
  const { ticketID } = use(params);

  return (
    <AdminShell breadcrumb="Тикет проверки">
      <div className="mx-auto max-w-4xl">
        <p className="mb-4 text-sm">
          <Link href="/app/admin/tickets" className="text-[#185fa5] hover:underline">
            ← Все тикеты onboarding
          </Link>
        </p>
        <TicketThread ticketID={ticketID} mode="admin" />
      </div>
    </AdminShell>
  );
}
