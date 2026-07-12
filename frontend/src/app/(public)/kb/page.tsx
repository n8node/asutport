import Link from "next/link";

export default function KnowledgeBasePage() {
  return (
    <div className="kb-page min-h-screen bg-[var(--bg)] text-[var(--text)]">
      <header className="border-b border-[var(--line)] bg-[var(--panel)]">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-5">
          <div>
            <p className="font-logo text-[10px] font-medium uppercase tracking-[0.18em] text-[var(--dim)]">
              ASUTPORT
            </p>
            <h1 className="mt-1 text-2xl font-semibold">База знаний</h1>
          </div>
          <Link
            href="/dashboard"
            className="rounded-lg border border-[var(--line2)] px-4 py-2 text-sm text-[var(--mut)] hover:bg-[var(--panel2)]"
          >
            В кабинет
          </Link>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-6 py-10">
        <div className="rounded-[10px] border border-[var(--line)] bg-[var(--panel)] p-8">
          <p className="text-[var(--mut)]">
            Публичная база знаний по документации производителей АСУ ТП. SEO-статьи и виджет
            агента появятся в фазе 5.
          </p>
        </div>
      </main>
    </div>
  );
}
