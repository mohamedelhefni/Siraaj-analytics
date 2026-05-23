package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

func BenchmarkEventCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = domain.Event{
			Timestamp:       time.Now(),
			EventName:       "benchmark_event",
			UserID:          fmt.Sprintf("user_%d", i),
			SessionID:       fmt.Sprintf("session_%d", i),
			SessionDuration: 120,
			URL:             "/benchmark",
			Referrer:        "https://benchmark.com",
			UserAgent:       "BenchmarkAgent/1.0",
			IP:              "192.168.1.1",
			Country:         "Palestine",
			Browser:         "Chrome",
			OS:              "Linux",
			Device:          "Desktop",
			IsBot:           false,
			ProjectID:       "benchmark",
		}
	}
}

func BenchmarkEventBatchCreation(b *testing.B) {
	b.ReportAllocs()

	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize-%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				events := make([]domain.Event, size)
				for j := range size {
					events[j] = domain.Event{
						Timestamp: time.Now(),
						EventName: "batch_benchmark",
						UserID:    fmt.Sprintf("user_%d", j),
						SessionID: fmt.Sprintf("session_%d", j),
						URL:       "/benchmark",
						ProjectID: "benchmark",
					}
				}
			}
		})
	}
}

func BenchmarkStatsCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = domain.Stats{
			PageViews:      1000,
			UniqueVisitors: 500,
			UniqueUsers:    450,
			TopPages:       make([]domain.PageStat, 10),
			TopReferrers:   make([]domain.ReferrerStat, 10),
			Countries:      make([]domain.CountryStat, 10),
			Browsers:       make([]domain.BrowserStat, 5),
			OSList:         make([]domain.OSStat, 5),
			Devices:        make([]domain.DeviceStat, 3),
			Timeline:       make([]domain.TimelineStat, 30),
			Events:         make(map[string]int64),
		}
	}
}

// BenchmarkJSONPropertyParsing removed - properties feature deprecated
