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
    <p>Use the autofill panel on the left to copy your profile fields, then open the original application page.</p>
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

  // Category listing pages often 404 / aren't apply forms.
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
    if (!upstream.ok) {
      return fallbackPage(
        target.toString(),
        `The site returned ${upstream.status}, so JobRight can’t load it inside the app.`,
      );
    }

    // Many ATS sites send HTML that refuses framing / is JS-only.
    if (!contentType.includes("text/html") && !contentType.includes("application/xhtml")) {
      return fallbackPage(
        target.toString(),
        "This application page isn’t plain HTML, so it can’t be shown inside JobRight.",
      );
    }

    let body = await upstream.text();
    const base = `${target.protocol}//${target.host}/`;
    if (!/<base\s/i.test(body)) {
      body = body.replace(/<head([^>]*)>/i, `<head$1><base href="${base}">`);
    }

    const bridge = `<script>
(function(){
  window.addEventListener('message', function(event){
    if (!event.data || event.data.type !== 'jobright-autofill') return;
    var data = event.data.payload || {};
    var map = [
      ['input[name*=email i],input[type=email],input[autocomplete=email]', data.email],
      ['input[name*=name i],input[autocomplete=name],input[name*=full_name i]', data.name],
      ['input[name*=phone i],input[type=tel],input[autocomplete=tel]', data.phone],
      ['input[name*=linkedin i]', data.linkedin],
      ['input[name*=github i]', data.github],
      ['input[name*=portfolio i],input[name*=website i]', data.website],
      ['textarea[name*=cover i],textarea[id*=cover i]', data.coverLetter]
    ];
    map.forEach(function(pair){
      if (!pair[1]) return;
      document.querySelectorAll(pair[0]).forEach(function(el){
        el.focus();
        el.value = pair[1];
        el.dispatchEvent(new Event('input', { bubbles: true }));
        el.dispatchEvent(new Event('change', { bubbles: true }));
      });
    });
  });
})();
</script>`;
    body = /<\/body>/i.test(body)
      ? body.replace(/<\/body>/i, `${bridge}</body>`)
      : body + bridge;

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
