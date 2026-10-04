import type { AnalyticsConfig, EventData, DeviceInfo, QueuedRequest, Survey } from './types';

/**
 * Siraaj Analytics SDK
 * Production-grade, lightweight client-side analytics tracking library
 * Bundle size: < 7KB gzipped
 */
class AnalyticsCore {
    private config: Required<AnalyticsConfig>;
    private buffer: EventData[] = [];
    private failedQueue: QueuedRequest[] = [];
    private sessionId: string | null = null;
    private userId: string | null = null;
    private flushTimer: number | null = null;
    private retryTimer: number | null = null;
    // While the server is failing, nothing is sent before backoffUntil; the delay doubles per failure.
    private backoffUntil = 0;
    private backoffDelay = 0;
    private sessionStartTime: number | null = null;
    private isDestroyed: boolean = false;
    private initialized: boolean = false;
    private pendingRequests: Set<Promise<any>> = new Set();

    // Cached values for performance
    private cachedDeviceInfo: DeviceInfo | null = null;
    private autoTrackingSetup: boolean = false;
    private clickHandler: ((e: MouseEvent) => void) | null = null;
    private submitHandler: ((e: Event) => void) | null = null;
    private errorHandler: ((e: ErrorEvent) => void) | null = null;
    private rejectionHandler: ((e: PromiseRejectionEvent) => void) | null = null;
    private visibilityHandler: (() => void) | null = null;
    private beforeUnloadHandler: (() => void) | null = null;
    private surveys?: Survey[];
    private surveyEvents?: Set<string>; // events seen while surveys load

    // Constants
    private readonly SESSION_TIMEOUT = 30 * 60 * 1000;
    private readonly MAX_BUFFER_SIZE = 100;
    private readonly MAX_FAILED_QUEUE_SIZE = 50;
    private readonly MAX_STRING_LENGTH = 2048;
    private readonly RETRY_BASE_DELAY = 1000;
    private readonly RETRY_MAX_DELAY = 32000;
    private readonly STORAGE_PREFIX = 'siraaj_analytics_';

    constructor(config: AnalyticsConfig = {}) {
        this.config = {
            apiUrl: config.apiUrl || 'http://localhost:8080',
            projectId: config.projectId || 'default',
            trackingToken: config.trackingToken || '',
            autoTrack: config.autoTrack !== false,
            bufferSize: Math.min(config.bufferSize || 10, this.MAX_BUFFER_SIZE),
            flushInterval: config.flushInterval || 30000,
            debug: config.debug || false,
            timeout: config.timeout || 10000,
            maxRetries: config.maxRetries || 3,
            useBeacon: config.useBeacon !== false,
            sampling: config.sampling || 1.0,
            maxQueueSize: config.maxQueueSize || this.MAX_FAILED_QUEUE_SIZE,
            enablePerformanceTracking: config.enablePerformanceTracking || false,
            respectDoNotTrack: config.respectDoNotTrack !== false,
            surveys: config.surveys !== false,
        };

        if (typeof window === 'undefined') {
            return;
        }

        if (this.config.respectDoNotTrack && this.isDNTEnabled()) {
            this.log('Do Not Track enabled, analytics disabled');
            this.isDestroyed = true;
            return;
        }

        if (this.config.apiUrl.startsWith('http://') && !this.config.apiUrl.includes('localhost')) {
            this.warn('Use HTTPS in production');
        }

        if (Math.random() > this.config.sampling) {
            this.log('Not sampled');
            this.isDestroyed = true;
            return;
        }

        this.initialized = true;
        this.log('Initialized', this.config);
    }

    init(config: AnalyticsConfig): void {
        if (typeof window === 'undefined') return;
        if (this.isDestroyed && !this.isDNTEnabled()) {
            this.isDestroyed = false;
        }

        Object.assign(this.config, config);

        if (!this.sessionId) {
            this.sessionId = this.getSessionId();
            this.userId = this.getUserId();
            this.sessionStartTime = Date.now();
        }

        if (this.config.autoTrack && !this.autoTrackingSetup) {
            this.setupAutoTracking();
        }

        if (this.config.flushInterval > 0 && !this.flushTimer) {
            this.startAutoFlush();
        }

        if (!this.beforeUnloadHandler) {
            this.beforeUnloadHandler = () => this.handleBeforeUnload();
            window.addEventListener('beforeunload', this.beforeUnloadHandler);
        }

        if (this.failedQueue.length > 0) {
            this.processFailedQueue();
        }

        if (this.config.surveys && this.config.trackingToken && !this.surveyEvents) {
            this.loadSurveys();
        }

        this.initialized = true;
        this.log('(Re)initialized');
    }

