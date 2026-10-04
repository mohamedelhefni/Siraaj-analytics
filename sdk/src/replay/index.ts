import { record } from '@rrweb/record';
import type { AnalyticsCore } from '../core/analytics';

export interface ReplayOptions {
  sampleRate?: number; // Share of sessions to record, 0.0 to 1.0 (default 1)
  flushInterval?: number; // Milliseconds between uploads (default 5000)
}

/**
 * Records the page with rrweb and uploads gzip NDJSON chunks to /api/replay.
 * Inputs are always masked; add `siraaj-block` to hide an element and
 * `siraaj-mask` to mask its text. Returns a function that stops recording.
 */
export function startReplay(analytics: AnalyticsCore, options: ReplayOptions = {}): () => void {
  const target = analytics.replayTarget();
  if (!target || typeof CompressionStream === 'undefined' || Math.random() >= (options.sampleRate ?? 1)) {
    return () => {};
  }

  // Each page load is its own recording; recordings from one visit share the session id.
  const recording = typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const endpoint = `${target.apiUrl}/api/replay?session=${encodeURIComponent(target.sessionId)}&recording=${recording}`;
  let events: unknown[] = [];
  let uploads = Promise.resolve(); // chained so chunks reach the server in order

  const upload = (keepalive: boolean) => {
    if (!events.length) return;
    const ndjson = events.map(event => JSON.stringify(event)).join('\n') + '\n';
    events = [];
    uploads = uploads
      .then(() => new Response(new Blob([ndjson]).stream().pipeThrough(new CompressionStream('gzip'))).blob())
      .then(body => fetch(`${endpoint}&url=${encodeURIComponent(location.href)}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/octet-stream', 'X-Siraaj-Token': target.trackingToken },
        body,
        keepalive: keepalive && body.size < 60000, // browsers cap keepalive bodies at 64KB
      }))
      // A full chunk is lost on any failure, so stop rather than keep posting to a server that is down.
      .then(response => { if (response.status === 413 || response.status === 429 || response.status >= 500) halt(); })
      .catch(halt);
  };

  const stopRecording = record({
    emit: event => { events.push(event); },
    maskAllInputs: true,
    blockClass: 'siraaj-block',
    maskTextClass: 'siraaj-mask',
    sampling: { mousemove: 50, scroll: 150 },
  });
  const timer = setInterval(() => upload(false), options.flushInterval ?? 5000);
  // ponytail: the last few seconds can be lost if the page dies mid-compression.
  const onHide = () => { if (document.visibilityState === 'hidden') upload(true); };
  document.addEventListener('visibilitychange', onHide);

  function halt() {
    stopRecording?.();
    clearInterval(timer);
    document.removeEventListener('visibilitychange', onHide);
  }
  function stop() {
    halt();
    upload(true);
  }
  return stop;
}
