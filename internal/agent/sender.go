package agent

import (
	"fmt"
	"net/http"
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
			Timeout: time.Duration(NewMetricsCollector().PollCount),
		},
	}
}

func (s *Sender) Send(metrics map[string]model.Metrics) error {
	// формируем URL
	URL := "http://" + s.ServerURL
	for _, metric := range metrics {
		url := fmt.Sprintf(
			"%s/update/%s/%s/",
			URL,
			metric.MType,
			metric.ID,
		)

		var strValue string
		switch metric.MType {
		case model.Gauge:
			if metric.Value == nil {
				continue
			}
			strValue = strconv.FormatFloat(*metric.Value, 'f', -2, 64)
		case model.Counter:
			if metric.Delta == nil {
				continue
			}
			strValue = strconv.FormatInt(*metric.Delta, 10)
		default:
			continue
		}

		url += strValue

		// запрос
		request, err := http.NewRequest("POST", url, nil)
		if err != nil {
			return fmt.Errorf("FAILED to create request: %w", err)
		}
		request.Header.Set("Content-Type", "text/plain")

		// ответ
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
