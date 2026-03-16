"use client";

import { useEffect, useRef } from "react";
import { tokenStore } from "@/lib/api";
import { useAuthStore } from "@/store/auth";
import { useNotificationStore } from "@/store/notifications";
import type { TicketReply } from "@/types/api";

export interface WSMessage {
  type: "notification" | "alert" | "support" | "auth";
  id: string;
  status?: string; // present on auth ack
  payload: {
    title?: string;
    body?: string;
    link?: string;
    severity?: string;
    ticket_id?: string;
    reply?: TicketReply;
  };
}

type SupportListener = (ticketId: string, reply: TicketReply) => void;

// Global support listeners registry (ticket page registers itself).
const supportListeners = new Map<string, SupportListener>();
export const wsEvents = {
  onSupport: (ticketId: string, fn: SupportListener) => {
    supportListeners.set(ticketId, fn);
    return () => { supportListeners.delete(ticketId); };
  },
};

// Close code 4001 means the server rejected the token — do not reconnect.
const WS_CLOSE_AUTH_FAILED = 4001;
const WS_CLOSE_CLIENT_STOP = 4000;

export function useWebSocket() {
  const { hasHydrated, isAuthenticated } = useAuthStore();
  const add = useNotificationStore((s) => s.add);
  const addRef = useRef(add);
  addRef.current = add;
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!hasHydrated || !isAuthenticated) return;

    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
    const wsBase = apiBase.replace(/^http/, "ws");
    const url = `${wsBase}/api/v1/ws`; // no token in URL

    let ws: WebSocket;
    let reconnectTimeout: ReturnType<typeof setTimeout>;
    let authenticated = false;
    let shouldReconnect = true;
    let reconnectDelayMs = 3000;

    function connect() {
      if (!shouldReconnect) return;
      authenticated = false;
      ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        // Send auth as the very first message — before any other traffic.
        const token = tokenStore.getAccess();
        if (!token) {
          shouldReconnect = false;
          ws.close(WS_CLOSE_CLIENT_STOP, "missing token");
          return;
        }
        reconnectDelayMs = 3000;
        ws.send(JSON.stringify({ type: "auth", token }));
      };

      ws.onmessage = (event) => {
        let msg: WSMessage;
        try { msg = JSON.parse(event.data); } catch { return; }

        // Auth ack must arrive first; ignore all other messages until then.
        if (!authenticated) {
          if (msg.type === "auth" && msg.status === "ok") {
            authenticated = true;
          }
          return;
        }

        if (msg.type === "support" && msg.payload.ticket_id && msg.payload.reply) {
          const ticketId = msg.payload.ticket_id;
          const reply = msg.payload.reply;
          const listener = supportListeners.get(ticketId);
          if (listener) {
            listener(ticketId, reply);
          } else {
            addRef.current({
              id: msg.id,
              type: "support",
              title: "New reply on support request",
              body: reply.message,
              link: `/dashboard/tickets/${ticketId}`,
              ticketId,
              reply,
              read: false,
              createdAt: new Date(),
            });
          }
          return;
        }

        if (msg.type === "notification") {
          addRef.current({
            id: msg.id,
            type: "notification",
            title: msg.payload.title ?? "",
            body: msg.payload.body ?? "",
            link: msg.payload.link,
            read: false,
            createdAt: new Date(),
          });
          return;
        }

        if (msg.type === "alert") {
          window.dispatchEvent(new CustomEvent("seashell:alert", { detail: msg.payload }));
          addRef.current({
            id: msg.id,
            type: "alert",
            title: msg.payload.title ?? "",
            body: msg.payload.body ?? "",
            read: false,
            createdAt: new Date(),
          });
        }
      };

      ws.onclose = (event) => {
        wsRef.current = null;
        if (!shouldReconnect) return;
        // 4001 = server rejected token; don't retry (token is invalid/expired).
        if (event.code === WS_CLOSE_AUTH_FAILED) return;
        reconnectTimeout = setTimeout(connect, reconnectDelayMs);
        reconnectDelayMs = Math.min(reconnectDelayMs * 2, 30000);
      };
    }

    connect();

    return () => {
      shouldReconnect = false;
      clearTimeout(reconnectTimeout);
      ws?.close(WS_CLOSE_CLIENT_STOP, "component unmount");
    };
  }, [hasHydrated, isAuthenticated]); // eslint-disable-line react-hooks/exhaustive-deps
}
