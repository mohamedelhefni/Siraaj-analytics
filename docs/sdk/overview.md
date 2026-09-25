# SDK Overview

Siraaj provides a lightweight, easy-to-use JavaScript SDK that works everywhere. No framework required, no complex setup - just include the script and start tracking!

## Installation

### CDN / Self-Hosted (Easiest)

The simplest way to get started - add this to your HTML:

```html
<script src="http://localhost:8080/sdk/analytics.js"></script>
<script>
  var analytics = new SiraajAnalytics.AnalyticsCore({
    apiUrl: 'http://localhost:8080',
    projectId: 'my-website',
    autoTrack: true
  });
  
  analytics.init({
    apiUrl: 'http://localhost:8080',
    projectId: 'my-website'
  });
</script>
```

That's it! Auto-tracking is now enabled. 🎉

### NPM Package

```bash
npm install @hefni101/siraaj
# or
pnpm add @hefni101/siraaj
# or
yarn add @hefni101/siraaj
```

```javascript
import { AnalyticsCore } from '@hefni101/siraaj';

const analytics = new AnalyticsCore({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-app',
  autoTrack: true
});

analytics.init({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-app'
});
```

## Why This SDK?

### 🪶 Lightweight
- **< 16KB gzipped** - Tiny bundle size
- **Zero dependencies** - Pure vanilla JavaScript
- **No impact on performance** - Async loading and batching

### ⚡ Simple
- **3 lines of code** to get started
- **Auto-tracking** - No need to manually track everything
- **Works everywhere** - Any website, any framework

### 🎯 Powerful
- **Custom events** - Track anything you want
- **User identification** - Know who your users are
- **E-commerce ready** - Built-in support for purchases, carts, etc.

### 🔒 Privacy-First
- **No cookies** - Uses sessionStorage and localStorage
- **Respects DNT** - Honors Do Not Track
- **Sampling support** - Track only what you need

## Core Features

### Auto-Tracking

Enable automatic tracking with one option:

```javascript
const analytics = new SiraajAnalytics.AnalyticsCore({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-website',
  autoTrack: true  // ✨ Magic happens here
});

analytics.init({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-website'
});
```

This automatically tracks:
- ✅ Page views
- ✅ Link clicks
- ✅ Form submissions
- ✅ JavaScript errors
- ✅ Page visibility changes

### Custom Events

Track any event you want:

```javascript
analytics.track('button_clicked', {
  button_id: 'signup',
  location: 'hero',
  variant: 'primary'
});

analytics.track('video_played', {
  video_id: 'intro-video',
  duration: 120,
  quality: '1080p'
});

analytics.track('search_performed', {
  query: 'analytics',
  results_count: 42
});
```

### User Identification

Identify users for better insights:

```javascript
// On login
analytics.identify('user-123', {
  email: 'user@example.com',
  name: 'John Doe',
  plan: 'premium',
  signup_date: '2024-01-01'
});

// Update user properties
analytics.setUserProperties({
  plan: 'enterprise',
  mrr: 299
});

// On logout
analytics.reset();
```

### Page Views

Track page navigation:

```javascript
// Simple page view
analytics.pageView();

// With custom URL
analytics.pageView('/products');

// With properties
analytics.pageView('/products/123', {
  product_name: 'Awesome Widget',
  category: 'electronics'
});
```

## Configuration Options

```javascript
const analytics = new SiraajAnalytics.AnalyticsCore({
  // Required
  apiUrl: 'http://localhost:8080',    // Your Siraaj server URL
  projectId: 'my-website',            // Project identifier
  
  // Optional - Auto-tracking
  autoTrack: true,                    // Enable auto-tracking (default: true)
  
  // Optional - Debugging
  debug: false,                       // Enable console logs (default: false)
  
  // Optional - Performance
  bufferSize: 10,                     // Events per batch (default: 10)
  flushInterval: 30000,               // Auto-flush ms (default: 30000)
  timeout: 10000,                     // Request timeout ms (default: 10000)
  maxRetries: 3,                      // Retry attempts (default: 3)
  useBeacon: true,                    // Use sendBeacon API (default: true)
  
  // Optional - Privacy
  sampling: 1.0,                      // Sample rate 0-1 (default: 1.0)
  respectDoNotTrack: true,            // Honor DNT header (default: true)
  
  // Optional - Advanced
  enablePerformanceTracking: false,   // Track Web Vitals (default: false)
  maxQueueSize: 50                    // Max failed events queue (default: 50)
});
```

