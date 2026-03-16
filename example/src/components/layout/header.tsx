"use client";

import { useRouter } from "next/navigation";
import { authApi } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LogOut } from "lucide-react";
import { LogoLink } from "@/components/layout/seashell-logo";
import { NotificationCenter } from "@/components/layout/notification-center";
import { ThemeToggle } from "@/components/layout/theme-toggle";
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
    <header className="sticky top-0 z-40 bg-background/80 backdrop-blur-md border-b border-primary/10">
      <div className="flex h-14 items-center justify-between px-4 max-w-2xl mx-auto">
        <LogoLink href="/" iconSize={28} />

        <div className="flex items-center gap-1.5">
          {hasHydrated && user && (
            <>
              <div className="hidden sm:flex items-center gap-2 mr-1">
                <span className="text-sm font-medium text-foreground/80">
                  {(user.first_name || user.last_name)
                    ? `${user.first_name} ${user.last_name}`.trim()
                    : user.username}
                </span>
                {user.role !== "user" && (
                  <Badge
                    variant="outline"
                    className="text-[10px] capitalize border-primary/30 text-primary"
                  >
                    {user.role}
                  </Badge>
                )}
              </div>
              <ThemeToggle />
              <NotificationCenter />
              <Button
                variant="ghost"
                size="icon"
                onClick={handleLogout}
                className="h-8 w-8 text-muted-foreground hover:text-foreground"
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
