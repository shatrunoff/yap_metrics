package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/shatrunoff/yap_metrics/internal/model"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handlerUpdate)
	return mux
}

func handlerUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// парсинг URL
	splitURL := strings.Split(r.URL.Path, "/")
	if len(splitURL) != 5 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	metricType := splitURL[2]
	metricName := splitURL[3]
	metricValue := splitURL[4]

	// обработка метрик
	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Printf("Save gaude metric %s=%f\n", metricName, value)

	case model.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fmt.Printf("Save counter metric %s=%d\n", metricName, value)

	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
