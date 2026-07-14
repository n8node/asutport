"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { TicketThread } from "@/components/TicketThread";
import { SlaTimer } from "@/components/dashboard/SlaTimer";
import { VendorShell } from "@/components/VendorShell";
import {
  VENDOR_PRIORITY_LABELS,
  VENDOR_TICKET_TYPE_LABELS,
  fetchVendorTicket,
  vendorStatusLabel,
  type VendorTicket,
} from "@/lib/vendor-dashboard";

export default function VendorTicketDetailPage() {
  const params = useParams();
  const ticketID = String(params.ticketID || "");
  const [ticket, setTicket] = useState<VendorTicket | null>(null);

  useEffect(() => {
    if (!ticketID) return;
    void fetchVendorTicket(ticketID).then(setTicket);
  }, [ticketID]);

  return (
    <VendorShell activePath="/app/vendor/tickets" pageTitle="Эскалация">
      <p className="mb-4">
        <Link href="/app/vendor/tickets" className="text-sm text-[#185fa5] hover:underline">
          ← К очереди
        </Link>
      </p>

      {ticket ? (
        <div className="mb-6 rounded-lg border border-[#dedbd3] bg-white p-4">
          <h1 className="text-xl font-medium text-[#18212f]">{ticket.subject}</h1>
          <div className="mt-2 flex flex-wrap gap-3 text-[12px] text-[#6f6a62]">
            <span>{ticket.client_org_name || "Клиент"}</span>
            <span>{VENDOR_TICKET_TYPE_LABELS[ticket.type] || ticket.type}</span>
            <span>{VENDOR_PRIORITY_LABELS[ticket.priority] || ticket.priority}</span>
            <span>{vendorStatusLabel(ticket.status)}</span>
            {ticket.sla_reaction_deadline ? (
              <span className="flex items-center gap-2">
                SLA: <SlaTimer deadline={ticket.sla_reaction_deadline} />
              </span>
            ) : null}
          </div>
        </div>
      ) : null}

      {ticketID ? (
        <TicketThread
          ticketID={ticketID}
          mode="vendor"
          context="support"
          onTicketUpdate={(updated) =>
            setTicket((prev) => (prev ? { ...prev, ...updated } : prev))
          }
        />
      ) : null}
    </VendorShell>
  );
}
