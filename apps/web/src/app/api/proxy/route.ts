import { NextRequest, NextResponse } from "next/server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

function fallbackPage(target: string, reason: string) {
  const safe = target.replace(/"/g, "&quot;");
  const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
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
    <h1 style="margin:0 0 8px;font-size:1.4rem">This board can’t be embedded</h1>
    <p>${reason}</p>
    <p>Use Copy on the left panel, then open the original application page and paste.</p>
    <p><code>${safe}</code></p>
    <a href="${safe}" target="_blank" rel="noreferrer">Open original application</a>
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

/** Injected into proxied ATS pages — React-aware + label/placeholder matching. */
const AUTOFILL_BRIDGE = `<script>
(function(){
  if (window.__jobrightAutofillReady) return;
  window.__jobrightAutofillReady = true;
  var lastPayload = null;
  var lastFilled = 0;

  function setNativeValue(el, value) {
    if (value == null || value === '') return false;
    var proto = el.tagName === 'TEXTAREA'
      ? window.HTMLTextAreaElement.prototype
      : window.HTMLInputElement.prototype;
    var desc = Object.getOwnPropertyDescriptor(proto, 'value');
    el.focus();
    if (desc && desc.set) desc.set.call(el, value);
    else el.value = value;
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
    try {
      el.dispatchEvent(new InputEvent('input', { bubbles: true, data: String(value), inputType: 'insertText' }));
    } catch (e) {}
    return true;
  }

  function visible(el) {
    if (!el || el.disabled || el.readOnly) return false;
    var s = window.getComputedStyle(el);
    if (s.display === 'none' || s.visibility === 'hidden' || s.opacity === '0') return false;
    var r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0;
  }

  function labelText(el) {
    var bits = [];
    if (el.id) {
      var byFor = document.querySelector('label[for=\"' + CSS.escape(el.id) + '\"]');
      if (byFor) bits.push(byFor.textContent || '');
    }
    var parentLabel = el.closest('label');
    if (parentLabel) bits.push(parentLabel.textContent || '');
    var labelled = el.getAttribute('aria-labelledby');
    if (labelled) {
      labelled.split(/\\s+/).forEach(function(id){
        var n = document.getElementById(id);
        if (n) bits.push(n.textContent || '');
      });
    }
    bits.push(el.getAttribute('aria-label') || '');
    bits.push(el.getAttribute('placeholder') || '');
    bits.push(el.getAttribute('name') || '');
    bits.push(el.id || '');
    bits.push(el.getAttribute('autocomplete') || '');
    return bits.join(' ').toLowerCase();
  }

  function pick(data, keys) {
    for (var i = 0; i < keys.length; i++) {
      var v = data[keys[i]];
      if (v) return String(v);
    }
    return '';
  }

  function matchValue(el, data) {
    var t = labelText(el);
    var type = (el.getAttribute('type') || el.type || '').toLowerCase();
    if (type === 'email' || /e-?mail|courriel/.test(t)) return pick(data, ['email']);
    if (type === 'tel' || /phone|mobile|telefon|tel\\b|handy/.test(t)) return pick(data, ['phone']);
    if (/linkedin/.test(t)) return pick(data, ['linkedin']);
    if (/github/.test(t)) return pick(data, ['github']);
    if (/portfolio|website|homepage|personal.?site|url/.test(t)) return pick(data, ['website']);
    if (/skill|kompeten|technologies|tech stack/.test(t)) return pick(data, ['skills']);
    if (/cover|anschreiben|motivation|message|why|letter|nachricht/.test(t)) return pick(data, ['coverLetter', 'cover_letter']);
    if (/first.?name|vorname|given.?name/.test(t)) {
      var full = pick(data, ['name']);
      return full ? full.split(/\\s+/)[0] : '';
    }
    if (/last.?name|nachname|family.?name|surname/.test(t)) {
      var full2 = pick(data, ['name']);
      if (!full2) return '';
      var parts = full2.trim().split(/\\s+/);
      return parts.length > 1 ? parts.slice(1).join(' ') : '';
    }
    if (/full.?name|your.?name|candidate.?name|name|vollst/.test(t) && type !== 'hidden') {
      return pick(data, ['name']);
    }
    if (/location|city|standort|ort|address|adresse/.test(t)) return pick(data, ['location']);
    if (/headline|title|job.?title|position/.test(t)) return pick(data, ['headline']);
    return '';
  }

  function fillAll(data) {
    if (!data) return 0;
    var count = 0;
    var nodes = document.querySelectorAll('input, textarea');
    nodes.forEach(function(el){
      var type = (el.getAttribute('type') || el.type || '').toLowerCase();
      if (['hidden','submit','button','checkbox','radio','file','password','image'].indexOf(type) >= 0) return;
      if (!visible(el) && type !== 'email') return;
      var value = matchValue(el, data);
      if (!value) return;
      if (el.value && el.value.trim() === value.trim()) { count++; return; }
      if (setNativeValue(el, value)) count++;
    });
    lastFilled = count;
    try {
      window.parent.postMessage({ type: 'jobright-autofill-result', filled: count }, '*');
    } catch (e) {}
    return count;
  }

  function scheduleFills(data) {
    lastPayload = data || lastPayload;
    if (!lastPayload) return;
    fillAll(lastPayload);
    [300, 800, 1600, 3200, 5000].forEach(function(ms){
      setTimeout(function(){ fillAll(lastPayload); }, ms);
    });
  }

  window.addEventListener('message', function(event){
    if (!event.data || event.data.type !== 'jobright-autofill') return;
    scheduleFills(event.data.payload || {});
  });

  // Ask parent for profile once the SPA shell is up.
  function requestProfile() {
    try {
      window.parent.postMessage({ type: 'jobright-autofill-ready' }, '*');
    } catch (e) {}
  }
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', requestProfile);
  } else {
    requestProfile();
  }
  setTimeout(requestProfile, 1000);

  var obs = new MutationObserver(function(){
    if (lastPayload) fillAll(lastPayload);
  });
  obs.observe(document.documentElement, { childList: true, subtree: true });
})();
</script>`;

export async function GET(req: NextRequest) {
  const raw = req.nextUrl.searchParams.get("url");
  if (!raw) {
    return NextResponse.json({ error: "url is required" }, { status: 400 });
  }

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

  if (/\/remote-jobs\/[a-z0-9-]+\/?$/i.test(target.pathname) && !/-\d+$/.test(target.pathname)) {
    return fallbackPage(
      target.toString(),
      "That link is a job board listing, not a single application page.",
    );
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
    const finalUrl = upstream.url || target.toString();

    if (!upstream.ok) {
      return fallbackPage(
        finalUrl,
        `The site returned ${upstream.status}, so JobRight can’t load it inside the app.`,
      );
    }

    if (!contentType.includes("text/html") && !contentType.includes("application/xhtml")) {
      return fallbackPage(
        finalUrl,
        "This application page isn’t plain HTML, so it can’t be shown inside JobRight.",
      );
    }

    let body = await upstream.text();
    const final = new URL(finalUrl);
    const base = `${final.protocol}//${final.host}${final.pathname.replace(/[^/]*$/, "")}`;

    // Drop frame-busting / CSP that break embedded apply.
    body = body
      .replace(/<meta[^>]+http-equiv=["']?Content-Security-Policy["']?[^>]*>/gi, "")
      .replace(/<meta[^>]+http-equiv=["']?X-Frame-Options["']?[^>]*>/gi, "");

    if (!/<base\s/i.test(body)) {
      body = body.replace(/<head([^>]*)>/i, `<head$1><base href="${base}">`);
    }

    body = /<\/body>/i.test(body)
      ? body.replace(/<\/body>/i, `${AUTOFILL_BRIDGE}</body>`)
      : body + AUTOFILL_BRIDGE;

    return new NextResponse(body, {
      status: 200,
      headers: {
        "Content-Type": "text/html; charset=utf-8",
        "Cache-Control": "no-store",
      },
    });
  } catch {
    return fallbackPage(
      target.toString(),
      "JobRight could not fetch this page (network or site block).",
    );
  }
}
