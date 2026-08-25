# API Overview

Complete RESTful API reference for Siraaj Analytics.

## Base URL

```
http://localhost:8080/api
```

## Authentication

Currently, Siraaj doesn't require authentication. API key authentication is planned for future releases.

When dashboard basic authentication is configured, short-link creation, listing, and analytics use the same credentials. Public `/s/{slug}` redirects never require authentication.

## Core Endpoints

### Health Check

Check server status and availability.

```http
GET /api/health
```

**Response**

```json
{
  "status": "healthy",
  "database": "connected"
}
```

---

### Track Event

Send a single analytics event.

```http
POST /api/track
Content-Type: application/json
```

**Request Body**

```json
{
  "event_name": "page_view",
  "user_id": "user-123",
  "session_id": "session-456",
  "url": "https://example.com/products",
  "referrer": "https://google.com",
  "user_agent": "Mozilla/5.0...",
  "timestamp": "2024-01-15T10:30:00Z",
  "browser": "Chrome",
  "os": "MacOS",
  "device": "Desktop",
  "project_id": "my-website",
  "ip": "192.168.1.1"
}
```

**Response**

```json
{
  "status": "ok"
}
```

**Note**: Channel classification happens automatically server-side based on referrer and URL parameters.

---

### Track Batch Events

Send multiple events at once for better performance (recommended).

```http
POST /api/track/batch
Content-Type: application/json
```

**Request Body**

```json
{
  "events": [
    {
      "event_name": "page_view",
      "user_id": "user-123",
      "url": "https://example.com/page1",
      "referrer": "https://google.com"
    },
    {
      "event_name": "button_clicked",
      "user_id": "user-123",
      "url": "https://example.com/page1"
    }
  ]
}
```

**Response**

```json
{
  "status": "ok",
  "received": 2
}
```

---

## Analytics Endpoints

### Create a Short Link

Create a tracked redirect. `custom_slug` is optional; when omitted, Siraaj generates an eight-character slug.

```http
POST /api/links
Content-Type: application/json
```

```json
{
  "destination_url": "https://example.com/launch",
  "custom_slug": "launch",
  "project_id": "marketing"
}
```

The destination must use HTTP or HTTPS. Custom slugs may contain 3–64 letters, numbers, hyphens, or underscores.

### List Short Links

```http
GET /api/links?project=marketing
```

Each link includes its public `short_url`, all-time `click_count`, and most recent click time.

### Redirect and Record a Click

```http
GET /s/launch
```

Siraaj records the click's country and referring domain, then responds with a `302` redirect. Visitor IP addresses are used transiently for country lookup and are not stored with link clicks.

### Get Short-Link Analytics

```http
GET /api/links/stats?slug=launch&start=2026-08-01&end=2026-08-31
```

The response includes total clicks, a daily timeline, and breakdowns by country and referring domain. The default range is the last 30 days.

---

### Get Statistics

Retrieve comprehensive analytics data with filtering support.

```http
GET /api/stats
```

**Query Parameters**

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| start | string | Start date (YYYY-MM-DD) | 7 days ago |
| end | string | End date (YYYY-MM-DD) | Today |
| project | string | Filter by project ID | All projects |
| country | string | Filter by country | All countries |
| browser | string | Filter by browser | All browsers |
| os | string | Filter by operating system | All OS |
| device | string | Filter by device type | All devices |
| source | string | Filter by referrer source | All sources |
| event | string | Filter by event name | All events |
| metric | string | Filter by specific metric | All metrics |
| botFilter | string | Filter bot traffic (human/bot) | All traffic |
| limit | integer | Limit top results | 50 |

**Example**

```bash
curl "http://localhost:8080/api/stats?start=2024-01-01&end=2024-01-31&project=my-website&botFilter=human"
```

---

### Get Overview Statistics

Get high-level statistics for dashboard overview.

```http
GET /api/stats/overview?start=2024-01-01&end=2024-01-31
```

**Response includes**: Total events, unique visitors, total visits, bounce rate, avg session duration, and comparisons with previous period.

---

### Get Timeline Data

Get event timeline data with automatic granularity (hourly/daily/monthly).

```http
GET /api/stats/timeline?start=2024-01-01&end=2024-01-31
```

---

### Get Top Pages

Get most visited pages with entry/exit statistics.

```http
GET /api/stats/pages?start=2024-01-01&end=2024-01-31&limit=20
```

---

### Get Entry/Exit Pages

Get entry and exit page statistics.

```http
GET /api/stats/pages/entry-exit?start=2024-01-01&end=2024-01-31
```

---

### Get Countries

Get visitor distribution by country.

```http
GET /api/stats/countries?start=2024-01-01&end=2024-01-31
```

---

### Get Sources

Get top referrer sources.

```http
GET /api/stats/sources?start=2024-01-01&end=2024-01-31
```

---

### Get Custom Events

Get custom event statistics.

```http
GET /api/stats/events?start=2024-01-01&end=2024-01-31
```

---

### Get Device Statistics

Get browser, OS, and device type distribution.

```http
GET /api/stats/devices?start=2024-01-01&end=2024-01-31
```

**Response includes**: Browsers, operating systems, and device types with counts.

---

### Get Channel Analytics

Get traffic channel distribution (Direct, Organic, Social, Referral, Paid).

