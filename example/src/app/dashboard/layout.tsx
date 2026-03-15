"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";
import { Header } from "@/components/layout/header";
import { MobileNav } from "@/components/layout/mobile-nav";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [mounted, setMounted] = useState(false);

  useEffect(() => { setMounted(true); }, []);

  useEffect(() => {
    if (!mounted) return;
    if (!isAuthenticated) router.replace("/login");
  }, [mounted, isAuthenticated, router]);

  if (!mounted) return null;
  if (!isAuthenticated) return null;

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
