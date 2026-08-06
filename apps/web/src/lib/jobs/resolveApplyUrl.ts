/** Prefer the board's actual application entrypoint when the listing URL isn't a form. */
export function resolveApplyUrl(sourceUrl: string): string {
  if (!sourceUrl) return sourceUrl;
  let url = sourceUrl;
  // Unwrap accidental nested proxy URLs (base-href bug)
  for (let i = 0; i < 5; i++) {
    try {
      const u = new URL(url);
      if (u.pathname.includes("/api/proxy")) {
        const inner = u.searchParams.get("url");
        if (inner) {
          url = inner;
          continue;
        }
      }
    } catch {
      break;
    }
    break;
  }
  try {
    const u = new URL(url);
    const host = u.hostname.toLowerCase();

    if (host.includes("arbeitnow.com")) {
      if (!u.pathname.endsWith("/apply")) {
        u.pathname = u.pathname.replace(/\/?$/, "") + "/apply";
      }
      return u.toString();
    }

    return u.toString();
  } catch {
    return url;
  }
}
