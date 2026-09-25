# Siraaj Analytics SDK

> 🎯 **The simplest analytics SDK you'll ever use** - Just 3 lines of code, < 5KB, works everywhere!

## ⚡ Quick Start (30 seconds)

### 1. Add the script to your HTML

```html
<script src="http://localhost:8080/sdk/analytics.js"></script>
<script>
  var analytics = new SiraajAnalytics.AnalyticsCore({
    apiUrl: 'http://localhost:8080',
    projectId: 'my-website',
    trackingToken: 'siraaj_trk_your_project_token',
    autoTrack: true
  });
  
  analytics.init({
    apiUrl: 'http://localhost:8080',
    projectId: 'my-website',
    trackingToken: 'siraaj_trk_your_project_token'
  });
</script>
```

### 2. Done! 🎉

Auto-tracking is enabled. Your page views, clicks, forms, and errors are now being tracked!

## 📦 What You Get

- **< 5KB** - Tiny bundle size, zero impact on performance
- **Auto-tracking** - Page views, clicks, forms, errors - all automatic
- **No dependencies** - Pure vanilla JavaScript
- **Easy to share** - Just send the `.js` file
- **TypeScript ready** - Full type definitions included
- **Privacy-first** - No cookies, respects Do Not Track

## 🚀 Features

### Auto-Tracking (Zero Config)

When `autoTrack: true`, the SDK automatically tracks:

```javascript
✅ Page views (on load and navigation)
✅ Link clicks (all <a> tags)
✅ Form submissions
✅ JavaScript errors
✅ Page visibility changes
```

### Custom Event Tracking

```javascript
// Track any custom event
analytics.track('button_clicked', {
  button_id: 'signup',
  location: 'hero',
  variant: 'primary'
});

// Track e-commerce events
analytics.track('purchase_completed', {
  order_id: 'order-123',
  total: 99.99,
  currency: 'USD',
  items: [{ product_id: 'prod-456', quantity: 1 }]
});

// Track video plays
analytics.track('video_played', {
  video_id: 'intro-video',
  duration: 120,
  quality: '1080p'
});
```

### User Identification

```javascript
// On login
analytics.identify('user-123', {
  email: 'user@example.com',
  name: 'John Doe',
  plan: 'premium'
});

// On logout
analytics.reset();
```

### Manual Page Views

```javascript
// Track page navigation in SPAs
analytics.pageView('/products');

// With custom properties
analytics.pageView('/products/123', {
  product_name: 'Awesome Widget',
  category: 'electronics'
});
```

## 📥 Installation Options

### Option 1: CDN / Self-Hosted (Recommended)

```html
<!-- From your Siraaj server -->
<script src="http://localhost:8080/sdk/analytics.js"></script>

<!-- Or from any CDN -->
<script src="https://your-cdn.com/analytics.js"></script>
```

### Option 2: NPM Package

```bash
npm install @hefni101/siraaj
```

```javascript
// ES Modules
import { AnalyticsCore } from '@hefni101/siraaj';

const analytics = new AnalyticsCore({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-app'
});

analytics.init({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-app'
});
```

### Option 3: Download & Include

1. Build the SDK: `cd sdk && pnpm install && pnpm build`
2. Get `analytics.js` or `analytics.min.js`
3. Include in your project

## ⚙️ Configuration

```javascript
var analytics = new SiraajAnalytics.AnalyticsCore({
  // Required (create the token in Dashboard → Users & tracking keys)
  apiUrl: 'http://localhost:8080',     // Your Siraaj server URL
  projectId: 'my-website',             // Your project ID
  trackingToken: 'siraaj_trk_...',     // Project-scoped write token
  
  // Optional
  autoTrack: true,                     // Enable auto-tracking (default: true)
  debug: false,                        // Console logging (default: false)
  bufferSize: 10,                      // Events per batch (default: 10)
  flushInterval: 30000,                // Auto-flush ms (default: 30000)
  timeout: 10000,                      // Request timeout ms (default: 10000)
  maxRetries: 3,                       // Retry attempts (default: 3)
  useBeacon: true,                     // Use sendBeacon API (default: true)
  sampling: 1.0,                       // Sample rate 0-1 (default: 1.0)
  respectDoNotTrack: true,             // Honor DNT (default: true)
  enablePerformanceTracking: false     // Web Vitals (default: false)
});
```

