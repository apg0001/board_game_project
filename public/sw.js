/* global self, caches, fetch */

const CACHE_NAME = "board-table-v2";
const PLAYING_CARD_RANKS = ["ace", "2", "3", "4", "5", "6", "7", "8", "9", "10", "jack", "queen", "king"];
const PLAYING_CARD_SUITS = ["clubs", "diamonds", "hearts", "spades"];
const PLAYING_CARD_ASSETS = [
  "/assets/cards/playing/back.png",
  "/assets/cards/playing/black_joker.svg",
  "/assets/cards/playing/red_joker.svg",
  ...PLAYING_CARD_SUITS.flatMap((suit) =>
    PLAYING_CARD_RANKS.map((rank) => `/assets/cards/playing/${rank}_of_${suit}.svg`)
  )
];
const APP_SHELL = [
  "/",
  "/index.html",
  "/manifest.webmanifest",
  "/icons/icon.svg",
  "/icons/maskable-icon.svg",
  "/assets/table-pattern.svg",
  ...PLAYING_CARD_ASSETS
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL))
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)))
      )
  );
  self.clients.claim();
});

self.addEventListener("fetch", (event) => {
  if (event.request.method !== "GET") return;

  event.respondWith(
    caches.match(event.request).then((cached) => {
      if (cached) return cached;

      return fetch(event.request)
        .then((response) => {
          const copy = response.clone();
          caches.open(CACHE_NAME).then((cache) => cache.put(event.request, copy));
          return response;
        })
        .catch(() => caches.match("/index.html"));
    })
  );
});
