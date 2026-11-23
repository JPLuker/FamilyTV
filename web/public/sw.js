// API responses, viewer pages and personal data never enter offline storage.
self.addEventListener('fetch', event => {
  const url = new URL(event.request.url)
  if (event.request.mode !== 'navigate' || url.origin !== self.location.origin || url.pathname !== '/') return
  event.respondWith(fetch(event.request).catch(() => new Response(
    '<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>FamilyTV offline</title><main><h1>FamilyTV is offline</h1><p>Connect to your household server to manage profiles and sessions.</p><a href="/">Try again</a></main></html>',
    {headers:{'Content-Type':'text/html; charset=utf-8','Cache-Control':'no-store'}}
  )))
})