## 🎯 Usage Examples

### Basic Website

```html
<!DOCTYPE html>
<html>
<head>
  <title>My Website</title>
  <script src="http://localhost:8080/sdk/analytics.js"></script>
  <script>
    var analytics = new SiraajAnalytics.AnalyticsCore({
      apiUrl: 'http://localhost:8080',
      projectId: 'my-website',
      autoTrack: true,
      debug: true
    });
    
    analytics.init({
      apiUrl: 'http://localhost:8080',
      projectId: 'my-website'
    });
  </script>
</head>
<body>
  <h1>Welcome!</h1>
  <button onclick="analytics.track('signup_clicked')">
    Sign Up
  </button>
</body>
</html>
```

### Single Page Application

```html
<script src="analytics.js"></script>
<script>
  var analytics = new SiraajAnalytics.AnalyticsCore({
    apiUrl: 'http://localhost:8080',
    projectId: 'my-spa',
    autoTrack: false  // Disable auto page views
  });
  
  analytics.init({
    apiUrl: 'http://localhost:8080',
    projectId: 'my-spa'
  });

  // Track route changes manually
  function navigate(path) {
    history.pushState({}, '', path);
    analytics.pageView(path);
    renderPage(path);
  }

  // Handle browser back/forward
  window.addEventListener('popstate', function() {
    analytics.pageView(window.location.pathname);
  });
</script>
```

### E-commerce Tracking

```javascript
// Product view
analytics.track('product_viewed', {
  product_id: 'prod-123',
  product_name: 'Wireless Headphones',
  price: 99.99,
  category: 'Electronics'
});

// Add to cart
analytics.track('add_to_cart', {
  product_id: 'prod-123',
  quantity: 1,
  price: 99.99
});

// Purchase
analytics.track('purchase_completed', {
  order_id: 'order-456',
  total: 109.99,
  currency: 'USD',
  items: [
    { product_id: 'prod-123', quantity: 1, price: 99.99 }
  ],
  shipping: 10.00
});
```

### React/Vue/Any Framework

The SDK works with any framework since it's vanilla JavaScript:

```jsx
// React example
function SignupButton() {
  return (
    <button onClick={() => {
      window.analytics.track('signup_clicked', {
        location: 'hero'
      });
    }}>
      Sign Up
    </button>
  );
}

// Vue example
<template>
  <button @click="handleClick">Sign Up</button>
</template>

<script>
export default {
  methods: {
    handleClick() {
      window.analytics.track('signup_clicked', {
        location: 'hero'
      });
    }
  }
}
</script>
```

## 📊 What Gets Tracked Automatically

With `autoTrack: true`:

| Event | Description | Properties |
|-------|-------------|------------|
| `page_view` | Page loads | url, title, referrer |
| `link_clicked` | Link clicks | url, text, external |
| `form_submit` | Form submissions | form_id, action |
| `error` | JS errors | message, stack, filename |
| `page_hidden`/`page_visible` | Tab visibility | - |

## 🔒 Privacy & Performance

### No Cookies
Uses `sessionStorage` and `localStorage` instead of cookies. No cookie banners needed!

### Respects Do Not Track
Automatically stops tracking if DNT is enabled.

### Sampling
Track only a percentage of users:

```javascript
var analytics = new SiraajAnalytics.AnalyticsCore({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-website',
  sampling: 0.1  // Track only 10% of users
});
```

### Batching & Buffering
Events are batched and sent together to reduce requests:

