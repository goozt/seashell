"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { Header } from "@/components/layout/header";
import { MobileNav } from "@/components/layout/mobile-nav";

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { isAuthenticated, user } = useAuthStore();

  useEffect(() => {
    if (!isAuthenticated) { router.replace("/login"); return; }
    if (user?.role !== "admin" && user?.role !== "superadmin") {
      router.replace("/dashboard");
    }
  }, [isAuthenticated, user, router]);

  if (!isAuthenticated || (user?.role !== "admin" && user?.role !== "superadmin")) return null;

  return (
    <div className="flex flex-col min-h-screen">
      <Header />
      <main className="flex-1 px-4 py-4 pb-24 max-w-2xl mx-auto w-full">
        {children}
      </main>
      <MobileNav />
    </div>
  );
}
