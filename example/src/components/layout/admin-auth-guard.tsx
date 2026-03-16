"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/auth";

export function AdminAuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { hasHydrated, isAuthenticated, user } = useAuthStore();

  useEffect(() => {
    if (!hasHydrated) return;
    if (!isAuthenticated) { router.replace("/login"); return; }
    if (user?.role !== "admin" && user?.role !== "superadmin") {
      router.replace("/dashboard");
    }
  }, [hasHydrated, isAuthenticated, user?.role, router]);

  if (!hasHydrated) return null;
  if (!isAuthenticated || (user?.role !== "admin" && user?.role !== "superadmin")) return null;

  return <>{children}</>;
}
