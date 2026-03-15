self.addEventListener("install", () => self.skipWaiting());
self.addEventListener("activate", (e) => e.waitUntil(self.clients.claim()));

// Handle incoming push messages (app closed or backgrounded).
self.addEventListener("push", (event) => {
  let data = { title: "SeaShell", body: "You have a new notification." };
  try { data = event.data?.json() ?? data; } catch {}

  event.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: data.icon ?? "/logo-icon.png",
      badge: "/logo-icon.png",
      tag: data.tag ?? "seashell",
      renotify: true,
      data: { url: data.url ?? "/" },
    })
  );
});

// Open the app (or focus an existing tab) when notification is clicked.
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = event.notification.data?.url ?? "/";
  event.waitUntil(
    self.clients
      .matchAll({ type: "window", includeUncontrolled: true })
      .then((clientList) => {
        for (const client of clientList) {
          if (new URL(client.url).pathname === new URL(target, self.location.origin).pathname) {
            return client.focus();
          }
        }
        return self.clients.openWindow(target);
      })
  );
});
