package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mohamedelhefni/siraaj/geolocation"
	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewEventHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockEventService(ctrl)
	handler := NewEventHandler(mockService, nil)

	if handler.service == nil {
		t.Error("Expected handler service to be set")
	}
}

func TestTrackEvent(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           any
		setupMock      func(*mocks.MockEventService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:   "Successful event tracking",
			method: http.MethodPost,
			body: domain.Event{
				EventName: "page_view",
				UserID:    "user123",
				URL:       "/home",
			},
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					TrackEvent(gomock.Any()).
					Return(nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			body:           nil,
			setupMock:      func(m *mocks.MockEventService) {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			body:           "invalid json",
			setupMock:      func(m *mocks.MockEventService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON",
		},
		{
			name:   "Service error",
			method: http.MethodPost,
			body: domain.Event{
				EventName: "click",
				UserID:    "user456",
			},
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					TrackEvent(gomock.Any()).
					Return(errors.New("database error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockEventService(ctrl)
			tt.setupMock(mockService)

			handler := NewEventHandler(mockService, nil)

			var body []byte
			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body = []byte(str)
				} else {
					body, _ = json.Marshal(tt.body)
				}
			}

			req := httptest.NewRequest(tt.method, "/track", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.TrackEvent(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !bytes.Contains(w.Body.Bytes(), []byte(tt.expectedBody)) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestTrackEventWithGeolocation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockEventService(ctrl)

	// Create a geolocation service (ignore error for test)
	geoService, _ := geolocation.NewService()

	handler := NewEventHandler(mockService, geoService)
	event := domain.Event{
		EventName: "page_view",
		UserID:    "user123",
		IP:        "8.8.8.8",
	}

	mockService.EXPECT().
		TrackEvent(gomock.Any()).
		DoAndReturn(func(e domain.Event) error {
			// Verify geolocation was enriched
			if e.Country == "" {
				t.Log("Warning: Geolocation enrichment may not have worked (this is expected if GeoIP DB is unavailable)")
			}
			return nil
		}).
		Times(1)

	body, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, "/track", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.TrackEvent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestGetStats(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*mocks.MockEventService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]any)
	}{
		{
			name:        "Default date range (last 7 days)",
			queryParams: "",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetStats(gomock.Any(), gomock.Any(), 50, gomock.Any()).
					Return(map[string]any{
						"total_events": 1000,
						"unique_users": 250,
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]any) {
				if total, ok := resp["total_events"].(float64); !ok || total != 1000 {
					t.Errorf("Expected total_events to be 1000, got %v", resp["total_events"])
				}
			},
		},
		{
			name:        "Custom date range",
			queryParams: "?start=2024-01-01&end=2024-01-31",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetStats(gomock.Any(), gomock.Any(), 50, gomock.Any()).
					Return(map[string]any{
						"total_events": 500,
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]any) {
				if total, ok := resp["total_events"].(float64); !ok || total != 500 {
					t.Errorf("Expected total_events to be 500, got %v", resp["total_events"])
				}
			},
		},
		{
			name:        "With filters",
			queryParams: "?project=myapp&country=Palestine&browser=Chrome",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetStats(gomock.Any(), gomock.Any(), 50, gomock.Any()).
					DoAndReturn(func(start, end time.Time, limit int, filters map[string]string) (map[string]any, error) {
						if filters["project"] != "myapp" {
							t.Error("Expected project filter to be 'myapp'")
						}
						if filters["country"] != "Palestine" {
							t.Error("Expected country filter to be 'Palestine'")
						}
						if filters["browser"] != "Chrome" {
							t.Error("Expected browser filter to be 'Chrome'")
						}
						return map[string]any{"total_events": 100}, nil
					}).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  func(t *testing.T, resp map[string]any) {},
		},
		{
			name:        "Service error",
			queryParams: "",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetStats(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse:  nil,
		},
		{
			name:        "Custom limit",
			queryParams: "?limit=100",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetStats(gomock.Any(), gomock.Any(), 100, gomock.Any()).
					Return(map[string]any{"total_events": 200}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  func(t *testing.T, resp map[string]any) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockEventService(ctrl)
			tt.setupMock(mockService)

			handler := NewEventHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/stats"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil && w.Code == http.StatusOK {
				var resp map[string]any
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestGetEvents(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*mocks.MockEventService)
		expectedStatus int
	}{
		{
			name:        "Default parameters",
			queryParams: "",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetEvents(gomock.Any(), gomock.Any(), 100, 0).
					Return(map[string]any{
						"events": []any{},
						"total":  0,
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "With pagination",
			queryParams: "?limit=50&offset=100",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetEvents(gomock.Any(), gomock.Any(), 50, 100).
					Return(map[string]any{
						"events": []any{},
						"total":  0,
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Service error",
			queryParams: "",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetEvents(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockEventService(ctrl)
			tt.setupMock(mockService)

			handler := NewEventHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/events"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.GetEvents(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetOnlineUsers(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(*mocks.MockEventService)
		expectedStatus int
	}{
		{
			name:        "Default time window",
			queryParams: "",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetOnlineUsers(5).
					Return(map[string]any{
						"online_users": 42,
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Custom time window",
			queryParams: "?window=10",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetOnlineUsers(10).
					Return(map[string]any{
						"online_users": 50,
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Service error",
			queryParams: "",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetOnlineUsers(gomock.Any()).
					Return(nil, errors.New("error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockEventService(ctrl)
			tt.setupMock(mockService)

			handler := NewEventHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/online"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.GetOnlineUsers(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestGetProjects(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockEventService)
		expectedStatus int
		expectedBody   []string
	}{
		{
			name: "Success",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetProjects().
					Return([]string{"project1", "project2"}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"project1", "project2"},
		},
		{
			name: "Service error",
			setupMock: func(m *mocks.MockEventService) {
				m.EXPECT().
					GetProjects().
					Return(nil, errors.New("error")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockService := mocks.NewMockEventService(ctrl)
			tt.setupMock(mockService)

			handler := NewEventHandler(mockService, nil)

			req := httptest.NewRequest(http.MethodGet, "/projects", nil)
			w := httptest.NewRecorder()

			handler.GetProjects(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != nil {
				var resp []string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if len(resp) != len(tt.expectedBody) {
					t.Errorf("Expected %d projects, got %d", len(tt.expectedBody), len(resp))
				}
			}
		})
	}
}

func TestHealth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockEventService(ctrl)
	geoService, _ := geolocation.NewService()
	tests := []struct {
		name       string
		geoService *geolocation.Service
		expectGeo  bool
	}{
		{
			name:       "Without geolocation",
			geoService: nil,
			expectGeo:  false,
		},
		{
			name:       "With geolocation",
			geoService: geoService,
			expectGeo:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewEventHandler(mockService, tt.geoService)

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			handler.Health(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			var resp map[string]any
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if status, ok := resp["status"].(string); !ok || status != "ok" {
				t.Error("Expected status to be 'ok'")
			}

			if geo, ok := resp["geolocation"].(bool); !ok || geo != tt.expectGeo {
				t.Errorf("Expected geolocation to be %v, got %v", tt.expectGeo, geo)
			}
		})
	}
}

func TestGeoTest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mocks.NewMockEventService(ctrl)

	t.Run("Without geolocation service", func(t *testing.T) {
		handler := NewEventHandler(mockService, nil)

		req := httptest.NewRequest(http.MethodGet, "/geotest", nil)
		w := httptest.NewRecorder()

		handler.GeoTest(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})

	t.Run("With geolocation service", func(t *testing.T) {
		geoService, _ := geolocation.NewService()
		handler := NewEventHandler(mockService, geoService)

		req := httptest.NewRequest(http.MethodGet, "/geotest?ip=8.8.8.8", nil)
		w := httptest.NewRecorder()

		handler.GeoTest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if ip, ok := resp["ip"].(string); !ok || ip != "8.8.8.8" {
			t.Errorf("Expected ip to be '8.8.8.8', got %v", resp["ip"])
		}
	})

	t.Run("Default to client IP", func(t *testing.T) {
		geoService, _ := geolocation.NewService()
		handler := NewEventHandler(mockService, geoService)

		req := httptest.NewRequest(http.MethodGet, "/geotest", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		handler.GeoTest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name: "X-Forwarded-For header",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 198.51.100.1",
			},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name: "X-Real-IP header",
			headers: map[string]string{
				"X-Real-IP": "203.0.113.2",
			},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.2",
		},
		{
			name:       "Remote address fallback",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
				"X-Real-IP":       "203.0.113.2",
			},
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			req.RemoteAddr = tt.remoteAddr

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
