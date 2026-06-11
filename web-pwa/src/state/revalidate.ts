import { useEffect, useRef } from "react";

type Options = {
  // SSE event names that should trigger an immediate revalidate.
  sseEvents?: string[];
  // Polling interval in ms when the tab is visible (fallback if SSE is dropped
  // by a proxy like Cloudflare). Default 20s. Set 0 to disable polling.
  pollMs?: number;
  // EventSource URL. When provided, the hook also listens to SSE.
  eventsUrl?: string;
};

/**
 * useRevalidate implements a stale-while-revalidate freshness strategy that is
 * resilient to unreliable realtime transports (mobile tabs suspending, SSE
 * buffered/dropped behind reverse proxies). It revalidates on:
 *  - window focus
 *  - tab becoming visible
 *  - network coming back online
 *  - SSE events (fast path)
 *  - a light interval poll (fallback) while the tab is visible
 *
 * `refresh` is always called through a ref so consumers don't need to memoize.
 */
export function useRevalidate(refresh: () => void, opts: Options = {}) {
  const refreshRef = useRef(refresh);
  refreshRef.current = refresh;

  const { sseEvents = [], pollMs = 20000, eventsUrl } = opts;

  useEffect(() => {
    let lastRun = 0;
    // Coalesce bursts (e.g. focus + visibility firing together).
    const run = () => {
      const now = Date.now();
      if (now - lastRun < 400) return;
      lastRun = now;
      refreshRef.current();
    };

    const onFocus = () => run();
    const onVisible = () => {
      if (document.visibilityState === "visible") run();
    };
    const onOnline = () => run();

    window.addEventListener("focus", onFocus);
    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("online", onOnline);

    let stream: EventSource | null = null;
    if (eventsUrl && sseEvents.length > 0) {
      try {
        stream = new EventSource(eventsUrl, { withCredentials: true });
        sseEvents.forEach((name) => stream?.addEventListener(name, run));
      } catch {
        stream = null;
      }
    }

    let timer: number | null = null;
    if (pollMs > 0) {
      timer = window.setInterval(() => {
        if (document.visibilityState === "visible") run();
      }, pollMs);
    }

    return () => {
      window.removeEventListener("focus", onFocus);
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("online", onOnline);
      if (stream) stream.close();
      if (timer !== null) window.clearInterval(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eventsUrl, pollMs, JSON.stringify(sseEvents)]);
}