```javascript
var analytics = new SiraajAnalytics.AnalyticsCore({
  apiUrl: 'http://localhost:8080',
  projectId: 'my-website',
  bufferSize: 20,       // Send after 20 events
  flushInterval: 30000  // Or every 30 seconds
});
```

## 🔧 API Reference

### `analytics.track(eventName, properties)`
Track a custom event.

### `analytics.pageView(url, properties)`
Track a page view.

### `analytics.identify(userId, traits)`
Identify a user.

### `analytics.trackClick(elementId, properties)`
Track a click event.

### `analytics.trackForm(formId, properties)`
Track a form submission.

### `analytics.trackError(error, context)`
Track an error.

### `analytics.flush()`
Manually flush the event buffer.

### `analytics.reset()`
Reset session and user ID (call on logout).

### `analytics.destroy()`
Clean up and destroy the instance.

## 🛠️ Development

```bash
# Install dependencies
cd sdk && pnpm install

# Build
pnpm build

# Watch mode
pnpm dev

# Check types
pnpm typecheck
```

This creates:
- `analytics.js` - Full UMD build for browsers
- `analytics.min.js` - Minified version
- `dist/analytics.esm.js` - ES Module for bundlers
- `dist/analytics.d.ts` - TypeScript definitions

## 📝 Examples

Check out the `examples/` folder:

- `simple-website.html` - Basic example with demo buttons
- `share-ready.html` - Beautiful example ready to share
- `cdn-example.html` - CDN usage
- `umd-example.html` - UMD build usage

## 🤝 Sharing with Others

### Option 1: Share the file
Send `analytics.js` or `analytics.min.js` to anyone. That's it!

### Option 2: Host on CDN
Upload to your CDN and share the URL:
```html
<script src="https://your-cdn.com/analytics.js"></script>
```

### Option 3: Serve from Siraaj
Your Siraaj server automatically serves the SDK at:
```
http://localhost:8080/sdk/analytics.js
```

## 🌐 Browser Support

- ✅ Chrome 90+
- ✅ Firefox 88+
- ✅ Safari 14+
- ✅ Edge 90+
- ✅ All modern browsers with ES2020 support

## 📄 License

MIT © Mohamed Elhefni

---

**Need help?** Check out the [full documentation](../docs) or open an issue!


## Features

- 🪶 **Lightweight** - Core is < 3KB gzipped
- ⚡ **Fast** - Batching, buffering, and automatic retries
- 🎯 **Framework-specific** - Optimized hooks and components for React, Vue, Svelte, Preact, Next.js, and Nuxt
- 📦 **Tree-shakeable** - Only import what you need
- 🔒 **Type-safe** - Full TypeScript support
- 🚀 **Auto-tracking** - Page views, clicks, forms, and errors out of the box
- 🔄 **SSR Ready** - Works seamlessly with Next.js and Nuxt server-side rendering

## Installation

```bash
# Using pnpm
pnpm add @hefni101/siraaj

# Using npm
npm install @hefni101/siraaj

# Using yarn
yarn add @hefni101/siraaj
```

## Quick Start
pnpm add @hefni101/siraaj
### Vanilla JavaScript / TypeScript
npm install @hefni101/siraaj
```javascript
yarn add @hefni101/siraaj

# Siraaj Analytics SDK

> 🎯 **The simplest analytics SDK you'll ever use** - Just 3 lines of code, < 5KB, works everywhere!

## ⚡ Quick Start (30 seconds)

### 1. Add the script to your HTML

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

### 2. Done! 🎉

Auto-tracking is enabled. Your page views, clicks, forms, and errors are now being tracked!

## 📦 What You Get

- **< 5KB** - Tiny bundle size, zero impact on performance
- **Auto-tracking** - Page views, clicks, forms, errors - all automatic
- **No dependencies** - Pure vanilla JavaScript
- **Easy to share** - Just send the `.js` file
- **TypeScript ready** - Full type definitions included
- **Privacy-first** - No cookies, respects Do Not Track

## 🚀 Features
```

