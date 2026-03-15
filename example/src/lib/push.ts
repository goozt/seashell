// Converts a base64url VAPID public key to a Uint8Array for the PushManager.
export function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(base64);
  return Uint8Array.from([...raw].map((c) => c.charCodeAt(0)));
}

// Register the service worker (idempotent).
export async function registerSW(): Promise<ServiceWorkerRegistration | null> {
  if (typeof window === "undefined" || !("serviceWorker" in navigator)) return null;
  try {
    return await navigator.serviceWorker.register("/sw.js", { scope: "/" });
  } catch {
    return null;
  }
}

// Request permission and create a push subscription, returning it as JSON
// or null if denied / not supported.
export async function subscribeToPush(
  vapidPublicKey: string
): Promise<{ endpoint: string; keys: { p256dh: string; auth: string } } | null> {
  if (!vapidPublicKey) return null;
  const reg = await navigator.serviceWorker.ready;
  if (!reg.pushManager) return null;

  // Check existing subscription first.
  let sub = await reg.pushManager.getSubscription();
  if (!sub) {
    const permission = await Notification.requestPermission();
    if (permission !== "granted") return null;
    sub = await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(vapidPublicKey).buffer as ArrayBuffer,
    });
  }

  const json = sub.toJSON() as {
    endpoint: string;
    keys: { p256dh: string; auth: string };
  };
  return json;
}
