// From webpush-go example
// self.addEventListener("push", (event) => {
//   console.log("[Service Worker] Push Received.");
//   console.log(`[Service Worker] Push had this data: "${event.data.text()}"`);
//
//   const title = "Test Webpush";
//   const options = {
//     body: event.data.text(),
//   };
//
//   event.waitUntil(self.registration.showNotification(title, options));
// });

// public/service-worker.js
self.addEventListener("install", (event) => {
  console.log("Service Worker installing...");
  self.skipWaiting(); // Forces the waiting service worker to become the active service worker
});

self.addEventListener("activate", (event) => {
  console.log("Service Worker activating...");
  event.waitUntil(clients.claim()); // Take control of un-controlled clients
});

self.addEventListener("push", (event) => {
  console.log("Push event received:", event);

  let data = {};
  if (event.data) {
    try {
      data = event.data.json(); // Assuming the payload is JSON
    } catch (e) {
      console.error("Error parsing push data:", e);
      data = { title: "New Notification", body: event.data.text() }; // Fallback to plain text
    }
  } else {
    data = { title: "New Notification", body: "You have a new message!" };
  }

  const title = data.title || "Default Title";
  const options = {
    body: data.body || "Default body message.",
    icon: data.icon || "/path/to/default-icon.png", // Make sure this path is valid relative to your domain root
    badge: data.badge || "/path/to/default-badge.png", // For Android
    data: data.data || { url: "/" }, // Custom data that can be used when notification is clicked
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener("notificationclick", (event) => {
  console.log("Notification click received:", event);
  event.notification.close(); // Close the notification

  const urlToOpen =
    event.notification.data && event.notification.data.url
      ? event.notification.data.url
      : "/";

  event.waitUntil(
    clients.matchAll({ type: "window" }).then((clientList) => {
      for (const client of clientList) {
        if (client.url === urlToOpen && "focus" in client) {
          return client.focus(); // If tab is open, focus it
        }
      }
      // If no tab is open or focused, open a new one
      return clients.openWindow(urlToOpen);
    }),
  );
});