### React

```jsx
import { createAnalytics, usePageTracking } from '@hefni101/siraaj/svelte';

function App() {
  return (
    <AnalyticsProvider config={{
      endpoint: 'https://your-analytics-server.com',
      apiKey: 'your-api-key',
import { initAnalytics } from '@hefni101/siraaj/svelte';
      <YourApp />
    </AnalyticsProvider>
  );
}

function YourComponent() {
import { AnalyticsProvider } from '@hefni101/siraaj/next';
  
  // Auto-track page views
  usePageTracking();
  
  const handleClick = () => {
    track('button_clicked', { button: 'signup' });
import { initNuxtAnalytics } from '@hefni101/siraaj/nuxt';
  
  return <button onClick={handleClick}>Sign Up</button>;
}
```
import { useNuxtAnalytics } from '@hefni101/siraaj/nuxt';
### Vue 3

```vue
<script setup>
import { useAnalytics, usePageTracking } from '@hefni101/siraaj/vue';

import { AnalyticsProvider, useAnalytics } from '@hefni101/siraaj/preact';
const { track, identify } = useAnalytics();

// Auto-track page view
usePageTracking();

const handleClick = () => {
  track('button_clicked', { button: 'signup' });
};
</script>

<template>
  <button @click="handleClick">Sign Up</button>
</template>
```

**Plugin registration:**

```javascript
import { createApp } from 'vue';
import { AnalyticsPlugin } from '@hefni101/siraaj/vue';
import App from './App.vue';

const app = createApp(App);

app.use(AnalyticsPlugin, {
  endpoint: 'https://your-analytics-server.com',
  apiKey: 'your-api-key',
});

app.mount('#app');
```

### Svelte

```svelte
<script>
import { createAnalytics, usePageTracking } from '@hefni101/siraaj/svelte';

const { track, userId } = createAnalytics();

// Auto-track page view
usePageTracking();

const handleClick = () => {
  track('button_clicked', { button: 'signup' });
};
</script>

<button on:click={handleClick}>Sign Up</button>
```

**Initialization:**

```javascript
// In your main file
import { initAnalytics } from '@hefni101/siraaj/svelte';

initAnalytics({
  endpoint: 'https://your-analytics-server.com',
  apiKey: 'your-api-key',
});
```

### Next.js (App Router)

```tsx
// app/layout.tsx
import { AnalyticsProvider } from '@hefni101/siraaj/next';
import { initNextAnalytics } from '@hefni101/siraaj/next';

initNextAnalytics({
  endpoint: 'https://your-analytics-server.com',
  apiKey: 'your-api-key',
});

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html>
      <body>
        <AnalyticsProvider config={{
          endpoint: 'https://your-analytics-server.com',
          apiKey: 'your-api-key',
        }}>
          {children}
        </AnalyticsProvider>
      </body>
    </html>
  );
}

// app/page.tsx
'use client';
import { useNextAnalytics, useAnalytics } from '@hefni101/siraaj/next';

export default function Page() {
  const { track } = useAnalytics();
  
  // Auto-track route changes
  useNextAnalytics();
  
  return (
    <button onClick={() => track('clicked')}>
      Click Me
    </button>
  );
}
```

### Next.js (Pages Router)

```tsx
// pages/_app.tsx
import { AnalyticsProvider } from '@hefni101/siraaj/next';
import { useNextPagesAnalytics } from '@hefni101/siraaj/next';

export default function App({ Component, pageProps }) {
  // Auto-track route changes
  useNextPagesAnalytics();
  
  return (
    <AnalyticsProvider config={{
      endpoint: 'https://your-analytics-server.com',
      apiKey: 'your-api-key',
    }}>
      <Component {...pageProps} />
    </AnalyticsProvider>
  );
}
```

### Nuxt 3