    track(eventName: string, properties: Record<string, any> = {}): void {
        if (!this.canTrack()) return;

        if (!this.initialized) {
            this.init({});
        }

        this.ensureActiveSession();

        eventName = this.sanitizeString(eventName);
        properties = this.sanitizeProperties(properties);

        const deviceInfo = this.getDeviceInfo();
        const channel = this.detectChannel(document.referrer, window.location.href);

        const event: EventData = {
            event_name: eventName,
            user_id: this.userId || this.getUserId(),
            session_id: this.sessionId || this.getSessionId(),
            session_duration: Math.floor((Date.now() - (this.sessionStartTime || Date.now())) / 1000),
            url: window.location.href,
            referrer: document.referrer,
            user_agent: navigator.userAgent,
            timestamp: new Date().toISOString(),
            browser: deviceInfo.browser,
            os: deviceInfo.os,
            device: deviceInfo.device,
            project_id: this.config.projectId,
            channel: channel,
            ...properties,
        };

        this.log('Event:', eventName);
        this.addToBuffer(event);
        if (this.surveys) this.maybeShowSurvey(eventName);
        else this.surveyEvents?.add(eventName);
    }

    pageView(url: string | null = null, properties: Record<string, any> = {}): void {
        if (!this.canTrack()) return;

        this.track('page_view', {
            path: url || window.location.pathname,
            title: document.title,
            search: window.location.search,
            hash: window.location.hash,
            ...properties,
        });
        this.flush(false)
    }

    identify(userId: string, traits: Record<string, any> = {}): void {
        if (!this.canTrack()) return;

        userId = this.sanitizeString(userId);
        this.userId = userId;

        if (typeof window !== 'undefined') {
            this.setUserIdInStorage(userId);
        }

        this.track('identify', this.sanitizeProperties(traits));
    }

    trackClick(elementId: string, properties: Record<string, any> = {}): void {
        this.track('click', {
            element: this.sanitizeString(elementId),
            ...this.sanitizeProperties(properties),
        });
    }

    trackForm(formId: string, properties: Record<string, any> = {}): void {
        this.track('form_submit', {
            form_id: this.sanitizeString(formId),
            ...this.sanitizeProperties(properties),
        });
    }

    trackError(error: Error | string, context: Record<string, any> = {}): void {
        const errorData = typeof error === 'string'
            ? { message: this.sanitizeString(error) }
            : {
                message: this.sanitizeString(error.message),
                stack: this.sanitizeString(error.stack || ''),
                name: error.name,
            };

        this.track('error', {
            ...errorData,
            ...this.sanitizeProperties(context),
        });
    }

    setUserProperties(properties: Record<string, any>): void {
        this.track('user_properties_updated', this.sanitizeProperties(properties));
    }

    reset(): void {
        if (typeof window === 'undefined') return;

        this.removeStorage('user_id');
        this.removeStorage('session_id');
        this.removeStorage('session_start');

        this.userId = this.generateId();
        this.sessionId = this.generateId();
        this.sessionStartTime = Date.now();

        this.log('Reset');
    }

    async flush(useBeacon: boolean = false): Promise<void> {
        if (this.buffer.length === 0) return;
        if (!this.canTrack()) return;

        const events = [...this.buffer];
        this.buffer = [];

        this.log('Flushing', events.length, 'events');

        if (Date.now() < this.backoffUntil) {
            events.forEach(event => this.queueFailedEvent(event));
            this.scheduleRetry();
            return;
        }

        if (useBeacon && this.config.useBeacon && typeof navigator !== 'undefined' && 'sendBeacon' in navigator) {
            this.sendBatch(events, true);
        } else {
            const promise = this.sendBatch(events, false).catch((err) => {
                this.log('Batch error:', err);
                if (this.isRetryableError(err)) {
                    for (const event of events) {
                        this.queueFailedEvent(event);
                    }
                    this.scheduleRetry();
                } else {
                    this.log('Non-retryable error, dropping batch');
                }
            });
            this.pendingRequests.add(promise);

            try {
                await promise;
            } finally {
                this.pendingRequests.delete(promise);
            }
        }
    }

    async destroy(): Promise<void> {
        if (this.isDestroyed) return;

        this.log('Destroying');
        this.isDestroyed = true;

        if (this.flushTimer) {
            clearInterval(this.flushTimer);
            this.flushTimer = null;
        }

        if (this.retryTimer) {
            clearTimeout(this.retryTimer);
            this.retryTimer = null;
        }

        if (typeof window !== 'undefined') {
            if (this.clickHandler) {
                document.removeEventListener('click', this.clickHandler, { capture: true } as any);
            }
            if (this.submitHandler) {
                document.removeEventListener('submit', this.submitHandler, { capture: true } as any);
            }
            if (this.errorHandler) {
                window.removeEventListener('error', this.errorHandler);
            }
            if (this.rejectionHandler) {
                window.removeEventListener('unhandledrejection', this.rejectionHandler);
            }
            if (this.visibilityHandler) {
                document.removeEventListener('visibilitychange', this.visibilityHandler);
            }
            if (this.beforeUnloadHandler) {
                window.removeEventListener('beforeunload', this.beforeUnloadHandler);
            }
        }

        if (this.pendingRequests.size > 0) {
            try {
                await Promise.race([
                    Promise.allSettled(this.pendingRequests),
                    new Promise(resolve => setTimeout(resolve, this.config.timeout)),
                ]);
            } catch (err) {
                this.log('Error waiting for requests:', err);
            }
        }

        try {
            await this.flush(true);
        } catch (err) {
            this.log('Error during flush:', err);
        }
    }