## API Methods

### `analytics.track(eventName, properties?)`

Track a custom event with optional properties.

```javascript
analytics.track('purchase_completed', {
  order_id: 'order-123',
  total: 99.99,
  currency: 'USD',
  items: [
    { product_id: 'prod-456', quantity: 1 }
  ]
});
```

### `analytics.pageView(url?, properties?)`

Track a page view.

```javascript
analytics.pageView('/products/123', {
  product_name: 'Widget',
  category: 'electronics'
});
```

### `analytics.identify(userId, traits?)`

Identify a user.

```javascript
analytics.identify('user-456', {
  name: 'John Doe',
  email: 'john@example.com',
  plan: 'enterprise'
});
```

### `analytics.trackClick(elementId, properties?)`

Track a click event.

```javascript
analytics.trackClick('signup-button', {
  location: 'navbar',
  variant: 'cta'
});
```

### `analytics.trackForm(formId, properties?)`

Track a form submission.

```javascript
analytics.trackForm('contact-form', {
  form_type: 'contact',
  source: 'landing-page'
});
```

### `analytics.trackError(error, context?)`

Track an error.

```javascript
try {
  riskyOperation();
} catch (error) {
  analytics.trackError(error, {
    component: 'checkout',
    action: 'process_payment'
  });
}
```

### `analytics.flush()`

Manually flush the event buffer.

```javascript
await analytics.flush();
```

### `analytics.reset()`

Reset session and user ID (call on logout).

```javascript
analytics.reset();
```

## Bundle Sizes

All sizes are **gzipped**:

| File | Size | Description |
|------|------|-------------|
| `analytics.min.js` | **< 5 KB** | Minified UMD build for browsers |
| `analytics.js` | ~8 KB | Full UMD build with source maps |
| `dist/analytics.esm.js` | ~8 KB | ES Module for bundlers |

## Browser Support

- ✅ Chrome/Edge ≥ 90
- ✅ Firefox ≥ 88
- ✅ Safari ≥ 14
- ✅ All modern browsers with ES2020 support

## TypeScript Support

Full TypeScript definitions included:

```typescript
import type { AnalyticsConfig, EventData } from '@hefni101/siraaj';

const config: AnalyticsConfig = {
  apiUrl: 'http://localhost:8080',
  projectId: 'my-app',
  debug: true,
  autoTrack: true
};
```

## Privacy Features

### Respect Do Not Track

Automatically respects the DNT header:

```javascript
const analytics = new SiraajAnalytics.AnalyticsCore({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-website',
  respectDoNotTrack: true  // default: true
});
```

### Sampling

Track only a percentage of users:

```javascript
const analytics = new SiraajAnalytics.AnalyticsCore({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-website',
  sampling: 0.5  // Track 50% of users
});
```

### No Cookies

Siraaj uses `sessionStorage` and `localStorage` instead of cookies:
- ✅ No cookie banners needed
- ✅ No third-party tracking
- ✅ Session-based analytics

## Performance

### Batching

Events are automatically batched to reduce requests:

```javascript
const analytics = new SiraajAnalytics.AnalyticsCore({
  bufferSize: 20,       // Send after 20 events
  flushInterval: 30000  // Or every 30 seconds
});
```

### Retry Logic

Failed requests are automatically retried with exponential backoff:

```javascript
const analytics = new SiraajAnalytics.AnalyticsCore({
  maxRetries: 3,        // Retry up to 3 times
  timeout: 10000        // 10 second timeout
});
```

### SendBeacon API

Uses `navigator.sendBeacon` for reliability during page unload:

```javascript
const analytics = new SiraajAnalytics.AnalyticsCore({
  useBeacon: true  // default: true
});
```

## Next Steps

- 📖 [Detailed Vanilla JS Guide →](/sdk/vanilla)
- 🎯 [Track Custom Events →](/sdk/custom-events)
- 👤 [User Identification →](/sdk/user-identification)
- ⚙️ [Configuration Options →](/sdk/configuration)
- 🚀 [Auto-Tracking →](/sdk/auto-tracking)

## Need Help?

- 💬 [GitHub Issues](https://github.com/mohamedelhefni/siraaj/issues)
- 📧 Email: mohamed.elhefni@outlook.com
- 📚 [Full Documentation](/guide/introduction)
