// Core Analytics Types
export interface AnalyticsConfig {
  apiUrl?: string;
  projectId?: string;
  trackingToken?: string; // Project-scoped write token created in the dashboard
  autoTrack?: boolean;
  debug?: boolean;
  bufferSize?: number;
  flushInterval?: number;
  timeout?: number;
  maxRetries?: number;
  useBeacon?: boolean; // Use navigator.sendBeacon for reliability
  sampling?: number; // Sample rate 0.0 to 1.0 (percentage of events to track)
  maxQueueSize?: number; // Maximum failed events to queue
  enablePerformanceTracking?: boolean; // Track Web Vitals
  respectDoNotTrack?: boolean; // Respect DNT header
}

export interface EventData {
  event_name: string;
  user_id: string;
  session_id: string;
  session_duration: number;
  url: string;
  referrer: string;
  user_agent: string;
  timestamp: string;
  browser: string;
  os: string;
  device: string;
  project_id: string;
  channel: string;
}

export interface SessionInfo {
  sessionId: string;
  userId: string;
}

export interface DeviceInfo {
  browser: string;
  os: string;
  device: string;
  screen_width?: number;
  screen_height?: number;
  viewport_width?: number;
  viewport_height?: number;
  language?: string;
  timezone?: string;
}

export interface QueuedRequest {
  event: EventData;
  retries: number;
  nextRetry: number; // Timestamp
}

export interface Analytics {
  track(eventName: string, properties?: Record<string, any>): void;
  pageView(url?: string | null): void;
  identify(userId: string, traits?: Record<string, any>): void;
  trackClick(elementId: string, properties?: Record<string, any>): void;
  trackForm(formId: string, properties?: Record<string, any>): void;
  trackError(error: Error | string, context?: Record<string, any>): void;
  setUserProperties(properties: Record<string, any>): void;
  flush(async?: boolean): Promise<any>;
  destroy(): void;
}
