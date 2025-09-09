package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/model"
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

// Новый метод для отправки через JSON
func (s *Sender) SendJSON(metrics map[string]model.Metrics) error {
	for _, metric := range metrics {
		// Пропускаем метрики без значений
		if (metric.MType == model.Gauge && metric.Value == nil) ||
			(metric.MType == model.Counter && metric.Delta == nil) {
			continue
		}

		// Подготавливаем JSON
		jsonData, err := json.Marshal(metric)
		if err != nil {
			log.Printf("ERROR: failed to marshal metric %s: %v", metric.ID, err)
			continue
		}

		// Сжимаем данные
		var compressedData bytes.Buffer
		gz := gzip.NewWriter(&compressedData)
		if _, err := gz.Write(jsonData); err != nil {
			log.Printf("ERROR: failed to compress data: %v", err)
			continue
		}
		if err := gz.Close(); err != nil {
			log.Printf("ERROR: failed to close gzip writer: %v", err)
			continue
		}

		// Создаем запрос
		url := "http://" + s.ServerURL + "/update/"
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("FAILED to create request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Content-Encoding", "gzip")
		request.Header.Set("Accept-Encoding", "gzip")

		// Отправляем
		response, err := s.Client.Do(request)
		if err != nil {
			return fmt.Errorf("FAILED to send metric %s: %w", metric.ID, err)
		}
		defer response.Body.Close()

		// Читаем тело ответа для диагностики
		body, _ := io.ReadAll(response.Body)

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("FAIL status for %s: %d, body: %s", metric.ID, response.StatusCode, string(body))
		}
	}
	return nil
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