```http
GET /api/channels?start=2024-01-01&end=2024-01-31
```

**Response**

```json
[
  {
    "channel": "Organic",
    "total_events": 2300,
    "unique_users": 890,
    "total_visits": 1100,
    "page_views": 2100,
    "conversion_rate": 1.91
  },
  {
    "channel": "Direct",
    "total_events": 1500,
    "unique_users": 450,
    "total_visits": 600,
    "page_views": 1200,
    "conversion_rate": 2.0
  }
]
```

**Channels**: Direct, Organic, Social, Referral, Paid, Unknown

---

### Get Online Users

Get current online users count (users active in last 5 minutes).

```http
GET /api/online?project=my-website
```

**Response**

```json
{
  "count": 42
}
```

---

### Get Projects

List all projects with event counts.

```http
GET /api/projects
```

**Response**

```json
[
  {
    "id": "my-website",
    "event_count": 15432
  },
  {
    "id": "my-app",
    "event_count": 8921
  }
]
```

---

### Get Funnel Analysis

Analyze user conversion funnels.

```http
POST /api/funnel
Content-Type: application/json
```

**Request Body**

```json
{
  "steps": [
    {
      "name": "Landing Page",
      "event_name": "page_view",
      "url": "/landing"
    },
    {
      "name": "Sign Up",
      "event_name": "signup_completed"
    },
    {
      "name": "First Purchase",
      "event_name": "purchase"
    }
  ],
  "start_date": "2024-01-01",
  "end_date": "2024-01-31",
  "filters": {
    "project": "my-website"
  }
}
```

**Response**

```json
{
  "steps": [
    {
      "step": { "name": "Landing Page" },
      "user_count": 1000,
      "session_count": 1200,
      "event_count": 1500,
      "conversion_rate": 100.0,
      "overall_rate": 100.0,
      "dropoff_rate": 0.0,
      "avg_time_to_next": 45.2,
      "median_time_to_next": 30.0
    }
  ],
  "total_users": 1000,
  "completed_users": 150,
  "completion_rate": 15.0,
  "avg_completion": 320.5,
  "time_range": "2024-01-01 to 2024-01-31"
}
```

---

### Get Raw Events

Retrieve raw event data (for debugging/export).

```http
GET /api/events?start=2024-01-01&end=2024-01-31&limit=100
```

---

## Error Responses

### 400 Bad Request

```json
{
  "error": "Invalid request parameters"
}
```

### 500 Internal Server Error

```json
{
  "error": "Internal server error"
}
```

## CORS Configuration

Configure allowed origins via environment variable:

```bash
export CORS=https://example.com,https://app.example.com
./siraaj
```

## Code Examples

### cURL

```bash
# Track single event
curl -X POST http://localhost:8080/api/track \
  -H "Content-Type: application/json" \
  -d '{
    "event_name": "page_view",
    "user_id": "user-123",
    "project_id": "my-website",
    "url": "https://example.com",
    "referrer": "https://google.com",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
  }'

# Get analytics
curl "http://localhost:8080/api/stats?start=2024-01-01&end=2024-01-31&project=my-website"
```

### JavaScript/TypeScript

```javascript
// Track event
await fetch('http://localhost:8080/api/track', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    event_name: 'button_clicked',
    user_id: 'user-123',
    project_id: 'my-app',
    url: window.location.href,
    referrer: document.referrer,
    timestamp: new Date().toISOString()
  })
});

// Get stats
const response = await fetch(
  'http://localhost:8080/api/stats?start=2024-01-01&end=2024-01-31'
);
const stats = await response.json();
```

### Python

```python
import requests
from datetime import datetime

# Track event
requests.post('http://localhost:8080/api/track', json={
    'event_name': 'purchase_completed',
    'user_id': 'user-123',
    'project_id': 'my-store',
    'url': 'https://example.com/checkout',
    'timestamp': datetime.utcnow().isoformat() + 'Z',
    'properties': {
        'amount': 99.99,
        'currency': 'USD'
    }
})

# Get analytics
response = requests.get('http://localhost:8080/api/stats', params={
    'start': '2024-01-01',
    'end': '2024-01-31',
    'project': 'my-store'
})
stats = response.json()
```

### Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "time"
)

type Event struct {
    EventName string    `json:"event_name"`
    UserID    string    `json:"user_id"`
    ProjectID string    `json:"project_id"`
    URL       string    `json:"url"`
    Timestamp time.Time `json:"timestamp"`
}

func trackEvent() error {
    event := Event{
        EventName: "page_view",
        UserID:    "user-123",
        ProjectID: "my-website",
        URL:       "https://example.com",
        Timestamp: time.Now(),
    }
    
    data, _ := json.Marshal(event)
    _, err := http.Post(
        "http://localhost:8080/api/track",
        "application/json",
        bytes.NewBuffer(data),
    )
    return err
}
```

## Performance Considerations

- Use batch endpoint (`/api/track/batch`) for multiple events
- Implement client-side buffering (SDK handles this automatically)
- Query endpoints are optimized with DuckDB columnar storage
- Typical response times: < 50ms for tracking, < 200ms for analytics queries
- Data is stored in Parquet format for efficient querying

## Next Steps

- [SDK Integration Guide →](/sdk/overview)
- [Channel Analytics →](/guide/channels)
- [Funnel Analysis →](/guide/funnels)
