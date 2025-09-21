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

// compressData сжимает данные с помощью gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// метод для отправки через JSON с поддержкой gzip
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
		compressedData, err := compressData(jsonData)
		if err != nil {
			log.Printf("ERROR: failed to compress data for metric %s: %v", metric.ID, err)
			continue
		}

		// Создаем запрос
		url := "http://" + s.ServerURL + "/update/"
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedData))
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
		request.Header.Set("Accept-Encoding", "gzip")

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

// отправляет метрики батчами через JSON с поддержкой gzip
func (s *Sender) SendBatch(metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	// Фильтруем метрики без значений
	validMetrics := make([]model.Metrics, 0, len(metrics))
	for _, metric := range metrics {
		if (metric.MType == model.Gauge && metric.Value != nil) ||
			(metric.MType == model.Counter && metric.Delta != nil) {
			validMetrics = append(validMetrics, metric)
		}
	}

	if len(validMetrics) == 0 {
		return nil
	}

	// Подготавливаем JSON
	jsonData, err := json.Marshal(validMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics batch: %w", err)
	}

	// Сжимаем данные
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("failed to compress batch data: %w", err)
	}

	// Создаем запрос
	url := "http://" + s.ServerURL + "/updates/"
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedData))
	if err != nil {
		return fmt.Errorf("failed to create batch request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Accept-Encoding", "gzip")

	// Отправляем
	response, err := s.Client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to send metrics batch: %w", err)
	}
	defer response.Body.Close()

	// Читаем тело ответа для диагностики
	body, _ := io.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("batch request failed: status %d, body: %s", response.StatusCode, string(body))
	}

	return nil
}
