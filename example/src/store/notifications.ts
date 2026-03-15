import { create } from "zustand";
import type { TicketReply } from "@/types/api";

export interface AppNotification {
  id: string;
  type: "notification" | "alert" | "support";
  title: string;
  body: string;
  link?: string;
  ticketId?: string;
  reply?: TicketReply;
  read: boolean;
  createdAt: Date;
}

interface NotificationState {
  items: AppNotification[];
  add: (n: AppNotification) => void;
  markRead: (id: string) => void;
  markAllRead: () => void;
  clear: () => void;
}

export const useNotificationStore = create<NotificationState>((set) => ({
  items: [],
  add: (n) => set((s) => ({ items: [n, ...s.items].slice(0, 100) })),
  markRead: (id) =>
    set((s) => ({
      items: s.items.map((n) => (n.id === id ? { ...n, read: true } : n)),
    })),
  markAllRead: () =>
    set((s) => ({ items: s.items.map((n) => ({ ...n, read: true })) })),
  clear: () => set({ items: [] }),
}));
