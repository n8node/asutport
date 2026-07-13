import { Suspense } from "react";
import { ConfirmRegistrationContent } from "@/components/ConfirmRegistrationContent";

export default function ConfirmRegistrationPage() {
  return (
    <Suspense
      fallback={
        <main className="auth-page min-h-screen px-4 py-8 text-[#1f2933]">
          <div className="mx-auto flex min-h-[calc(100vh-4rem)] w-full max-w-[520px] items-center">
            <p className="text-sm text-[#6b7280]">Загрузка...</p>
          </div>
        </main>
      }
    >
      <ConfirmRegistrationContent />
    </Suspense>
  );
}
