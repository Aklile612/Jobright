/** Prefer the board's actual application entrypoint when the listing URL isn't a form. */
export function resolveApplyUrl(sourceUrl: string): string {
  if (!sourceUrl) return sourceUrl;
  try {
    const u = new URL(sourceUrl);
    const host = u.hostname.toLowerCase();

    if (host.includes("arbeitnow.com")) {
      if (!u.pathname.endsWith("/apply")) {
        u.pathname = u.pathname.replace(/\/?$/, "") + "/apply";
      }
      return u.toString();
    }

    if (host.includes("jobright.ai") && /\/jobs\/info\//i.test(u.pathname)) {
      return u.toString();
    }

    return u.toString();
  } catch {
    return sourceUrl;
  }
}
