import type { ReactNode } from "react";

export function DashboardPanel({ title, children, action }: { title: string; children: ReactNode; action?: ReactNode }) {
  return (
    <section className="rounded-lg border border-[#dedbd3] bg-white p-4 sm:p-5">
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="text-[10px] font-medium uppercase tracking-[0.12em] text-[#9a948c]">{title}</h2>
        {action}
      </div>
      {children}
    </section>
  );
}

export function DashboardEmpty({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-[#d7d2ca] bg-[#faf9f7] px-5 py-8 text-center">
      <p className="text-[14px] font-medium text-[#18212f]">{title}</p>
      <p className="mx-auto mt-2 max-w-md text-[13px] leading-5 text-[#6f6a62]">{children}</p>
    </div>
  );
}

export function FieldLabel({ children }: { children: ReactNode }) {
  return <label className="mb-1 block text-[11px] font-medium uppercase tracking-wide text-[#8a857d]">{children}</label>;
}

export function TextInput(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full rounded-lg border border-[#d7d2ca] bg-white px-3 py-2 text-[13px] text-[#18212f] outline-none focus-visible:ring-2 focus-visible:ring-[#185fa5] ${props.className || ""}`}
    />
  );
}

export function TextArea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={`w-full rounded-lg border border-[#d7d2ca] bg-white px-3 py-2 text-[13px] text-[#18212f] outline-none focus-visible:ring-2 focus-visible:ring-[#185fa5] ${props.className || ""}`}
    />
  );
}

export function SelectInput(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={`w-full rounded-lg border border-[#d7d2ca] bg-white px-3 py-2 text-[13px] text-[#18212f] outline-none focus-visible:ring-2 focus-visible:ring-[#185fa5] ${props.className || ""}`}
    />
  );
}

export function PrimaryButton({
  children,
  type = "button",
  disabled,
  onClick,
}: {
  children: ReactNode;
  type?: "button" | "submit";
  disabled?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type={type}
      disabled={disabled}
      onClick={onClick}
      className="rounded-full bg-[#18212f] px-4 py-2 text-[12px] font-medium text-white hover:opacity-90 disabled:opacity-50"
    >
      {children}
    </button>
  );
}

export function SecondaryButton({
  children,
  type = "button",
  disabled,
  onClick,
}: {
  children: ReactNode;
  type?: "button" | "submit";
  disabled?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type={type}
      disabled={disabled}
      onClick={onClick}
      className="rounded-full border border-[#d7d2ca] px-4 py-2 text-[12px] font-medium text-[#18212f] hover:bg-[#ebe9e4] disabled:opacity-50"
    >
      {children}
    </button>
  );
}

export function ErrorNote({ children }: { children: ReactNode }) {
  return (
    <div className="rounded-lg border border-[#f0b8b8] bg-[#fff5f5] px-4 py-3 text-[13px] text-[#b42318]">
      {children}
    </div>
  );
}