```typescript
// plugins/analytics.client.ts
import { initNuxtAnalytics } from '@hefni101/siraaj/nuxt';

export default defineNuxtPlugin(() => {
  initNuxtAnalytics({
    endpoint: 'https://your-analytics-server.com',
    apiKey: 'your-api-key',
  });
});

// composables/useTracking.ts
import { useNuxtAnalytics } from '@hefni101/siraaj/nuxt';

export function useTracking() {
  // Auto-track route changes
  useNuxtAnalytics();
}

// pages/index.vue
<script setup>
import { useNuxtApp } from '#app';

const { $analytics } = useNuxtApp();

const handleClick = () => {
  $analytics.track('button_clicked', { button: 'signup' });
};
</script>

<template>
  <button @click="handleClick">Sign Up</button>
</template>
```

### Preact

```jsx
import { AnalyticsProvider, useAnalytics } from '@hefni101/siraaj/preact';

export function App() {
  return (
    <AnalyticsProvider config={{
      endpoint: 'https://your-analytics-server.com',
      apiKey: 'your-api-key',
    }}>
      <YourApp />
    </AnalyticsProvider>
  );
}

function YourComponent() {
  const { track } = useAnalytics();
  
  return (
    <button onClick={() => track('clicked')}>
      Click Me
    </button>
  );
}
```

## Configuration

```typescript
interface AnalyticsConfig {
  endpoint: string;        // Your analytics server endpoint
  apiKey: string;          // API key for authentication
  autoTrack?: boolean;     // Enable auto-tracking (default: true)
  debug?: boolean;         // Enable debug logs (default: false)
  batchSize?: number;      // Events per batch (default: 10)
  flushInterval?: number;  // Flush interval in ms (default: 5000)
  maxRetries?: number;     // Max retry attempts (default: 3)
  timeout?: number;        // Request timeout in ms (default: 10000)
}
```

## API Reference

### Core Methods

#### `analytics.init(config)`
Initialize the analytics SDK with configuration.

#### `analytics.track(event, properties?)`
Track a custom event with optional properties.

```javascript
analytics.track('purchase_completed', {
  productId: 'abc123',
  price: 29.99,
  currency: 'USD',
});
```

#### `analytics.page(properties?)`
Track a page view with optional properties.

```javascript
analytics.page({
  category: 'documentation',
  section: 'getting-started',
});
```

#### `analytics.identify(userId, properties?)`
Identify a user with optional properties.

```javascript
analytics.identify('user-123', {
  email: 'user@example.com',
  plan: 'premium',
  signupDate: '2024-01-01',
});
```

#### `analytics.flush()`
Manually flush the event queue.

```javascript
await analytics.flush();
```

#### `analytics.reset()`
Reset the session and user ID.

```javascript
analytics.reset();
```

## Auto-Tracking

When `autoTrack: true`, the SDK automatically tracks:

- **Page views** - On initialization and navigation
- **Clicks** - On links and buttons
- **Form submissions** - On form submit events
- **Errors** - JavaScript errors and exceptions

You can disable auto-tracking and manually track events as needed.

## Bundle Sizes

All sizes are **gzipped**:

| Package | Size | Description |
|---------|------|-------------|
| Core | < 3 KB | Vanilla JS/TS |
| React | < 4 KB | React hooks + core |
| Vue | < 4 KB | Vue composables + core |
| Svelte | < 4 KB | Svelte stores + core |
| Preact | < 4 KB | Preact hooks + core |
| Next.js | < 4 KB | Next.js integration + core |
| Nuxt | < 4 KB | Nuxt integration + core |

## Browser Support

- Chrome/Edge ≥ 90
- Firefox ≥ 88
- Safari ≥ 14
- All modern browsers with ES2020 support

## Development

```bash
# Install dependencies
pnpm install

# Build all packages
pnpm run build

# Watch mode
pnpm run dev

# Check bundle sizes
pnpm run size

# Analyze bundle
pnpm run analyze
```

## License

MIT © Mohamed Elhefni
