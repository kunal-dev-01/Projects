"use client";

import Back from "@/common/Back";
import React from "react";

interface AuthLayoutProps {
  children: React.ReactNode;
}

export default function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="relative min-h-screen w-full overflow-hidden text-zinc-200 selection:bg-zinc-700">
      {/* --- GRID BACKGROUND LINES --- */}

      {/* Vertical Lines (Wide Gap) */}
      <div className="pointer-events-none fixed inset-0 flex justify-center px-6">
        <div className="h-full w-full max-w-4xl border-x border-zinc-900" />
      </div>

      {/* Horizontal Line Pairs (Aggressive Track Style) */}
      <div className="pointer-events-none fixed inset-0 flex flex-col justify-between py-[10vh]">
        {/* Top Pair */}
        <div className="flex w-full flex-col gap-3 md:gap-5">
          <div className="w-full border-t border-zinc-900" />
          <div className="w-full border-t border-zinc-900" />
        </div>

        {/* Bottom Pair */}
        <div className="flex w-full flex-col gap-3 md:gap-5">
          <div className="w-full border-t border-zinc-900" />
          <div className="w-full border-t border-zinc-900" />
        </div>
      </div>

      {/* --- LAYOUT CONTENT --- */}
      <div className="relative z-10 flex min-h-screen flex-col">
        {/* Top-Left Pinned Button Section */}
        <header className="absolute left-0 top-0 px-12 py-6 md:p-10">
          <Back type="button" label="Back" />
        </header>

        {/* Main Component Area */}
        <main className="flex flex-1 items-center justify-center px-6">
          <section className="relative w-full max-w-md">
            <div className="flex flex-col">{children}</div>
          </section>
        </main>
      </div>
    </div>
  );
}
