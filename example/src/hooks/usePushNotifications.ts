"use client";

import { useEffect } from "react";
import { useAuthStore } from "@/store/auth";
import { registerSW, subscribeToPush } from "@/lib/push";

const PUSH_SUBSCRIBE_URL = "/api/v1/user/push/subscribe";
const VAPID_KEY_URL = "/api/v1/push/vapid-public-key";

async function fetchVapidKey(): Promise<string> {
  const res = await fetch(VAPID_KEY_URL);
  if (!res.ok) return "";
  const data = await res.json();
  return data?.data?.public_key ?? "";
}

async function sendSubscriptionToServer(
  sub: { endpoint: string; keys: { p256dh: string; auth: string } },
  accessToken: string
) {
  await fetch(PUSH_SUBSCRIBE_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(sub),
  });
}

export function usePushNotifications() {
  const { isAuthenticated } = useAuthStore();

  useEffect(() => {
    if (!isAuthenticated) return;
    if (typeof window === "undefined") return;
    if (!("Notification" in window) || !("serviceWorker" in navigator)) return;

    (async () => {
      const reg = await registerSW();
      if (!reg) return;

      const vapidKey = await fetchVapidKey();
      if (!vapidKey) return; // push not configured on server

      const sub = await subscribeToPush(vapidKey);
      if (!sub) return;

      // Get access token from localStorage directly (avoid importing tokenStore cycle).
      const token = localStorage.getItem("seashell_access");
      if (!token) return;

      await sendSubscriptionToServer(sub, token);
    })();
  }, [isAuthenticated]);
}
