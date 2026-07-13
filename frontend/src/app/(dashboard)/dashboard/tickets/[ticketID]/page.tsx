"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { DashboardShell } from "@/components/DashboardShell";
import { TicketThread } from "@/components/TicketThread";
import { SlaTimer } from "@/components/dashboard/SlaTimer";
import { TICKET_PRIORITY_LABELS, TICKET_STATUS_LABELS, TICKET_TYPE_LABELS } from "@/lib/client-dashboard";
import { useEffect, useState } from "react";
import { authFetch } from "@/lib/auth-session";

type Ticket = {
  id: string;
  subject: string;
  status: string;
  type: string;
  priority: string;
  sla_reaction_deadline?: string;
};

export default function TicketDetailPage() {
  const params = useParams();
  const ticketID = String(params.ticketID || "");
  const [ticket, setTicket] = useState<Ticket | null>(null);

  useEffect(() => {
    if (!ticketID) return;
    void authFetch(`/api/v1/tickets/${ticketID}`)
      .then(async (r) => {
        const body = (await r.json()) as { data?: Ticket };
        if (r.ok) setTicket(body.data ?? null);
      })
      .catch(() => undefined);
  }, [ticketID]);

  return (
    <DashboardShell activePath="/app/dashboard/tickets" pageTitle="Тикет">
      <p className="mb-4">
        <Link href="/app/dashboard/tickets" className="text-[#185fa5] hover:underline">← К списку тикетов</Link>
      </p>

      {ticket ? (
        <div className="mb-6 rounded-lg border border-[#dedbd3] bg-white p-4">
          <h1 className="text-xl font-medium text-[#18212f]">{ticket.subject}</h1>
          <div className="mt-2 flex flex-wrap gap-3 text-[12px] text-[#6f6a62]">
            <span>{TICKET_TYPE_LABELS[ticket.type] || ticket.type}</span>
            <span>{TICKET_PRIORITY_LABELS[ticket.priority] || ticket.priority}</span>
            <span>{TICKET_STATUS_LABELS[ticket.status] || ticket.status}</span>
            {ticket.sla_reaction_deadline ? (
              <span className="flex items-center gap-2">
                SLA: <SlaTimer deadline={ticket.sla_reaction_deadline} />
              </span>
            ) : null}
          </div>
        </div>
      ) : null}

      {ticketID ? <TicketThread ticketID={ticketID} mode="client" context="support" /> : null}
    </DashboardShell>
  );
}
