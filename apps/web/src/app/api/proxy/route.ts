import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

function escapeJs(s: string) {
  return s
    .replace(/\\/g, "\\\\")
    .replace(/'/g, "\\'")
    .replace(/\n/g, "\\n")
    .replace(/\r/g, "\\r")
    .replace(/</g, "\\u003c");
}

/** Unwrap nested /api/proxy?url=... mistakes (any host). */
export function unwrapProxyUrl(raw: string, depth = 0): string {
  if (!raw || depth > 5) return raw;
  try {
    const u = new URL(raw);
    if (u.pathname === "/api/proxy" || u.pathname.endsWith("/api/proxy")) {
      const inner = u.searchParams.get("url");
      if (inner) return unwrapProxyUrl(inner, depth + 1);
    }
  } catch {
    if (raw.includes("/api/proxy?url=")) {
      try {
        const u = new URL(raw, "http://local.invalid");
        const inner = u.searchParams.get("url");
        if (inner) return unwrapProxyUrl(inner, depth + 1);
      } catch {
        /* ignore */
      }
    }
  }
  return raw;
}

function isAuthUrl(raw: string): boolean {
  try {
    const u = new URL(raw);
    const host = u.hostname.toLowerCase();
    const href = u.href.toLowerCase();
    if (
      host === "accounts.google.com" ||
      host.endsWith(".google.com") && (/\/gsi\//.test(href) || /\/o\/oauth/.test(href) || host.startsWith("accounts."))
    ) {
      return true;
    }
    if (host === "login.microsoftonline.com" || host === "login.live.com") return true;
    if (host === "appleid.apple.com") return true;
    if (host === "github.com" && href.includes("/login")) return true;
    if (host.endsWith("linkedin.com") && (/\/oauth/i.test(href) || /\/uas\//i.test(href))) return true;
    if (host.includes("auth0.com") || host.includes("okta.com")) return true;
    if (host.includes("facebook.com") && href.includes("login")) return true;
    return false;
  } catch {
    return false;
  }
}

function authBlockedPage(proxyOrigin: string, target: string, jobPage?: string) {
  const safe = unwrapProxyUrl(target).replace(/"/g, "&quot;");
  const back = jobPage
    ? `${proxyOrigin}/api/proxy?url=${encodeURIComponent(jobPage)}`
    : proxyOrigin;
  const openReal = (jobPage || unwrapProxyUrl(target)).replace(/"/g, "&quot;");
  const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <base href="${proxyOrigin}/" />
  <title>Sign-in can’t run in embed</title>
  <style>
    body{margin:0;font-family:system-ui,sans-serif;background:#0a0a0a;color:#f5f5f5;display:grid;place-items:center;min-height:100vh;padding:24px}
    .card{max-width:540px;border:1px solid #2a2a2a;background:#141414;border-radius:16px;padding:24px;line-height:1.5}
    .actions{display:flex;flex-wrap:wrap;gap:10px;margin-top:16px}
    a,button{display:inline-flex;padding:12px 18px;border-radius:999px;font-weight:700;text-decoration:none;border:0;cursor:pointer;font-size:0.95rem}
    .primary{background:#f5f5f5;color:#0a0a0a}
    .ghost{background:transparent;color:#f5f5f5;border:1px solid #333}
    p{color:#a3a3a3}
    code{color:#ddd;word-break:break-all;font-size:0.78rem}
  </style>
</head>
<body>
  <div class="card">
    <h1 style="margin:0 0 8px;font-size:1.35rem">Google / account login can’t run inside JobRight</h1>
    <p>Sign-in providers (Google, etc.) block being opened through our embed for security. That’s why you see 403.</p>
    <p>To log in: open the real application site in a new tab, sign in there, then return here to continue applying / autofill.</p>
    <p><code>${safe}</code></p>
    <div class="actions">
      <a class="primary" href="${openReal}" target="_blank" rel="noopener">Open site to sign in</a>
      <a class="ghost" href="${back}">Back to job page</a>
    </div>
  </div>
  <script>
    try {
      window.parent.postMessage({ type: 'jobright-auth-blocked', url: ${JSON.stringify(unwrapProxyUrl(target))} }, '*');
    } catch (e) {}
  </script>
</body>
</html>`;
  return new NextResponse(html, {
    status: 200,
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}

function fallbackPage(proxyOrigin: string, target: string, reason: string) {
  const real = unwrapProxyUrl(target);
  const safe = real.replace(/"/g, "&quot;");
  const proxied = `${proxyOrigin}/api/proxy?url=${encodeURIComponent(real)}`;
  const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <base href="${proxyOrigin}/" />
  <title>JobRight apply</title>
  <style>
    body{margin:0;font-family:system-ui,sans-serif;background:#0a0a0a;color:#f5f5f5;display:grid;place-items:center;min-height:100vh;padding:24px}
    .card{max-width:520px;border:1px solid #2a2a2a;background:#141414;border-radius:16px;padding:24px;line-height:1.5}
    a{display:inline-flex;margin-top:16px;padding:12px 18px;border-radius:999px;background:#f5f5f5;color:#0a0a0a;font-weight:700;text-decoration:none}
    p{color:#a3a3a3}
    code{color:#ddd;word-break:break-all}
  </style>
</head>
<body>
  <div class="card">
    <h1 style="margin:0 0 8px;font-size:1.4rem">Couldn’t load this application page</h1>
    <p>${reason}</p>
    <p><code>${safe}</code></p>
    <a href="${proxied}">Retry inside JobRight</a>
  </div>
</body>
</html>`;
  return new NextResponse(html, {
    status: 200,
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "no-store",
    },
  });
}

/**
 * Lightweight bridge:
 * - Does NOT rewrite every link (that caused full reload loops on SPAs)
 * - Does NOT location.assign on pushState (that caused infinite refresh)
 * - Only strips target=_blank, traps window.open, keeps history under /api/proxy
 * - Autofill without stealing focus
 */
function buildBridge(pageUrl: string, proxyOrigin: string) {
  const page = escapeJs(pageUrl);
  const origin = escapeJs(proxyOrigin);
  return `<script>
(function(){
  if (window.__jobrightBridge) return;
  window.__jobrightBridge = true;

  var PAGE_URL = '${page}';
  var PROXY_ORIGIN = '${origin}';
  var lastPayload = null;
  var userTyping = false;
  var fillTimer = null;
  var nativeOpen = window.open.bind(window);

  function toProxyHistoryUrl(absHref) {
    return PROXY_ORIGIN + '/api/proxy?url=' + encodeURIComponent(absHref);
  }

  function absoluteFromPage(raw) {
    return new URL(String(raw), PAGE_URL).href;
  }

  function isAuth(raw) {
    try {
      var u = new URL(String(raw), PAGE_URL);
      var host = u.hostname.toLowerCase();
      var href = u.href.toLowerCase();
      if (host === 'accounts.google.com') return true;
      if (host.indexOf('google.') >= 0 && (href.indexOf('/gsi/') >= 0 || href.indexOf('/o/oauth') >= 0)) return true;
      if (host === 'login.microsoftonline.com' || host === 'login.live.com') return true;
      if (host === 'appleid.apple.com') return true;
      if (host === 'github.com' && href.indexOf('/login') >= 0) return true;
      if (host.indexOf('linkedin.com') >= 0 && (href.indexOf('/oauth') >= 0 || href.indexOf('/uas/') >= 0)) return true;
      if (host.indexOf('auth0.com') >= 0 || host.indexOf('okta.com') >= 0) return true;
      return false;
    } catch (e) {
      return false;
    }
  }

  // Keep SPA history from leaving /api/proxy — never full-reload, never change iframe path.
  try {
    var _push = history.pushState.bind(history);
    var _replace = history.replaceState.bind(history);
    function stay(nativeFn, state, title, url) {
      if (url != null && url !== '') {
        try {
          PAGE_URL = absoluteFromPage(url);
          window.parent.postMessage({ type: 'jobright-navigate', url: PAGE_URL }, '*');
        } catch (e) {}
      }
      return nativeFn(state, title, window.location.href);
    }
    history.pushState = function(state, title, url) { return stay(_push, state, title, url); };
    history.replaceState = function(state, title, url) { return stay(_replace, state, title, url); };
  } catch (e) {}

  // Auth popups must be REAL popups. Everything else stays in the iframe via proxy.
  try {
    window.open = function(url, name, specs) {
      if (!url) return null;
      try {
        var abs = absoluteFromPage(url);
        if (isAuth(abs)) {
          try {
            window.parent.postMessage({ type: 'jobright-auth-popup', url: abs }, '*');
          } catch (e) {}
          return nativeOpen(abs, name || 'jobright-auth', specs || 'popup=yes,width=520,height=640');
        }
        PAGE_URL = abs;
        window.location.assign(toProxyHistoryUrl(abs));
      } catch (e) {}
      return null;
    };
  } catch (e) {}

  function scrubTargets(root) {
    var scope = root && root.querySelectorAll ? root : document;
    try {
      scope.querySelectorAll('a[target], form[target], area[target]').forEach(function(el){
        var href = el.getAttribute('href') || el.getAttribute('action') || '';
        if (href && isAuth(href)) return; // let Google keep target=_blank / popup behavior
        el.removeAttribute('target');
      });
    } catch (e) {}
  }

  document.addEventListener('click', function(e) {
    var a = e.target && e.target.closest ? e.target.closest('a[href]') : null;
    if (!a) return;
    var href = a.getAttribute('href');
    if (!href || href.charAt(0) === '#' || href.indexOf('javascript:') === 0) return;

    try {
      var abs = absoluteFromPage(href);
      if (isAuth(abs)) {
        // Native popup / new tab for Google login — do not proxy (403).
        e.preventDefault();
        nativeOpen(abs, 'jobright-auth', 'popup=yes,width=520,height=640');
        try { window.parent.postMessage({ type: 'jobright-auth-popup', url: abs }, '*'); } catch (err) {}
        return;
      }
    } catch (err) {}

    a.removeAttribute('target');

    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button === 1) {
      e.preventDefault();
      try {
        window.location.assign(toProxyHistoryUrl(absoluteFromPage(href)));
      } catch (err) {}
      return;
    }

    if (/^https?:\\/\\//i.test(href)) {
      try {
        var abs2 = absoluteFromPage(href);
        var curHost = new URL(PAGE_URL).hostname;
        var nextHost = new URL(abs2).hostname;
        if (nextHost !== curHost) {
          e.preventDefault();
          window.location.assign(toProxyHistoryUrl(abs2));
        }
      } catch (err) {}
    }
  }, true);

  document.addEventListener('focusin', function(e) {
    var t = e.target;
    if (t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT')) userTyping = true;
  }, true);
  document.addEventListener('pointerdown', function(){ userTyping = true; }, true);

  function setNativeValue(el, value) {
    if (value == null || value === '') return false;
    if (el.value && String(el.value).trim() === String(value).trim()) return true;
    var proto = el.tagName === 'TEXTAREA' ? window.HTMLTextAreaElement.prototype : window.HTMLInputElement.prototype;
    var desc = Object.getOwnPropertyDescriptor(proto, 'value');
    if (desc && desc.set) desc.set.call(el, value); else el.value = value;
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
    return true;
  }

  function visible(el) {
    if (!el || el.disabled || el.readOnly) return false;
    var s = window.getComputedStyle(el);
    if (s.display === 'none' || s.visibility === 'hidden') return false;
    var r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  }

  function labelText(el) {
    var bits = [];
    if (el.id) {
      try {
        var byFor = document.querySelector('label[for=\"' + CSS.escape(el.id) + '\"]');
        if (byFor) bits.push(byFor.textContent || '');
      } catch (e) {}
    }
    var parentLabel = el.closest('label');
    if (parentLabel) bits.push(parentLabel.textContent || '');
    bits.push(el.getAttribute('aria-label') || '');
    bits.push(el.getAttribute('placeholder') || '');
    bits.push(el.getAttribute('name') || '');
    bits.push(el.id || '');
    bits.push(el.getAttribute('autocomplete') || '');
    return bits.join(' ').toLowerCase();
  }

  function pick(data, keys) {
    for (var i = 0; i < keys.length; i++) if (data[keys[i]]) return String(data[keys[i]]);
    return '';
  }

  function matchValue(el, data) {
    var t = labelText(el);
    var type = (el.getAttribute('type') || el.type || '').toLowerCase();
    if (type === 'email' || /e-?mail/.test(t)) return pick(data, ['email']);
    if (type === 'tel' || /phone|mobile|telefon|tel\\b/.test(t)) return pick(data, ['phone']);
    if (/linkedin/.test(t)) return pick(data, ['linkedin']);
    if (/github/.test(t)) return pick(data, ['github']);
    if (/portfolio|website|homepage|url/.test(t)) return pick(data, ['website']);
    if (/skill|kompeten|technologies/.test(t)) return pick(data, ['skills']);
    if (/cover|anschreiben|motivation|message|letter|nachricht/.test(t)) return pick(data, ['coverLetter', 'cover_letter']);
    if (/first.?name|vorname|given.?name/.test(t)) {
      var full = pick(data, ['name']); return full ? full.split(/\\s+/)[0] : '';
    }
    if (/last.?name|nachname|family.?name|surname/.test(t)) {
      var full2 = pick(data, ['name']); if (!full2) return '';
      var parts = full2.trim().split(/\\s+/); return parts.length > 1 ? parts.slice(1).join(' ') : '';
    }
    if (/full.?name|your.?name|candidate.?name|name|vollst/.test(t) && type !== 'hidden') return pick(data, ['name']);
    if (/location|city|standort|address|adresse/.test(t)) return pick(data, ['location']);
    return '';
  }

  function fillAll(data, force) {
    if (!data) return 0;
    if (userTyping && !force) return 0;
    var active = document.activeElement;
    var count = 0;
    document.querySelectorAll('input, textarea').forEach(function(el){
      if (!force && active === el) return;
      var type = (el.getAttribute('type') || el.type || '').toLowerCase();
      if (['hidden','submit','button','checkbox','radio','file','password','image'].indexOf(type) >= 0) return;
      if (!visible(el) && type !== 'email') return;
      var value = matchValue(el, data);
      if (!value) return;
      if (setNativeValue(el, value)) count++;
    });
    try { window.parent.postMessage({ type: 'jobright-autofill-result', filled: count, page: PAGE_URL }, '*'); } catch (e) {}
    return count;
  }

  window.addEventListener('message', function(event){
    if (!event.data || event.data.type !== 'jobright-autofill') return;
    userTyping = false;
    lastPayload = event.data.payload || {};
    fillAll(lastPayload, true);
    if (fillTimer) clearTimeout(fillTimer);
    fillTimer = setTimeout(function(){ fillAll(lastPayload, true); }, 1200);
  });

  function hello() {
    try {
      window.parent.postMessage({ type: 'jobright-autofill-ready', page: PAGE_URL }, '*');
      window.parent.postMessage({ type: 'jobright-navigate', url: PAGE_URL }, '*');
    } catch (e) {}
  }

  scrubTargets(document);
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', hello);
  else hello();

  var obs = new MutationObserver(function(){ scrubTargets(document); });
  obs.observe(document.documentElement, { childList: true, subtree: true });
})();
</script>`;
}

function rewriteHtml(body: string, finalUrl: string, proxyOrigin: string) {
  const final = new URL(finalUrl);
  const assetBase = `${final.protocol}//${final.host}/`;

  body = body
    .replace(/<meta[^>]+http-equiv=["']?Content-Security-Policy["']?[^>]*>/gi, "")
    .replace(/<meta[^>]+http-equiv=["']?X-Frame-Options["']?[^>]*>/gi, "");

  body = body.replace(/\s+target=(["'])_blank\1/gi, "");
  body = body.replace(/<base\b[^>]*>/gi, "");
  body = body.replace(/<head([^>]*)>/i, `<head$1><base href="${assetBase}">`);

  const bridge = buildBridge(finalUrl, proxyOrigin);
  body = /<\/body>/i.test(body)
    ? body.replace(/<\/body>/i, `${bridge}</body>`)
    : body + bridge;

  return body;
}

export async function GET(req: NextRequest) {
  const rawParam = req.nextUrl.searchParams.get("url");
  if (!rawParam) {
    return NextResponse.json({ error: "url is required" }, { status: 400 });
  }

  const proxyOrigin = req.nextUrl.origin;
  const raw = unwrapProxyUrl(rawParam);

  let target: URL;
  try {
    target = new URL(raw);
  } catch {
    return NextResponse.json({ error: "invalid url" }, { status: 400 });
  }

  if (target.protocol !== "http:" && target.protocol !== "https:") {
    return NextResponse.json({ error: "invalid protocol" }, { status: 400 });
  }

  const host = target.hostname.toLowerCase();
  if (host === "localhost" || host === "127.0.0.1" || host.endsWith(".local")) {
    return NextResponse.json({ error: "url not allowed" }, { status: 400 });
  }

  if (target.pathname.includes("/api/proxy")) {
    return fallbackPage(
      proxyOrigin,
      raw,
      "Bad nested proxy URL — tap Retry to load the real job page.",
    );
  }

  if (isAuthUrl(target.toString())) {
    const referer = req.headers.get("referer") || "";
    let jobPage: string | undefined;
    try {
      const ref = new URL(referer);
      const nested = ref.searchParams.get("url");
      if (nested && !isAuthUrl(nested)) jobPage = unwrapProxyUrl(nested);
    } catch {
      /* ignore */
    }
    return authBlockedPage(proxyOrigin, target.toString(), jobPage);
  }

  try {
    const upstream = await fetch(target.toString(), {
      headers: {
        "User-Agent":
          "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
        Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.9",
      },
      redirect: "follow",
      cache: "no-store",
    });

    const contentType = upstream.headers.get("content-type") || "text/html";
    const finalUrl = unwrapProxyUrl(upstream.url || target.toString());

    if (!upstream.ok) {
      return fallbackPage(
        proxyOrigin,
        finalUrl,
        `The site returned ${upstream.status}, so JobRight can’t load it inside the app.`,
      );
    }

    if (!contentType.includes("text/html") && !contentType.includes("application/xhtml")) {
      return fallbackPage(
        proxyOrigin,
        finalUrl,
        "This application page isn’t plain HTML, so it can’t be shown inside JobRight.",
      );
    }

    const body = rewriteHtml(await upstream.text(), finalUrl, proxyOrigin);

    return new NextResponse(body, {
      status: 200,
      headers: {
        "Content-Type": "text/html; charset=utf-8",
        "Cache-Control": "no-store",
      },
    });
  } catch {
    return fallbackPage(
      proxyOrigin,
      target.toString(),
      "JobRight could not fetch this page (network or site block).",
    );
  }
}
