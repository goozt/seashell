"use client";

import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { User } from "@/types/api";
import { tokenStore } from "@/lib/api";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  hasHydrated: boolean;
  setHasHydrated: (value: boolean) => void;
  setUser: (user: User) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      hasHydrated: false,
      setHasHydrated: (value) => set({ hasHydrated: value }),
      setUser: (user) => set({ user, isAuthenticated: true }),
      logout: () => {
        tokenStore.clear();
        set({ user: null, isAuthenticated: false });
      },
    }),
    {
      name: "seashell_auth",
      partialize: (state) => ({ user: state.user, isAuthenticated: state.isAuthenticated }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true);
      },
    }
  )
);
