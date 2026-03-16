"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, useEffect } from "react";
import { Toaster, toast } from "sonner";
import { useAuthStore } from "@/store/auth";

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            retry: 1,
          },
        },
      })
  );

  const logout = useAuthStore((s) => s.logout);

  useEffect(() => {
    function onSessionExpired() {
      logout();
      toast.error("Session expired. Please sign in again.");
    }
    window.addEventListener("seashell:session-expired", onSessionExpired);
    return () => window.removeEventListener("seashell:session-expired", onSessionExpired);
  }, [logout]);

  useEffect(() => {
    function onAlert(e: Event) {
      const { title, body, severity } = (e as CustomEvent).detail;
      const msg = body ? `${title}: ${body}` : title;
      if (severity === "error") toast.error(msg);
      else if (severity === "warning") toast.warning(msg);
      else toast.info(msg);
    }
    window.addEventListener("seashell:alert", onAlert);
    return () => window.removeEventListener("seashell:alert", onAlert);
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <Toaster position="top-right" richColors />
    </QueryClientProvider>
  );
}
