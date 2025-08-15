package agent

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	model "github.com/shatrunoff/yap_metrics/internal/model"
)

type Sender struct {
	ServerURL string
	Client    *http.Client
}

func NewSender(ServerURL string) *Sender {
	return &Sender{
		ServerURL: ServerURL,
		Client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

func newMetricURL(baseURL, metricType, metricID, value string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("ERROR: invalid base URL %w", err)
	}

	u.Path = path.Join(u.Path, "update", metricType, metricID, value)
	return u.String(), nil
}

func (s *Sender) Send(metrics map[string]model.Metrics) error {

	for _, metric := range metrics {
		// парсим значение метрики в строку
		var strValue string
		switch metric.MType {
		case model.Gauge:
			if metric.Value == nil {
				continue
			}
			strValue = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		case model.Counter:
			if metric.Delta == nil {
				continue
			}
			strValue = strconv.FormatInt(*metric.Delta, 10)
		default:
			continue
		}

		// полный URL
		url, err := newMetricURL(
			"http://"+s.ServerURL,
			metric.MType,
			metric.ID,
			strValue,
		)
		if err != nil {
			log.Printf("ERROR: failed to create URL %v", err)
		}

		// POST-запрос
		request, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return fmt.Errorf("FAILED to create request: %w", err)
		}
		request.Header.Set("Content-Type", "text/plain")

		// отправляем
		response, err := s.Client.Do(request)
		if err != nil {
			return fmt.Errorf("FAILED to send metric %s: %w", metric.ID, err)
		}
		response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("FAIL status for %s: %d", metric.ID, response.StatusCode)
		}
	}
	return nil
}
