import type { Metadata } from "next";
import { JetBrains_Mono, Onest, Unbounded } from "next/font/google";
import "./globals.css";

const unbounded = Unbounded({
  subsets: ["latin", "cyrillic"],
  weight: ["500", "700"],
  variable: "--font-unbounded",
});

const onest = Onest({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500", "600"],
  variable: "--font-onest",
});

const jetbrains = JetBrains_Mono({
  subsets: ["latin", "cyrillic"],
  weight: ["400", "500"],
  variable: "--font-jetbrains",
});

export const metadata: Metadata = {
  title: "ASUTPORT",
  description: "Единый портал технической поддержки АСУ ТП",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ru">
      <body className={`${unbounded.variable} ${onest.variable} ${jetbrains.variable}`}>
        {children}
      </body>
    </html>
  );
}