    /** Where the optional replay bundle sends recordings; null when tracking is off or no token is set. */
    replayTarget(): { apiUrl: string; trackingToken: string; sessionId: string } | null {
        if (!this.canTrack() || !this.config.trackingToken) return null;
        if (!this.initialized) this.init({});
        this.ensureActiveSession();
        return { apiUrl: this.config.apiUrl, trackingToken: this.config.trackingToken, sessionId: this.sessionId! };
    }

    private canTrack(): boolean {
        return typeof window !== 'undefined' && !this.isDestroyed;
    }

    private addToBuffer(event: EventData): void {
        this.buffer.push(event);

        if (this.buffer.length >= this.config.bufferSize) {
            setTimeout(() => this.flush(), 0);
        }
    }

    // Performs the network request only — throws on failure, no internal queuing.
    // Callers are responsible for catching and handling retry/queuing logic.
    private async sendBatch(events: EventData[], useBeacon: boolean): Promise<void> {
        if (events.length === 0) return;

        const endpoint = `${this.config.apiUrl}/api/track/batch`;

        // sendBeacon cannot attach the tracking-token header, so authenticated requests use fetch keepalive.
        if (useBeacon && !this.config.trackingToken && typeof navigator !== 'undefined' && 'sendBeacon' in navigator) {
            for (const event of events) {
                try {
                    const blob = new Blob([JSON.stringify(event)], { type: 'application/json' });
                    const success = navigator.sendBeacon(`${this.config.apiUrl}/api/track`, blob);
                    if (!success) {
                        this.queueFailedEvent(event);
                    }
                } catch (err) {
                    this.queueFailedEvent(event);
                }
            }
            return;
        }

        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

        let response: Response;
        try {
            response = await this.backoffFetch(endpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Siraaj-Token': this.config.trackingToken,
                },
                body: JSON.stringify({ events }),
                keepalive: true,
                signal: controller.signal,
            });
        } finally {
            clearTimeout(timeoutId);
        }

        if (!response!.ok) {
            throw new Error(`HTTP ${response!.status}`);
        }

        this.log('Batch sent:', events.length);
    }

    // fetch that backs off on network errors, 5xx and 429, and resets once the server answers.
    private async backoffFetch(url: string, init: RequestInit): Promise<Response> {
        try {
            const response = await fetch(url, init);
            if (response.status >= 500 || response.status === 429) {
                this.backOff();
            } else {
                this.backoffDelay = 0;
                this.backoffUntil = 0;
            }
            return response;
        } catch (err) {
            this.backOff();
            throw err;
        }
    }

    private backOff(): void {
        this.backoffDelay = Math.min(Math.max(this.backoffDelay * 2, this.RETRY_BASE_DELAY), this.RETRY_MAX_DELAY);
        this.backoffUntil = Date.now() + this.backoffDelay;
        this.log('Server unavailable, pausing requests for', this.backoffDelay, 'ms');
    }

    private queueFailedEvent(event: EventData): void {
        if (this.failedQueue.length >= this.config.maxQueueSize) {
            this.log('Queue full, dropping event');
            this.failedQueue.shift();
        }

        this.failedQueue.push({
            event,
            retries: 0,
            nextRetry: Date.now() + this.RETRY_BASE_DELAY,
        });
    }

    private scheduleRetry(): void {
        if (this.retryTimer || this.failedQueue.length === 0) return;

        const now = Date.now();
        const nextTime = Math.max(this.backoffUntil, Math.min(...this.failedQueue.map(item => item.nextRetry)));
        const delay = Math.max(nextTime - now, 100);

        this.retryTimer = window.setTimeout(() => {
            this.retryTimer = null;
            this.processFailedQueue();
        }, delay);
    }

    private async processFailedQueue(): Promise<void> {
        if (this.failedQueue.length === 0) return;
        if (Date.now() < this.backoffUntil) {
            this.scheduleRetry();
            return;
        }

        const now = Date.now();
        const readyToRetry: QueuedRequest[] = [];
        const notReady: QueuedRequest[] = [];

        for (const item of this.failedQueue) {
            if (item.nextRetry <= now) {
                readyToRetry.push(item);
            } else {
                notReady.push(item);
            }
        }

        this.failedQueue = notReady;

        if (readyToRetry.length === 0) {
            this.scheduleRetry();
            return;
        }

        const eligible = readyToRetry.filter(item => {
            if (item.retries >= this.config.maxRetries) {
                this.log('Max retries reached, dropping:', item.event.event_name);
                return false;
            }
            return true;
        });

        if (eligible.length > 0) {
            try {
                await this.sendBatch(eligible.map(item => item.event), false);
            } catch (err) {
                if (this.isRetryableError(err)) {
                    for (const item of eligible) {
                        const delay = Math.min(
                            this.RETRY_BASE_DELAY * Math.pow(2, item.retries),
                            this.RETRY_MAX_DELAY
                        );
                        this.failedQueue.push({
                            event: item.event,
                            retries: item.retries + 1,
                            nextRetry: Date.now() + delay,
                        });
                    }
                } else {
                    this.log('Non-retryable error, dropping retry batch');
                }
            }
        }

        if (this.failedQueue.length > 0) {
            this.scheduleRetry();
        }
    }

    private setupAutoTracking(): void {
        if (typeof window === 'undefined' || this.autoTrackingSetup) return;

        this.autoTrackingSetup = true;

        if (document.readyState === 'complete') {
            this.pageView();
        } else {
            window.addEventListener('load', () => this.pageView(), { once: true, passive: true });
        }

        this.visibilityHandler = () => {
            this.track(document.hidden ? 'page_hidden' : 'page_visible');
        };
        document.addEventListener('visibilitychange', this.visibilityHandler, { passive: true });

        this.clickHandler = (e: MouseEvent) => {
            const link = (e.target as HTMLElement).closest('a');
            if (link) {
                const href = link.getAttribute('href');
                const text = link.textContent?.trim().substring(0, 100);

                this.track('link_clicked', {
                    url: href ? this.sanitizeString(href) : null,
                    text: text ? this.sanitizeString(text) : null,
                    external: href ? !href.startsWith(window.location.origin) : false,
                });
            }
        };
        document.addEventListener('click', this.clickHandler, { capture: true, passive: true });

        this.submitHandler = (e: Event) => {
            const form = e.target as HTMLFormElement;
            if (form && form.tagName === 'FORM') {
                const formData: Record<string, any> = {};

                if (form.id) {
                    formData.form_id = this.sanitizeString(form.id);
                }

                if (form.action) {
                    formData.action = this.sanitizeString(form.action);
                }

                this.track('form_submit', formData);
            }
        };
        document.addEventListener('submit', this.submitHandler, { capture: true, passive: true });

        this.errorHandler = (e: ErrorEvent) => {
            this.trackError(e.error || e.message, {
                filename: e.filename ? this.sanitizeString(e.filename) : null,
                lineno: e.lineno,
                colno: e.colno,
            });
        };
        window.addEventListener('error', this.errorHandler);

        this.rejectionHandler = (e: PromiseRejectionEvent) => {
            const reason = e.reason;
            const error = reason instanceof Error
                ? reason
                : typeof reason === 'string'
                    ? reason
                    : String(reason ?? 'Unknown rejection');
            this.trackError(error, { type: 'unhandled_promise_rejection' });
        };
        window.addEventListener('unhandledrejection', this.rejectionHandler);

        if (this.config.enablePerformanceTracking) {
            this.trackWebVitals();
        }

        this.log('Auto-tracking enabled');
    }

    private trackWebVitals(): void {
        if (typeof window === 'undefined' || !('PerformanceObserver' in window)) {
            return;
        }

        try {
            const lcpObserver = new PerformanceObserver((list) => {
                const entries = list.getEntries();
                const lastEntry = entries[entries.length - 1] as any;

                this.track('web_vital_lcp', {
                    value: lastEntry.renderTime || lastEntry.loadTime,
                    metric: 'LCP',
                });
            });
            lcpObserver.observe({ entryTypes: ['largest-contentful-paint'] });

            const fidObserver = new PerformanceObserver((list) => {
                const entries = list.getEntries();
                entries.forEach((entry: any) => {
                    this.track('web_vital_fid', {
                        value: entry.processingStart - entry.startTime,
                        metric: 'FID',
                    });
                });
            });
            fidObserver.observe({ entryTypes: ['first-input'] });

            let clsValue = 0;
            const clsObserver = new PerformanceObserver((list) => {
                const entries = list.getEntries();
                entries.forEach((entry: any) => {
                    if (!entry.hadRecentInput) {
                        clsValue += entry.value;
                    }
                });
            });
            clsObserver.observe({ entryTypes: ['layout-shift'] });

            window.addEventListener('visibilitychange', () => {
                if (document.hidden) {
                    this.track('web_vital_cls', {
                        value: clsValue,
                        metric: 'CLS',
                    });
                }
            }, { once: true, passive: true });
        } catch (err) {
            this.log('Web Vitals error:', err);
        }
    }

    private startAutoFlush(): void {
        if (typeof window === 'undefined' || this.flushTimer) return;

        this.flushTimer = window.setInterval(() => {
            if (this.buffer.length > 0) {
                this.flush();
            }
        }, this.config.flushInterval);
    }

    private handleBeforeUnload(): void {
        this.log('Unloading, flushing');

        if (this.buffer.length > 0) {
            this.flush(true);
        }
    }

    private ensureActiveSession(): void {
        if (!this.sessionId || !this.sessionStartTime) {
            this.sessionId = this.getSessionId();
            this.sessionStartTime = Date.now();
            return;
        }

        const sessionAge = Date.now() - this.sessionStartTime;
        if (sessionAge > this.SESSION_TIMEOUT) {
            this.log('Session expired');
            this.sessionId = this.generateId();
            this.sessionStartTime = Date.now();

            if (typeof window !== 'undefined') {
                this.setStorage('session_id', this.sessionId);
                this.setStorage('session_start', this.sessionStartTime.toString());
            }
        }
    }

    private getSessionId(): string {
        if (typeof window === 'undefined') return this.generateId();

        let sessionId = this.getStorage('session_id');
        const sessionStart = this.getStorage('session_start');

        if (!sessionId || !sessionStart || (Date.now() - parseInt(sessionStart)) > this.SESSION_TIMEOUT) {
            sessionId = this.generateId();
            this.setStorage('session_id', sessionId);
            this.setStorage('session_start', Date.now().toString());
            this.sessionStartTime = Date.now();
        } else {
            this.sessionStartTime = parseInt(sessionStart);
        }

        return sessionId;
    }

    private getUserId(): string {
        if (typeof window === 'undefined') return this.generateId();

        let userId = this.getUserIdFromStorage();
        if (!userId) {
            userId = this.generateId();
            this.setUserIdInStorage(userId);
        }
        return userId;
    }

    private generateId(): string {
        if (typeof window !== 'undefined' && window.crypto && window.crypto.randomUUID) {
            return window.crypto.randomUUID();
        }

        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
            const r = (Math.random() * 16) | 0;
            const v = c === 'x' ? r : (r & 0x3) | 0x8;
            return v.toString(16);
        });
    }

    private getDeviceInfo(): DeviceInfo {
        if (this.cachedDeviceInfo) {
            return this.cachedDeviceInfo;
        }

        if (typeof window === 'undefined') {
            return {
                browser: 'Unknown',
                os: 'Unknown',
                device: 'Unknown',
            };
        }

        this.cachedDeviceInfo = {
            browser: this.getBrowser(),
            os: this.getOS(),
            device: this.getDevice(),
            screen_width: window.screen.width,
            screen_height: window.screen.height,
            viewport_width: window.innerWidth,
            viewport_height: window.innerHeight,
            language: navigator.language,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        };

        return this.cachedDeviceInfo;
    }

    private getBrowser(): string {
        if (typeof window === 'undefined') return 'Unknown';

        const ua = navigator.userAgent;

        // Order matters: specific engines before generic Chrome
        if (ua.includes('Edg/') || ua.includes('Edge/')) return 'Edge';
        if (ua.includes('OPR/') || ua.includes('Opera')) return 'Opera';
        if (ua.includes('SamsungBrowser')) return 'Samsung';
        if (ua.includes('Chrome')) return 'Chrome';
        if (ua.includes('Safari')) return 'Safari';
        if (ua.includes('Firefox')) return 'Firefox';
        if (ua.includes('MSIE') || ua.includes('Trident')) return 'IE';

        return 'Unknown';
    }

    private getOS(): string {
        if (typeof window === 'undefined') return 'Unknown';

        const ua = navigator.userAgent;

        // Check Android before Linux (Android UA contains both)
        if (ua.includes('Android')) return 'Android';
        if (ua.includes('iOS') || ua.includes('iPhone') || ua.includes('iPad')) return 'iOS';
        if (ua.includes('Win')) return 'Windows';
        if (ua.includes('Mac')) return 'MacOS';
        if (ua.includes('Linux')) return 'Linux';

        return 'Unknown';
    }

    private getDevice(): string {
        if (typeof window === 'undefined') return 'Desktop';

        const ua = navigator.userAgent;

        if (/(tablet|ipad|playbook|silk)|(android(?!.*mobi))/i.test(ua)) {
            return 'Tablet';
        }
        if (/Mobile|Android|iP(hone|od)|IEMobile|BlackBerry|Kindle|Silk-Accelerated|(hpw|web)OS|Opera M(obi|ini)/.test(ua)) {
            return 'Mobile';
        }

        return 'Desktop';
    }

    private detectChannel(referrer: string, url: string): string {
        let params: URLSearchParams;
        let referrerHostname = '';
        let currentHostname = '';

        try {
            const parsed = new URL(url);
            params = parsed.searchParams;
            currentHostname = parsed.hostname;
        } catch {
            params = new URLSearchParams();
        }

        try {
            referrerHostname = new URL(referrer).hostname;
        } catch {
            // non-parseable referrer treated as empty
        }

        const utmMedium = params.get('utm_medium')?.toLowerCase() ?? '';
        const utmSource = params.get('utm_source')?.toLowerCase() ?? '';
        const referrerLower = referrer.toLowerCase();

        // Priority 1: Paid channels
        if (
            utmMedium === 'cpc' || utmMedium === 'ppc' || utmMedium === 'paid' ||
            utmSource === 'paid' ||
            params.has('gclid') ||
            params.has('fbclid') ||
            referrerLower.includes('googleads') ||
            referrerLower.includes('adwords')
        ) {
            return 'Paid';
        }

        // Priority 2: Direct (no referrer or same domain)
        if (!referrer || referrerHostname === currentHostname) {
            return 'Direct';
        }

        // Priority 3: Social media
        const socialDomains = [
            'facebook.com', 'fb.com', 'twitter.com', 't.co', 'x.com', 'linkedin.com',
            'instagram.com', 'tiktok.com', 'pinterest.com', 'reddit.com',
            'youtube.com', 'snapchat.com', 'whatsapp.com', 'telegram.org',
            'vk.com', 'weibo.com', 'tumblr.com', 'discord.com', 'twitch.tv',
        ];

        for (const domain of socialDomains) {
            if (referrerHostname === domain || referrerHostname.endsWith('.' + domain)) {
                return 'Social';
            }
        }

        // Priority 4: Organic search
        const searchEngines = [
            'google.com', 'bing.com', 'yahoo.com', 'duckduckgo.com',
            'baidu.com', 'yandex.com', 'yandex.ru', 'ask.com',
            'ecosia.org', 'qwant.com', 'startpage.com',
        ];

        for (const engine of searchEngines) {
            if (referrerHostname === engine || referrerHostname.endsWith('.' + engine)) {
                return 'Organic';
            }
        }

        // Priority 5: Referral
        return 'Referral';
    }

    private isRetryableError(err: unknown): boolean {
        if (!(err instanceof Error)) return true;
        // Network errors (no response) are always retryable
        if (err.name === 'AbortError' || err.name === 'TypeError') return true;
        // HTTP errors: only 5xx and 429 (Too Many Requests) are transient
        const match = err.message.match(/^HTTP (\d+)/);
        if (match) {
            const status = parseInt(match[1], 10);
            return status >= 500 || status === 429;
        }
        return true;
    }

    private isDNTEnabled(): boolean {
        if (typeof window === 'undefined') return false;

        const dnt = navigator.doNotTrack || (window as any).doNotTrack || (navigator as any).msDoNotTrack;
        return dnt === '1' || dnt === 'yes';
    }

    private sanitizeString(str: string): string {
        if (typeof str !== 'string') return '';
        return str.substring(0, this.MAX_STRING_LENGTH);
    }

    private sanitizeProperties(props: Record<string, any>): Record<string, any> {
        const sanitized: Record<string, any> = {};

        for (const [key, value] of Object.entries(props)) {
            const sanitizedKey = this.sanitizeString(key);

            if (typeof value === 'string') {
                sanitized[sanitizedKey] = this.sanitizeString(value);
            } else if (typeof value === 'number' || typeof value === 'boolean') {
                sanitized[sanitizedKey] = value;
            } else if (value === null || value === undefined) {
                sanitized[sanitizedKey] = null;
            } else if (typeof value === 'object' && !Array.isArray(value)) {
                sanitized[sanitizedKey] = this.sanitizeProperties(value);
            } else if (Array.isArray(value)) {
                sanitized[sanitizedKey] = value.slice(0, 100).map(item =>
                    typeof item === 'string' ? this.sanitizeString(item) : item
                );
            } else {
                sanitized[sanitizedKey] = this.sanitizeString(String(value));
            }
        }

        return sanitized;
    }

    private getStorage(key: string): string | null {
        try {
            if (typeof window !== 'undefined' && window.sessionStorage) {
                return sessionStorage.getItem(this.STORAGE_PREFIX + key);
            }
        } catch (err) {
            this.log('Storage error:', err);
        }
        return null;
    }

    private setStorage(key: string, value: string): void {
        try {
            if (typeof window !== 'undefined' && window.sessionStorage) {
                sessionStorage.setItem(this.STORAGE_PREFIX + key, value);
            }
        } catch (err) {
            this.log('Storage error:', err);
        }
    }

    private removeStorage(key: string): void {
        try {
            if (typeof window !== 'undefined' && window.sessionStorage) {
                sessionStorage.removeItem(this.STORAGE_PREFIX + key);
            }
        } catch (err) {
            this.log('Storage error:', err);
        }
    }

    private getUserIdFromStorage(): string | null {
        try {
            if (typeof window !== 'undefined' && window.localStorage) {
                return localStorage.getItem(this.STORAGE_PREFIX + 'user_id');
            }
        } catch (err) {
            this.log('Storage error:', err);
        }
        return null;
    }

    private setUserIdInStorage(userId: string): void {
        try {
            if (typeof window !== 'undefined' && window.localStorage) {
                localStorage.setItem(this.STORAGE_PREFIX + 'user_id', userId);
            }
        } catch (err) {
            this.log('Storage error:', err);
        }
    }

    private async loadSurveys(): Promise<void> {
        this.surveyEvents = new Set();
        try {
            const response = await this.backoffFetch(`${this.config.apiUrl}/api/surveys/active`, {
                headers: { 'X-Siraaj-Token': this.config.trackingToken },
            });
            this.surveys = response.ok ? await response.json() : [];
        } catch (err) {
            this.surveys = [];
            this.log('Survey load error:', err);
        }
        this.surveyEvents.forEach(name => this.maybeShowSurvey(name));
    }

    private maybeShowSurvey(eventName: string): void {
        const survey = this.surveys?.find(s => s.trigger_event === eventName);
        if (!survey || typeof document === 'undefined') return;
        // Checks run when the delay ends, so a survey opened meanwhile still blocks this one.
        if (survey.delay_seconds) {
            setTimeout(() => this.showSurveyOnce(survey), survey.delay_seconds * 1000);
        } else {
            this.showSurveyOnce(survey);
        }
    }

    private showSurveyOnce(survey: Survey): void {
        if (document.getElementById('siraaj-survey')) return;
        if (!this.inSurveySample(survey)) return;
        // Stores when it was last shown; older SDKs stored '1', which reads as long ago.
        const seenKey = this.STORAGE_PREFIX + 'survey_' + survey.id;
        try {
            const last = Number(localStorage.getItem(seenKey)) || 0;
            const frequency = survey.frequency || 'once';
            if (last && (frequency === 'once'
                || (frequency === 'until_answered' && localStorage.getItem(seenKey + '_answered'))
                || Date.now() - last < (survey.repeat_days || 0) * 86400000)) return;
            localStorage.setItem(seenKey, String(Date.now()));
        } catch {
            return; // can't remember it was shown, so don't nag on every event
        }
        this.showSurvey(survey);
    }

    // Same visitor, same survey, same answer every time, so sampling never flickers.
    private inSurveySample(survey: Survey): boolean {
        const percent = survey.sample_percent ?? 100;
        if (percent >= 100) return true;
        let hash = 0;
        for (const char of (this.userId || this.getUserId()) + ':' + survey.id) {
            hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
        }
        return hash % 100 < percent;
    }

    private showSurvey(survey: Survey): void {
        const el = <K extends keyof HTMLElementTagNameMap>(tag: K, className = '', text = '') => {
            const node = document.createElement(tag);
            node.className = className;
            node.textContent = text; // never innerHTML: survey text is user-defined
            return node;
        };
        const form = el('form');
        form.id = 'siraaj-survey';
        form.setAttribute('aria-label', survey.name);
        const S = '#siraaj-survey';
        form.append(el('style', '', `
${S}{position:fixed;bottom:20px;right:20px;z-index:2147483647;width:340px;max-width:calc(100vw - 40px);box-sizing:border-box;padding:20px;background:#fff;color:#111827;border:1px solid #e5e7eb;border-radius:16px;box-shadow:0 12px 32px rgba(0,0,0,.12),0 2px 6px rgba(0,0,0,.06);font:14px/1.45 system-ui,-apple-system,sans-serif;animation:siraaj-in .25s ease-out}
${S} *{box-sizing:border-box}
@keyframes siraaj-in{from{opacity:0;transform:translateY(12px)}}
${S} .sj-close{position:absolute;top:10px;right:10px;width:28px;height:28px;border:0;border-radius:50%;background:none;color:#6b7280;font-size:20px;line-height:1;cursor:pointer}
${S} .sj-close:hover{background:#f3f4f6;color:#111827}
${S} fieldset{border:0;margin:0 0 16px;padding:0;min-width:0}
${S} legend{font-weight:600;margin-bottom:10px;padding:0 28px 0 0}
${S} textarea{width:100%;padding:10px;font:inherit;color:inherit;border:1px solid #d1d5db;border-radius:10px;resize:vertical;outline:none}
${S} textarea:focus{border-color:#111827;box-shadow:0 0 0 3px rgba(17,24,39,.1)}
${S} .sj-opts{display:flex;flex-direction:column;gap:6px}
${S} .sj-opt{display:flex;align-items:center;gap:10px;padding:8px 10px;border:1px solid #e5e7eb;border-radius:10px;cursor:pointer;transition:border-color .15s,background .15s}
${S} .sj-opt:hover{background:#f9fafb}
${S} .sj-opt:has(input:checked){border-color:#111827;background:#f9fafb}
${S} .sj-opt input{width:20px;height:20px;margin:0;accent-color:#111827;cursor:pointer}
${S} .sj-rating{display:flex;justify-content:space-between;gap:6px}
${S} .sj-emoji{flex:1;cursor:pointer;text-align:center}
${S} .sj-emoji input{position:absolute;opacity:0;pointer-events:none}
${S} .sj-emoji span{display:block;padding:6px 0;font-size:28px;border-radius:12px;filter:grayscale(1);opacity:.55;transition:transform .15s,filter .15s,opacity .15s,background .15s}
${S} .sj-emoji:hover span{filter:none;opacity:1;transform:scale(1.15)}
${S} .sj-emoji input:checked+span{filter:none;opacity:1;transform:scale(1.2);background:#f3f4f6}
${S} .sj-emoji input:focus-visible+span{outline:2px solid #111827}
${S} .sj-send{width:100%;padding:10px 14px;border:0;border-radius:10px;background:#111827;color:#fff;font:inherit;font-weight:600;cursor:pointer}
${S} .sj-send:hover{background:#374151}
${S} .sj-mark{display:block;margin-top:12px;text-align:center;font-size:11px;color:#9ca3af;text-decoration:none}
${S} .sj-mark:hover{color:#4b5563}
`));
        const close = el('button', 'sj-close', '×');
        close.type = 'button';
        close.setAttribute('aria-label', 'Close survey');
        form.append(close);

        const faces = ['😞', '🙁', '😐', '🙂', '😄'];
        survey.questions.forEach((question, i) => {
            const field = el('fieldset');
            field.append(el('legend', '', question.text));
            if (question.type === 'text') {
                const input = el('textarea');
                input.name = 'q' + i;
                input.rows = 3;
                input.maxLength = 2000;
                input.placeholder = 'Type your answer...';
                field.append(input);
            } else {
                const rating = question.type === 'rating';
                const group = el('div', rating ? 'sj-rating' : 'sj-opts');
                (rating ? ['1', '2', '3', '4', '5'] : question.options || []).forEach((value, j) => {
                    const label = el('label', rating ? 'sj-emoji' : 'sj-opt');
                    const input = el('input');
                    input.type = 'radio';
                    input.name = 'q' + i;
                    input.value = value;
                    if (rating) {
                        label.title = value + ' / 5';
                        input.setAttribute('aria-label', value + ' out of 5');
                    }
                    label.append(input, rating ? el('span', '', faces[j]) : el('span', '', value));
                    group.append(label);
                });
                field.append(group);
            }
            form.append(field);
        });

        form.append(el('button', 'sj-send', 'Send'));
        const mark = el('a', 'sj-mark', 'Powered by Siraaj');
        mark.href = 'https://github.com/mohamedelhefni/siraaj';
        mark.target = '_blank';
        mark.rel = 'noopener';
        form.append(mark);
        close.onclick = () => {
            form.remove();
            this.track('survey_dismissed', { survey_id: survey.id });
            this.surveyPost('event', { survey_id: survey.id, kind: 'dismissed' });
        };
        form.onsubmit = event => {
            event.preventDefault();
            const data = new FormData(form);
            const answers = survey.questions.map((_, i) => String(data.get('q' + i) || '').trim());
            if (answers.every(answer => !answer)) return;
            form.remove();
            try {
                localStorage.setItem(this.STORAGE_PREFIX + 'survey_' + survey.id + '_answered', '1');
            } catch { /* only until_answered reads it; worst case the survey shows again */ }
            this.track('survey_sent', { survey_id: survey.id });
            this.surveyPost('respond', { survey_id: survey.id, user_id: this.userId || this.getUserId(), answers });
        };

        document.body.append(form);
        this.track('survey_shown', { survey_id: survey.id });
        this.surveyPost('event', { survey_id: survey.id, kind: 'shown' });
    }

    private surveyPost(path: 'respond' | 'event', body: Record<string, unknown>): void {
        this.backoffFetch(`${this.config.apiUrl}/api/surveys/${path}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'X-Siraaj-Token': this.config.trackingToken },
            body: JSON.stringify(body),
            keepalive: true,
        }).catch(err => this.log('Survey request error:', err));
    }

    private log(...args: any[]): void {
        if (this.config.debug && typeof console !== 'undefined') {
            console.log('[Siraaj]', ...args);
        }
    }

    private warn(...args: any[]): void {
        if (typeof console !== 'undefined') {
            console.warn('[Siraaj]', ...args);
        }
    }
}

// Export both the class and a default instance
export { AnalyticsCore };
export const analytics = new AnalyticsCore();
export type { AnalyticsConfig, EventData, DeviceInfo } from './types';
