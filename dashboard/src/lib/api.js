// API configuration
// Use environment variable or fallback to relative path for production
const API_BASE_URL = import.meta.env.VITE_PUBLIC_API_URL || '/api';
const AUTH_STORAGE_KEY = 'siraaj_access_token';
const nativeFetch = globalThis.fetch.bind(globalThis);

export function getAccessToken() {
    return typeof localStorage === 'undefined' ? '' : localStorage.getItem(AUTH_STORAGE_KEY) || '';
}

export function clearAccessToken() {
    if (typeof localStorage !== 'undefined') localStorage.removeItem(AUTH_STORAGE_KEY);
}

/** @param {RequestInfo | URL} input @param {RequestInit} [init] */
async function fetch(input, init = {}) {
    const headers = new Headers(init.headers || {});
    const token = getAccessToken();
    if (token) headers.set('Authorization', `Bearer ${token}`);
    const response = await nativeFetch(input, { ...init, headers });
    if (response.status === 401 && token && typeof window !== 'undefined') {
        clearAccessToken();
        window.dispatchEvent(new CustomEvent('siraaj:unauthorized'));
    }
    return response;
}

/** @param {string} email @param {string} password */
export async function login(email, password) {
    const response = await nativeFetch(`${API_BASE_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || 'Unable to sign in');
    localStorage.setItem(AUTH_STORAGE_KEY, payload.access_token);
    return payload.user;
}

/** @param {string} email @param {string} password */
export async function signup(email, password) {
    const response = await nativeFetch(`${API_BASE_URL}/auth/signup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || 'Unable to create account');
    localStorage.setItem(AUTH_STORAGE_KEY, payload.access_token);
    return payload.user;
}

/** @param {string} email @param {string} password */
export async function bootstrap(email, password) {
    const response = await nativeFetch(`${API_BASE_URL}/auth/bootstrap`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || 'Unable to create administrator');
    return login(email, password);
}

export async function fetchCurrentUser() {
    const response = await fetch(`${API_BASE_URL}/auth/me`);
    if (!response.ok) throw new Error('Authentication required');
    return response.json();
}

export async function fetchUsers() {
    const response = await fetch(`${API_BASE_URL}/users`);
    if (!response.ok) throw new Error((await response.json()).error || 'Unable to load users');
    return response.json();
}

/** @param {{email: string, password: string, role: string}} request */
export async function createUser(request) {
    const response = await fetch(`${API_BASE_URL}/users`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(request)
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || 'Unable to create user');
    return payload;
}

export async function fetchTrackingTokens() {
    const response = await fetch(`${API_BASE_URL}/tracking-tokens`);
    if (!response.ok) throw new Error((await response.json()).error || 'Unable to load tracking tokens');
    return response.json();
}

/** @param {{project_id: string, name: string}} request */
export async function createTrackingToken(request) {
    const response = await fetch(`${API_BASE_URL}/tracking-tokens`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(request)
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || 'Unable to create tracking token');
    return payload;
}

/** @param {string} id */
export async function revokeTrackingToken(id) {
    const response = await fetch(`${API_BASE_URL}/tracking-tokens?id=${encodeURIComponent(id)}`, { method: 'DELETE' });
    if (!response.ok) throw new Error('Unable to revoke tracking token');
}

/**
 * Fetch analytics stats from the backend
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for top results (default 50)
 * @param {Record<string, string>} filters - Optional filters {source, country, browser, device, os, event, project, metric, botFilter, page}
 * @returns {Promise<Object>} Analytics stats
 */
