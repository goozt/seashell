"use client";

import { useRouter } from "next/navigation";
import { authApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LogOut, User } from "lucide-react";
import { LogoLink } from "@/components/layout/seashell-logo";
import { NotificationCenter } from "@/components/layout/notification-center";
import { useWebSocket } from "@/hooks/useWebSocket";
import { usePushNotifications } from "@/hooks/usePushNotifications";

export function Header() {
  const router = useRouter();
  const { hasHydrated, user, logout } = useAuthStore();
  useWebSocket();
  usePushNotifications();

  async function handleLogout() {
    await authApi.logout();
    logout();
    router.push("/");
  }

  return (
    <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur">
      <div className="flex h-14 items-center justify-between px-4">
        <LogoLink href="/" iconSize={28} />

        <div className="flex items-center gap-2">
          {hasHydrated && user && (
            <>
              <div className="hidden sm:flex items-center gap-2">
                <User className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">{(user.first_name || user.last_name) ? `${user.first_name} ${user.last_name}`.trim() : user.username}</span>
                {user.role !== "user" && (
                  <Badge variant="secondary" className="text-xs capitalize">
                    {user.role}
                  </Badge>
                )}
              </div>
              <NotificationCenter />
              <Button
                variant="ghost"
                size="icon"
                onClick={handleLogout}
                className="h-8 w-8"
                title="Sign out"
              >
                <LogOut className="h-4 w-4" />
              </Button>
            </>
          )}
        </div>
      </div>
    </header>
  );
}
