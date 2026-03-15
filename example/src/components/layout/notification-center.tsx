"use client";

import { Bell, MessageSquareText, AlertTriangle, Info } from "lucide-react";
import { useNotificationStore } from "@/store/notifications";
import { Button } from "@/components/ui/button";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { formatRelative } from "@/lib/utils";

export function NotificationCenter() {
  const { items, markRead, markAllRead } = useNotificationStore();
  const [open, setOpen] = useState(false);
  const router = useRouter();

  const unread = items.filter((n) => !n.read).length;

  function handleClick(id: string, link?: string) {
    markRead(id);
    setOpen(false);
    if (link) router.push(link);
  }

  return (
    <div className="relative">
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8 relative"
        onClick={() => setOpen((v) => !v)}
        aria-label="Notifications"
      >
        <Bell className="h-4 w-4" />
        {unread > 0 && (
          <span className="absolute -top-0.5 -right-0.5 bg-red-500 text-white text-[9px] font-bold rounded-full min-w-[16px] h-4 flex items-center justify-center px-0.5">
            {unread > 99 ? "99+" : unread}
          </span>
        )}
      </Button>

      {open && (
        <>
          <div
            className="fixed inset-0 z-40"
            onClick={() => setOpen(false)}
          />
          <div className="absolute right-0 top-9 z-50 w-80 rounded-lg border bg-background shadow-lg overflow-hidden">
            <div className="flex items-center justify-between px-3 py-2 border-b">
              <span className="text-sm font-semibold">Notifications</span>
              {unread > 0 && (
                <button
                  className="text-xs text-muted-foreground hover:text-foreground"
                  onClick={markAllRead}
                >
                  Mark all read
                </button>
              )}
            </div>

            <div className="max-h-96 overflow-y-auto divide-y">
              {items.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-8">
                  No notifications
                </p>
              ) : (
                items.map((n) => (
                  <button
                    key={n.id}
                    className={`w-full text-left px-3 py-2.5 flex gap-2.5 hover:bg-accent/50 transition-colors ${
                      !n.read ? "bg-accent/20" : ""
                    }`}
                    onClick={() => handleClick(n.id, n.link)}
                  >
                    <div className="mt-0.5 shrink-0">
                      {n.type === "support" ? (
                        <MessageSquareText className="h-4 w-4 text-primary" />
                      ) : n.type === "alert" ? (
                        <AlertTriangle className="h-4 w-4 text-orange-500" />
                      ) : (
                        <Info className="h-4 w-4 text-blue-500" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-xs font-medium truncate">{n.title}</p>
                      <p className="text-xs text-muted-foreground truncate">
                        {n.body}
                      </p>
                      <p className="text-[10px] text-muted-foreground mt-0.5">
                        {formatRelative(n.createdAt.toISOString())}
                      </p>
                    </div>
                    {!n.read && (
                      <span className="mt-1.5 w-2 h-2 rounded-full bg-primary shrink-0" />
                    )}
                  </button>
                ))
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
