package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shatrunoff/yap_metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL[7:], "") // убираем http://
	metrics := map[string]model.Metrics{
		"test_gauge": {
			ID:    "test_gauge",
			MType: model.Gauge,
			Value: float64Ptr(42.5),
		},
		"test_counter": {
			ID:    "test_counter",
			MType: model.Counter,
			Delta: int64Ptr(100),
		},
	}

	err := sender.Send(metrics)
	assert.NoError(t, err)
}

func TestSendJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		gr, err := gzip.NewReader(bytes.NewReader(body))
		assert.NoError(t, err)
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		assert.NoError(t, err)

		var metric model.Metrics
		err = json.Unmarshal(decompressed, &metric)
		assert.NoError(t, err)

		assert.Equal(t, "test_metric", metric.ID)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL[7:], "")
	metrics := map[string]model.Metrics{
		"test_metric": {
			ID:    "test_metric",
			MType: model.Gauge,
			Value: float64Ptr(42.5),
		},
	}

	err := sender.SendJSON(metrics)
	assert.NoError(t, err)
}

func TestSendBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		gr, err := gzip.NewReader(bytes.NewReader(body))
		assert.NoError(t, err)
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		assert.NoError(t, err)

		var metrics []model.Metrics
		err = json.Unmarshal(decompressed, &metrics)
		assert.NoError(t, err)

		assert.Len(t, metrics, 2)
		assert.Equal(t, "batch_gauge", metrics[0].ID)
		assert.Equal(t, "batch_counter", metrics[1].ID)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL[7:], "")
	batch := []model.Metrics{
		{
			ID:    "batch_gauge",
			MType: model.Gauge,
			Value: float64Ptr(42.5),
		},
		{
			ID:    "batch_counter",
			MType: model.Counter,
			Delta: int64Ptr(100),
		},
	}

	err := sender.SendBatch(batch)
	assert.NoError(t, err)
}

func TestSendMetricJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		gr, err := gzip.NewReader(bytes.NewReader(body))
		assert.NoError(t, err)
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		assert.NoError(t, err)

		var metric model.Metrics
		err = json.Unmarshal(decompressed, &metric)
		assert.NoError(t, err)

		assert.Equal(t, "single_metric", metric.ID)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL[7:], "")
	metric := model.Metrics{
		ID:    "single_metric",
		MType: model.Counter,
		Delta: int64Ptr(1),
	}

	err := sender.SendMetricJSON(metric)
	assert.NoError(t, err)
}

func TestSendWithHMAC(t *testing.T) {
	key := "test-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hashHeader := r.Header.Get("HashSHA256")
		assert.NotEmpty(t, hashHeader)

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		// decompress
		gr, err := gzip.NewReader(bytes.NewReader(body))
		assert.NoError(t, err)
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		assert.NoError(t, err)

		expectedSig := computeHMAC(decompressed, key)
		assert.Equal(t, expectedSig, hashHeader)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewSender(server.URL[7:], key)
	metric := model.Metrics{
		ID:    "hmac_metric",
		MType: model.Counter,
		Delta: int64Ptr(5),
	}

	err := sender.SendMetricJSON(metric)
	assert.NoError(t, err)
}

func TestSendMetricWithoutValue(t *testing.T) {
	sender := NewSender("localhost:8080", "")

	metricGauge := model.Metrics{
		ID:    "empty_gauge",
		MType: model.Gauge,
		Value: nil,
	}
	err := sender.SendMetricJSON(metricGauge)
	assert.NoError(t, err) // должна быть пропущена без ошибки

	metricCounter := model.Metrics{
		ID:    "empty_counter",
		MType: model.Counter,
		Delta: nil,
	}
	err = sender.SendMetricJSON(metricCounter)
	assert.NoError(t, err) // должна быть пропущена без ошибки
}

func TestCompressData(t *testing.T) {
	data := []byte(`{"id":"test","type":"gauge","value":42.0}`)
	compressed, err := compressData(data)
	assert.NoError(t, err)

	// Для маленьких данных сжатие может не дать выигрыша
	// Проверим, что данные можно распаковать обратно
	gr, err := gzip.NewReader(bytes.NewReader(compressed))
	assert.NoError(t, err)
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	assert.NoError(t, err)
	assert.Equal(t, data, decompressed)
}

func TestNewMetricURL(t *testing.T) {
	base := "http://localhost:8080"
	url, err := newMetricURL(base, model.Counter, "my_counter", "100")
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/update/counter/my_counter/100", url)
}

func TestNewMetricURLError(t *testing.T) {
	_, err := newMetricURL(":", model.Counter, "my_counter", "100")
	assert.Error(t, err)
}

func float64Ptr(f float64) *float64 {
	return &f
}

func int64Ptr(i int64) *int64 {
	return &i
}

func computeHMAC(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}
