"use client";

// Minimal toast hook — shadcn's full toast implementation is added via
// `npx shadcn@latest add toast` which writes the complete version.
// This stub provides the same API surface so the rest of the code compiles
// before the shadcn CLI has been run.

import { useState, useCallback } from "react";

export type ToastVariant = "default" | "destructive";

export interface Toast {
  id: string;
  title?: string;
  description?: string;
  variant?: ToastVariant;
}

let toastIdCounter = 0;

// Global listeners so any component can add toasts
const listeners: Array<(toast: Toast) => void> = [];

export function addToast(toast: Omit<Toast, "id">) {
  const t: Toast = { id: String(++toastIdCounter), ...toast };
  listeners.forEach((fn) => fn(t));
}

export function useToast() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const toast = useCallback((t: Omit<Toast, "id">) => {
    addToast(t);
  }, []);

  return { toast, toasts };
}
