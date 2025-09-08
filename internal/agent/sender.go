package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	UseJSON   bool
}

func NewSender(ServerURL string, useJSON bool) *Sender {
	return &Sender{
		ServerURL: ServerURL,
		Client: &http.Client{
			Timeout: 4 * time.Second,
		},
		UseJSON: useJSON,
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

func newJSONMetricURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("ERROR: invalid base URL %w", err)
	}

	u.Path = path.Join(u.Path, "update")
	return u.String(), nil
}

func (s *Sender) Send(metrics map[string]model.Metrics) error {
	for _, metric := range metrics {
		var err error

		if s.UseJSON {
			err = s.sendJSON(metric)
		} else {
			err = s.sendURLParams(metric)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// отправка в старом формате (URL параметры)
func (s *Sender) sendURLParams(metric model.Metrics) error {
	// парсим значение метрики в строку
	var strValue string
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return nil
		}
		strValue = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
	case model.Counter:
		if metric.Delta == nil {
			return nil
		}
		strValue = strconv.FormatInt(*metric.Delta, 10)
	default:
		return nil
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
		return err
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
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("FAIL status for %s: %d", metric.ID, response.StatusCode)
	}

	return nil
}

// отправка в новом JSON формате
func (s *Sender) sendJSON(metric model.Metrics) error {
	// пропускаем метрики без значений
	if (metric.MType == model.Gauge && metric.Value == nil) ||
		(metric.MType == model.Counter && metric.Delta == nil) {
		return nil
	}

	// подготавливаем URL
	url, err := newJSONMetricURL("http://" + s.ServerURL)
	if err != nil {
		log.Printf("ERROR: failed to create URL %v", err)
		return err
	}

	// сериализуем метрику в JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("FAILED to marshal JSON for %s: %w", metric.ID, err)
	}

	// создаем запрос с JSON телом
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("FAILED to create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// отправляем запрос
	response, err := s.Client.Do(request)
	if err != nil {
		return fmt.Errorf("FAILED to send metric %s: %w", metric.ID, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("FAIL status for %s: %d", metric.ID, response.StatusCode)
	}

	return nil
}
