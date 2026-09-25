<div align="center">
  <img src="./logo.png" alt="Siraaj Logo" width="200"/>
  <h1>Siraaj Analytics</h1>
  <p><strong>Privacy-First, Self-Hosted Web Analytics</strong></p>
  
  <p>
    <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go" alt="Go"/>
    <img src="https://img.shields.io/badge/DuckDB-Powered-yellow?style=flat" alt="DuckDB"/>
    <img src="https://img.shields.io/badge/License-AGPL--3.0-blue?style=flat" alt="License"/>
    <a href="https://hub.docker.com/r/mohamedelhefni/siraaj">
      <img src="https://img.shields.io/docker/pulls/mohamedelhefni/siraaj?style=flat&logo=docker" alt="Docker"/>
    </a>
  </p>
  
  <p>
    Fast, lightweight analytics platform built with Go and DuckDB. Track your web traffic without cookies or compromising privacy.
  </p>
</div>

---

## ✨ Features

- 🔒 **Privacy-First** - No cookies, GDPR compliant
- ⚡ **Lightning Fast** - DuckDB delivers sub-50ms queries
- 📊 **Real-Time Dashboard** - Beautiful Svelte UI
- 🎯 **Simple Integration** - Drop-in JavaScript SDK (< 5KB)
- 🌍 **Multi-Project** - Track unlimited websites
- 📈 **Funnel Analysis** - Measure conversions
- 🤖 **Bot Detection** - Filter automated traffic
- 🎨 **Channel Attribution** - Understand traffic sources
- 🔗 **Short-Link Analytics** - Create compact links and measure clicks by country and referrer
- 👤 **User Authentication** - Bootstrap an administrator, add dashboard users, and use signed sessions
- 🔑 **Private Ingestion** - Revocable tracking tokens are scoped to one project
- 🧱 **Tenant Isolation** - Every account can query only projects, events, links, and tokens it owns

---

## 🚀 Quick Start

**Docker:**
```bash
docker run -d -p 8080:8080 -v $(pwd)/data:/data mohamedelhefni/siraaj:latest
```

**Docker Compose:**
```yaml
version: '3.8'
services:
  siraaj:
    image: mohamedelhefni/siraaj:latest
    ports: ["8080:8080"]
    volumes: ["./data:/data"]
    environment:
      - DUCKDB_MEMORY_LIMIT=4GB
      - AUTH_SECRET=replace-with-at-least-32-random-characters
    restart: unless-stopped
```

**Build from Source:**
```bash
git clone https://github.com/mohamedelhefni/siraaj.git && cd siraaj
go build -o siraaj && ./siraaj
```

**Dashboard:** http://localhost:8080/dashboard/

On first launch, choose **Server owner: first-time setup** to create the administrator. After that, anyone can create an isolated account from **Create account**. Each account starts empty and creates its first project by issuing a tracking token under **Users & tracking keys**.

> Pre-built binaries coming soon! ⭐

---

## � Usage

**Add SDK to your website:**
```html
<script>
  !function(){var s=document.createElement('script');
  s.src='http://your-server:8080/sdk/analytics.js';s.defer=!0;
  document.head.appendChild(s);s.onload=function(){
    window.siraaj=new Analytics({
      apiUrl:'http://your-server:8080',
      projectId:'my-website',
      trackingToken:'siraaj_trk_your_project_token',
      autoTrack:true
    });
  }}();
</script>
```

**Track custom events:**
```javascript
siraaj.track('purchase', { product: 'Premium', price: 99 });
siraaj.identify('user-123', { plan: 'premium' });
```

**Create and measure a short link:**
```bash
curl -X POST http://localhost:8080/api/links \
  -H 'Content-Type: application/json' \
  -d '{"destination_url":"https://example.com/launch","custom_slug":"launch","project_id":"marketing"}'
```

Open the **Links** area in the dashboard to copy the short URL and inspect its click timeline, countries, and referring sites.

**Framework integrations:** React, Vue, Svelte, Next.js → [SDK Docs](./sdk/README.md)

---

## ⚙️ Configuration

**Key Environment Variables:**
```bash
PORT=8080                           # Server port
DB_PATH=data/analytics.db           # Database path
DUCKDB_MEMORY_LIMIT=4GB             # Memory limit
AUTH_SECRET=32-or-more-random-characters # Signs dashboard sessions
AUTH_TOKEN_TTL=24h                  # Optional session lifetime
CORS=https://example.com            # CORS origins
```

**API Endpoints:** `/api/track`, `/api/stats`, `/api/funnel`, `/api/channels` → [Full API Docs](./docs/api/overview.md)

---

## 🏗️ Tech Stack

**Backend:** Go 1.24 + DuckDB + Parquet  
**Frontend:** SvelteKit 5 + Tailwind  
**SDK:** Vanilla JS/TypeScript  
**Architecture:** Clean Architecture (domain, repository, service, handler)

---

## 🤝 Contributing

Contributions welcome! Fork, create a feature branch, and submit a PR.

```bash
go mod download && cd dashboard && pnpm install && cd .. && make test
```

---

<div align="center">
  <p>Built with ❤️ by <a href="https://github.com/mohamedelhefni">Mohamed Elhefni</a></p>
  <p>⭐ Star this repo if you find it useful!</p>
</div>