export async function fetchStats(startDate, endDate, limit = 50, filters = {}) {
    const params = new URLSearchParams();
    if (startDate) params.append('start', startDate);
    if (endDate) params.append('end', endDate);
    if (limit) params.append('limit', limit.toString());

    // Add filters
    if (filters.source) params.append('source', filters.source);
    if (filters.country) params.append('country', filters.country);
    if (filters.browser) params.append('browser', filters.browser);
    if (filters.device) params.append('device', filters.device);
    if (filters.os) params.append('os', filters.os);
    if (filters.event) params.append('event', filters.event);
    if (filters.project) params.append('project', filters.project);
    if (filters.metric) params.append('metric', filters.metric);
    if (filters.botFilter) params.append('botFilter', filters.botFilter);
    if (filters.page) params.append('page', filters.page);

    const response = await fetch(`${API_BASE_URL}/stats?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch stats: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch comparison stats for previous period
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for top results (default 50)
 * @param {Record<string, string>} filters - Optional filters {source, country, browser, device, os, event, project, metric, botFilter}
 * @returns {Promise<Object>} Analytics stats for comparison period
 */
export async function fetchComparisonStats(startDate, endDate, limit = 50, filters = {}) {
    // Calculate the date range duration
    const start = new Date(startDate);
    const end = new Date(endDate);
    const duration = end.getTime() - start.getTime();

    // Calculate previous period dates
    const prevEnd = new Date(start.getTime() - 24 * 60 * 60 * 1000); // Day before start
    const prevStart = new Date(prevEnd.getTime() - duration);

    const prevStartStr = prevStart.toISOString().split('T')[0];
    const prevEndStr = prevEnd.toISOString().split('T')[0];

    // Fetch stats for previous period
    return fetchStats(prevStartStr, prevEndStr, limit, filters);
}

/**
 * Fetch all events with pagination
 * @param {string} startDate - Start date
 * @param {string} endDate - End date
 * @param {number} limit - Number of events to fetch
 * @param {number} offset - Offset for pagination
 * @returns {Promise<Object>} Events data
 */
export async function fetchEvents(startDate, endDate, limit = 100, offset = 0) {
    const params = new URLSearchParams();
    if (startDate) params.append('start', startDate);
    if (endDate) params.append('end', endDate);
    params.append('limit', limit.toString());
    params.append('offset', offset.toString());

    const response = await fetch(`${API_BASE_URL}/events?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch events: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch online users count
 * @param {number} timeWindowMinutes - Time window in minutes (default 5)
 * @returns {Promise<Object>} Online users data
 */
export async function fetchOnlineUsers(timeWindowMinutes = 5) {
    const params = new URLSearchParams();
    params.append('window', timeWindowMinutes.toString());

    const response = await fetch(`${API_BASE_URL}/online?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch online users: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch list of available projects
 * @returns {Promise<Array<string>>} List of project IDs
 */
export async function fetchProjects() {
    const response = await fetch(`${API_BASE_URL}/projects`);
    if (!response.ok) {
        throw new Error(`Failed to fetch projects: ${response.statusText}`);
    }
    return response.json();
}

/**
 * @param {{destination_url: string, custom_slug: string, project_id: string}} request
 */
export async function createShortLink(request) {
    const response = await fetch(`${API_BASE_URL}/links`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
    });
    if (!response.ok) {
        throw new Error((await response.text()).trim() || 'Failed to create short link');
    }
    return response.json();
}

export async function fetchShortLinks(project = '') {
    const params = new URLSearchParams();
    if (project) params.set('project', project);
    const response = await fetch(`${API_BASE_URL}/links?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch short links: ${response.statusText}`);
    }
    return response.json();
}

/**
 * @param {string} slug
 * @param {string} startDate
 * @param {string} endDate
 */
export async function fetchShortLinkStats(slug, startDate, endDate) {
    const params = new URLSearchParams({ slug });
    if (startDate) params.set('start', startDate);
    if (endDate) params.set('end', endDate);
    const response = await fetch(`${API_BASE_URL}/links/stats?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch link stats: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Health check
 * @returns {Promise<Object>} Health status
 */
export async function healthCheck() {
    const response = await fetch(`${API_BASE_URL}/health`);
    if (!response.ok) {
        throw new Error(`Failed to fetch health status: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch funnel analysis results
 * @param {Object} funnelRequest - Funnel configuration
 * @param {Array<Object>} funnelRequest.steps - Array of funnel steps
 * @param {string} funnelRequest.start_date - Start date (YYYY-MM-DD)
 * @param {string} funnelRequest.end_date - End date (YYYY-MM-DD)
 * @param {Object} funnelRequest.filters - Global filters
 * @returns {Promise<Object>} Funnel analysis results
 */
export async function fetchFunnelAnalysis(funnelRequest) {
    const response = await fetch(`${API_BASE_URL}/funnel`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(funnelRequest),
    });
    if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to fetch funnel analysis: ${errorText || response.statusText}`);
    }
    return response.json();
}

// ========== New Focused Endpoints ==========

/**
 * Helper to build query params from filters
 * @param {string} startDate - Start date
 * @param {string} endDate - End date
 * @param {number} limit - Limit for results
 * @param {Record<string, string>} filters - Filters object
 * @returns {URLSearchParams} Query parameters
 */
function buildQueryParams(startDate, endDate, limit = 50, filters = {}) {
    const params = new URLSearchParams();
    if (startDate) params.append('start', startDate);
    if (endDate) params.append('end', endDate);
    if (limit) params.append('limit', limit.toString());

    // Add filters
    if (filters.source) params.append('source', filters.source);
    if (filters.country) params.append('country', filters.country);
    if (filters.browser) params.append('browser', filters.browser);
    if (filters.device) params.append('device', filters.device);
    if (filters.os) params.append('os', filters.os);
    if (filters.event) params.append('event', filters.event);
    if (filters.project) params.append('project', filters.project);
    if (filters.metric) params.append('metric', filters.metric);
    if (filters.botFilter) params.append('botFilter', filters.botFilter);
    if (filters.page) params.append('page', filters.page);

    return params;
}

/**
 * Fetch top-level statistics (counts, rates, trends)
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Object>} Top stats
 */
export async function fetchTopStats(startDate, endDate, filters = {}) {
    const params = buildQueryParams(startDate, endDate, 50, filters);

    const response = await fetch(`${API_BASE_URL}/stats/overview?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch top stats: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch timeline data for the main chart
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {Record<string, string>} filters - Optional filters (including metric)
 * @returns {Promise<Object>} Timeline data with format
 */
export async function fetchTimeline(startDate, endDate, filters = {}) {
    const params = buildQueryParams(startDate, endDate, 50, filters);

    const response = await fetch(`${API_BASE_URL}/stats/timeline?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch timeline: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch top pages
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for results
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Object>} Top pages data
 */
export async function fetchTopPages(startDate, endDate, limit = 10, filters = {}) {
    const params = buildQueryParams(startDate, endDate, limit, filters);

    const response = await fetch(`${API_BASE_URL}/stats/pages?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch top pages: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch entry and exit pages
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for results
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Object>} Entry and exit pages
 */
export async function fetchEntryExitPages(startDate, endDate, limit = 10, filters = {}) {
    const params = buildQueryParams(startDate, endDate, limit, filters);

    const response = await fetch(`${API_BASE_URL}/stats/pages/entry-exit?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch entry/exit pages: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch top countries
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for results
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Array<unknown>>} Top countries
 */
export async function fetchTopCountries(startDate, endDate, limit = 10, filters = {}) {
    const params = buildQueryParams(startDate, endDate, limit, filters);

    const response = await fetch(`${API_BASE_URL}/stats/countries?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch top countries: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch top traffic sources
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for results
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Array<unknown>>} Top sources
 */
export async function fetchTopSources(startDate, endDate, limit = 10, filters = {}) {
    const params = buildQueryParams(startDate, endDate, limit, filters);

    const response = await fetch(`${API_BASE_URL}/stats/sources?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch top sources: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch top events
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for results
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Array<unknown>>} Top events
 */
export async function fetchTopEvents(startDate, endDate, limit = 10, filters = {}) {
    const params = buildQueryParams(startDate, endDate, limit, filters);

    const response = await fetch(`${API_BASE_URL}/stats/events?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch top events: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch browsers, devices, and operating systems
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {number} limit - Limit for results
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Object>} Browsers, devices, and OS data
 */
export async function fetchBrowsersDevicesOS(startDate, endDate, limit = 10, filters = {}) {
    const params = buildQueryParams(startDate, endDate, limit, filters);

    const response = await fetch(`${API_BASE_URL}/stats/devices?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch browsers/devices/OS: ${response.statusText}`);
    }
    return response.json();
}

/**
 * Fetch channel analytics (traffic source breakdown)
 * @param {string} startDate - Start date in YYYY-MM-DD format
 * @param {string} endDate - End date in YYYY-MM-DD format
 * @param {Record<string, string>} filters - Optional filters
 * @returns {Promise<Array<unknown>>} Channel breakdown with metrics
 */
export async function fetchChannels(startDate, endDate, filters = {}) {
    const params = buildQueryParams(startDate, endDate, 50, filters);

    const response = await fetch(`${API_BASE_URL}/channels?${params}`);
    if (!response.ok) {
        throw new Error(`Failed to fetch channels: ${response.statusText}`);
    }
    return response.json();
}
