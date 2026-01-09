package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/shatrunoff/yap_metrics/internal/crypto"
	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/shatrunoff/yap_metrics/internal/utils"
)

type Sender struct {
	ServerURL string
	Client    *http.Client
	Key       string
	PublicKey *rsa.PublicKey
}

func NewSender(ServerURL string, key string) *Sender {
	return &Sender{
		ServerURL: ServerURL,
		Key:       key,
		Client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

func NewSenderWithCrypto(ServerURL string, key string, publicKeyPath string) (*Sender, error) {
	sender := &Sender{
		ServerURL: ServerURL,
		Key:       key,
		Client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}

	if publicKeyPath != "" {
		publicKey, err := crypto.LoadPublicKey(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key: %w", err)
		}
		sender.PublicKey = publicKey
	}

	return sender, nil
}

// sendRequest отправляет gzip-сжатые JSON-данные на указанный URL
// и при наличии ключа добавляет HMAC-SHA256 подпись исходного JSON.
func (s *Sender) sendRequest(url string, jsonData []byte) error {
	var dataToSend []byte
	var err error

	// Шифруем данные если есть публичный ключ
	if s.PublicKey != nil {
		dataToSend, err = crypto.Encrypt(jsonData, s.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %w", err)
		}
	} else {
		compressedData, err := compressData(jsonData)
		if err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
		dataToSend = compressedData
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(dataToSend))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if s.PublicKey != nil {
		req.Header.Set("Content-Encoding", "rsa")
	} else {
		req.Header.Set("Content-Encoding", "gzip")
	}

	req.Header.Set("Accept-Encoding", "gzip")

	if s.Key != "" {
		sig := utils.ComputeHMACSHA256(jsonData, s.Key)
		req.Header.Set("HashSHA256", sig)
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
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

// SendMetricJSON отправляет одну метрику в формате JSON c gzip
// и опциональной подписью HMAC (заголовок HashSHA256)
func (s *Sender) SendMetricJSON(metric model.Metrics) error {
	// Пропускаем метрики без значений
	if (metric.MType == model.Gauge && metric.Value == nil) ||
		(metric.MType == model.Counter && metric.Delta == nil) {
		return nil
	}

	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric %s: %w", metric.ID, err)
	}

	url := "http://" + s.ServerURL + "/update/"
	if err := s.sendRequest(url, jsonData); err != nil {
		return fmt.Errorf("FAILED to send metric %s: %w", metric.ID, err)
	}
	return nil
}

// SendMetricWithRetry — обёртка над SendMetricJSON с ретраями
// для сетевых ошибок
func (s *Sender) SendMetricWithRetry(metric model.Metrics) error {
	return utils.RetryNetworkNoCtx("SendMetric", func(m model.Metrics) error {
		return s.SendMetricJSON(m)
	}, metric)
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

		url := "http://" + s.ServerURL + "/update/"
		if err := s.sendRequest(url, jsonData); err != nil {
			return fmt.Errorf("FAILED to send metric %s: %w", metric.ID, err)
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
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("FAIL status for %s: %d", metric.ID, response.StatusCode)
		}
	}
	return nil
}

// обертка с retry для SendBatch
func (s *Sender) SendBatchWithRetry(metrics []model.Metrics) error {
	return utils.RetryNetworkNoCtx("SendBatch", func(metrics []model.Metrics) error {
		return s.SendBatch(metrics)
	}, metrics)
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

	url := "http://" + s.ServerURL + "/updates/"
	if err := s.sendRequest(url, jsonData); err != nil {
		return fmt.Errorf("failed to send metrics batch: %w", err)
	}
	return nil
}
