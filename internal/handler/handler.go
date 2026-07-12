// Package handler содержит HTTP-обработчики сервера метрик:
// приём обновлений, чтение значений, ping БД и HTML-индекс.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Radiushina/metrics-receiver.git/internal/audit"
	"github.com/Radiushina/metrics-receiver.git/internal/logger"
	models "github.com/Radiushina/metrics-receiver.git/internal/model"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type (
	// Handler связывает HTTP-обработчики со слоем сервиса.
	Handler struct {
		service   ServiceProvider
		saver     Saver
		log       *zap.Logger
		db        DBChecker
		secretKey string
		audit     AuditPublisher
	}

	// AuditPublisher рассылает события аудита после успешного приёма метрик.
	AuditPublisher interface {
		Enabled() bool
		Notify(ctx context.Context, e audit.Event)
	}

	// ServiceProvider описывает операции сервиса, которые нужны Handler.
	ServiceProvider interface {
		SetGauge(ctx context.Context, name string, value float64) error
		AddCounter(ctx context.Context, name string, delta int64) error
		GetGauge(ctx context.Context, name string) (float64, bool)
		GetCounter(ctx context.Context, name string) (int64, bool)
		Gauges(ctx context.Context) map[string]float64
		Counters(ctx context.Context) map[string]int64
		UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error
	}

	// Saver сохраняет метрики после обновлений, если сохранение включено.
	Saver interface {
		Save(ctx context.Context) error
	}

	// DBChecker проверяет доступность БД (например *pgxpool.Pool).
	DBChecker interface {
		Ping(ctx context.Context) error
	}
)

// NewHandler создаёт Handler с указанным сервисом, опциональным Saver, логгером
// и опциональной проверкой БД (db может быть nil, если DSN не задан).
// audit может быть nil или без подписчиков — тогда аудит отключаем.
func NewHandler(
	service ServiceProvider,
	saver Saver,
	log *zap.Logger,
	db DBChecker,
	secretKey string,
	auditPub AuditPublisher,
) *Handler {
	log = logger.OrNop(log)
	return &Handler{
		service:   service,
		saver:     saver,
		log:       log,
		db:        db,
		secretKey: secretKey,
		audit:     auditPub,
	}
}

// GetMetrics возвращает HTTP-обработчик, который записывает в ответ
// все собранные метрики.
func (h *Handler) GetMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeMetricsIndex(r.Context(), w, h.service, h.log)
	}
}

// PingDB возвращает HTTP-обработчик, который проверяет соединение с БД.
func (h *Handler) PingDB() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.db == nil {
			http.Error(w, "database not configured", http.StatusInternalServerError)
			return
		}
		if err := h.db.Ping(r.Context()); err != nil {
			h.log.Error("db ping", zap.Error(err))
			http.Error(w, "database unavailable", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetMetric возвращает HTTP-обработчик, который записывает значение
// одной метрики в виде простого текста.
func (h *Handler) GetMetric() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getMetricValue(w, r, h.service, h.log)
	}
}

// GetMetricValue возвращает HTTP-обработчик, который записывает значение
// одной метрики в формате JSON.
func (h *Handler) GetMetricValue() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		getMetric(w, r, h.service, h.log)
	}
}

// UpdateFromPath возвращает HTTP-обработчик, который обновляет метрику
// по параметрам пути URL.
func (h *Handler) UpdateFromPath() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateMetricsFromPath(w, r, h.service, h.saver, h.log, h.secretKey, h.audit)
	}
}

// UpdateFromBody возвращает HTTP-обработчик, который обновляет метрику
// из JSON-тела запроса.
func (h *Handler) UpdateFromBody() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateMetricsFromBody(w, r, h.service, h.saver, h.log, h.secretKey, h.audit)
	}
}

// UpdateMetrics возвращает HTTP-обработчик, который обновляет метрики батчем
// из JSON-массива в теле запроса (POST /updates/)
func (h *Handler) UpdateMetrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		updateMetricsBatch(w, r, h.service, h.saver, h.log, h.secretKey, h.audit)
	}
}

func notifyAudit(auditPub AuditPublisher, r *http.Request, metrics []string) {
	if auditPub == nil || !auditPub.Enabled() {
		return
	}
	auditPub.Notify(r.Context(), audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: clientIP(r),
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func updateMetricsFromPath(
	w http.ResponseWriter,
	r *http.Request,
	service ServiceProvider,
	saver Saver,
	log *zap.Logger,
	secretKey string,
	auditPub AuditPublisher,
) {
	if err := verifyPathUpdateRequestHash(r, secretKey); err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, err.Error())
		return
	}

	mtype := models.MetricType(strings.ToLower(chi.URLParam(r, "mtype")))
	metric := chi.URLParam(r, "metric")
	valueStr := chi.URLParam(r, "value")

	if metric == "" {
		http.NotFound(w, r)
		return
	}

	if valueStr == "" {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, "missing metric value")
		return
	}

	if mtype == "" {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, "missing metric type")
		return
	}

	switch mtype {
	case models.Gauge:
		v, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			writeProtectedPlainError(w, secretKey, http.StatusBadRequest, "invalid gauge value")
			return
		}
		if err := service.SetGauge(r.Context(), metric, v); err != nil {
			log.Error("set gauge", zap.Error(err))
			writeProtectedPlainError(w, secretKey, http.StatusInternalServerError, "internal server error")
			return
		}
		log.Sugar().Infof("server: gauge %s = %g", metric, v)
	case models.Counter:
		v, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			writeProtectedPlainError(w, secretKey, http.StatusBadRequest, "invalid counter value")
			return
		}
		if err := service.AddCounter(r.Context(), metric, v); err != nil {
			log.Error("add counter", zap.Error(err))
			writeProtectedPlainError(w, secretKey, http.StatusInternalServerError, "internal server error")
			return
		}
		log.Sugar().Infof("server: counter %s += %d", metric, v)
	default:
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, fmt.Sprintf("invalid metric type: %q", mtype))
		return
	}

	if saver != nil {
		if err := saver.Save(r.Context()); err != nil {
			writeProtectedPlainError(w, secretKey, http.StatusInternalServerError, "failed to persist metrics")
			return
		}
	}

	notifyAudit(auditPub, r, []string{metric})

	writeProtectedEmptyOK(w, secretKey)
}

func updateMetricsFromBody(
	w http.ResponseWriter,
	r *http.Request,
	service ServiceProvider,
	saver Saver,
	log *zap.Logger,
	secretKey string,
	auditPub AuditPublisher,
) {
	defer func() { _ = r.Body.Close() }()

	reqBody, errRead := readRequestBody(r)
	if errRead != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, errRead.Error())
		return
	}
	if err := verifyRequestBodyHashSHA256(secretKey, r, reqBody); err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, err.Error())
		return
	}
	logRequestBody(log, "metrics update request body", reqBody)

	in, err := unmarshalMetric(reqBody)
	if err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, err.Error())
		return
	}

	out, status, err := applyMetricUpdate(r.Context(), log, service, in)
	if err != nil {
		writeProtectedPlainError(w, secretKey, status, err.Error())
		return
	}

	if err := persistIfEnabled(r.Context(), saver); err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusInternalServerError, "failed to persist metrics")
		return
	}

	notifyAudit(auditPub, r, []string{in.ID})

	writeJSON(w, secretKey, out)
}

func updateMetricsBatch(
	w http.ResponseWriter,
	r *http.Request,
	service ServiceProvider,
	saver Saver,
	log *zap.Logger,
	secretKey string,
	auditPub AuditPublisher,
) {
	defer func() { _ = r.Body.Close() }()

	reqBody, errRead := readRequestBody(r)
	if errRead != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, errRead.Error())
		return
	}
	if err := verifyRequestBodyHashSHA256(secretKey, r, reqBody); err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, err.Error())
		return
	}
	logRequestBody(log, "metrics update request body", reqBody)

	in, err := unmarshalMetrics(reqBody)
	if err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateMetricsBatch(in); err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusBadRequest, err.Error())
		return
	}

	if err := service.UpdateMetricsBatch(r.Context(), in); err != nil {
		log.Error("update metrics batch", zap.Error(err))
		writeProtectedPlainError(w, secretKey, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := persistIfEnabled(r.Context(), saver); err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusInternalServerError, "failed to persist metrics")
		return
	}

	names := make([]string, len(in))
	for i, m := range in {
		names[i] = m.ID
	}
	notifyAudit(auditPub, r, names)

	writeJSON(w, secretKey, in)
}

func validateMetricsBatch(metrics []models.Metrics) error {
	for _, m := range metrics {
		if strings.TrimSpace(m.ID) == "" {
			return errors.New("missing metric id")
		}
		if m.MType == "" {
			return errors.New("missing metric type")
		}
		switch m.MType {
		case models.Gauge:
			if m.Value == nil || m.Delta != nil {
				return errors.New("gauge metric requires value, delta must be omitted")
			}
		case models.Counter:
			if m.Delta == nil || m.Value != nil {
				return errors.New("counter metric requires delta, value must be omitted")
			}
		default:
			return fmt.Errorf("invalid metric type: %q", m.MType)
		}
	}
	return nil
}

func readRequestBody(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("failed to read body")
	}
	return b, nil
}

func logRequestBody(log *zap.Logger, msg string, b []byte) {
	log.Info(msg, zap.String("body", string(b)))
}

func unmarshalMetric(b []byte) (models.Metrics, error) {
	var m models.Metrics
	if err := json.Unmarshal(b, &m); err != nil {
		return models.Metrics{}, errors.New("invalid JSON")
	}
	if strings.TrimSpace(m.ID) == "" {
		return models.Metrics{}, errors.New("missing metric id")
	}
	m.MType = models.MetricType(strings.ToLower(string(m.MType)))
	if m.MType == "" {
		return models.Metrics{}, errors.New("missing metric type")
	}
	return m, nil
}

func unmarshalMetrics(b []byte) ([]models.Metrics, error) {
	var m []models.Metrics
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, errors.New("invalid JSON")
	}
	for i := range m {
		m[i].MType = models.MetricType(strings.ToLower(string(m[i].MType)))
	}
	return m, nil
}

func applyMetricUpdate(
	ctx context.Context,
	log *zap.Logger,
	service ServiceProvider,
	in models.Metrics,
) (out models.Metrics, statusCode int, err error) {
	switch in.MType {
	case models.Counter:
		return applyCounterUpdate(ctx, log, service, in)
	case models.Gauge:
		return applyGaugeUpdate(ctx, log, service, in)
	default:
		return models.Metrics{}, http.StatusBadRequest, fmt.Errorf("invalid metric type: %q", in.MType)
	}
}

func applyCounterUpdate(
	ctx context.Context,
	log *zap.Logger,
	service ServiceProvider,
	in models.Metrics,
) (out models.Metrics, statusCode int, err error) {
	if in.Delta == nil {
		return models.Metrics{}, http.StatusBadRequest, errors.New("missing counter delta")
	}

	if err := service.AddCounter(ctx, in.ID, *in.Delta); err != nil {
		log.Error("add counter", zap.Error(err))
		return models.Metrics{}, http.StatusInternalServerError, errors.New("internal server error")
	}
	log.Sugar().Infof("server: counter %s += %d", in.ID, *in.Delta)

	total, ok := service.GetCounter(ctx, in.ID)
	if !ok {
		return models.Metrics{}, http.StatusInternalServerError, errors.New("internal server error")
	}

	return models.Metrics{
		ID:    in.ID,
		MType: models.Counter,
		Delta: &total,
	}, http.StatusOK, nil
}

func applyGaugeUpdate(
	ctx context.Context,
	log *zap.Logger,
	service ServiceProvider,
	in models.Metrics,
) (out models.Metrics, statusCode int, err error) {
	if in.Value == nil {
		return models.Metrics{}, http.StatusBadRequest, errors.New("missing gauge value")
	}

	if err := service.SetGauge(ctx, in.ID, *in.Value); err != nil {
		log.Error("set gauge", zap.Error(err))
		return models.Metrics{}, http.StatusInternalServerError, errors.New("internal server error")
	}
	log.Sugar().Infof("server: gauge %s = %g", in.ID, *in.Value)

	v := *in.Value
	return models.Metrics{
		ID:    in.ID,
		MType: models.Gauge,
		Value: &v,
	}, http.StatusOK, nil
}

func persistIfEnabled(ctx context.Context, saver Saver) error {
	if saver == nil {
		return nil
	}
	return saver.Save(ctx)
}

func writeJSON(w http.ResponseWriter, secretKey string, v any) {
	respBody, err := json.Marshal(v)
	if err != nil {
		writeProtectedPlainError(w, secretKey, http.StatusInternalServerError, "internal server error")
		return
	}
	setResponseHashSHA256(w, secretKey, respBody)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBody)
}

func getMetric(w http.ResponseWriter, r *http.Request, service ServiceProvider, log *zap.Logger) {
	defer func() { _ = r.Body.Close() }()

	var req models.Metrics
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	log.Info("metric value request body", zap.String("body", buf.String()))

	if err := json.Unmarshal(buf.Bytes(), &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "missing metric id", http.StatusBadRequest)
		return
	}

	mtype := models.MetricType(strings.ToLower(string(req.MType)))
	if mtype == "" {
		http.Error(w, "missing metric type", http.StatusBadRequest)
		return
	}

	out := models.Metrics{
		ID:    req.ID,
		MType: mtype,
	}

	switch mtype {
	case models.Gauge:
		v, ok := service.GetGauge(r.Context(), req.ID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		out.Value = &v
	case models.Counter:
		v, ok := service.GetCounter(r.Context(), req.ID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		out.Delta = &v
	default:
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
		return
	}

	respBody, err := json.Marshal(out)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBody)
}

func getMetricValue(w http.ResponseWriter, r *http.Request, service ServiceProvider, _ *zap.Logger) {
	mtype := models.MetricType(strings.ToLower(chi.URLParam(r, "mtype")))
	metric := chi.URLParam(r, "metric")

	if metric == "" {
		http.NotFound(w, r)
		return
	}

	if mtype == "" {
		http.Error(w, "missing metric type", http.StatusBadRequest)
		return
	}

	switch mtype {
	case models.Gauge:
		v, ok := service.GetGauge(r.Context(), metric)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatFloat(v, 'g', -1, 64)))
	case models.Counter:
		v, ok := service.GetCounter(r.Context(), metric)
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatInt(v, 10)))
	default:
		http.Error(w, fmt.Sprintf("invalid metric type: %q", mtype), http.StatusBadRequest)
	}
}

func writeMetricsIndex(ctx context.Context, w http.ResponseWriter, service ServiceProvider, log *zap.Logger) {
	gauges := service.Gauges(ctx)
	counters := service.Counters(ctx)

	var b strings.Builder
	b.Grow(256 + (len(gauges)+len(counters))*48)

	builderWriteString(&b, "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n\t<meta charset=\"utf-8\">\n\t<title>Metrics</title>\n</head>\n<body>\n<h1>Metrics</h1>\n")
	if len(gauges) > 0 {
		builderWriteString(&b, "<h2>Gauges</h2>\n<ul>\n")
		writeSortedGaugeItems(&b, gauges)
		builderWriteString(&b, "</ul>\n")
	}
	if len(counters) > 0 {
		builderWriteString(&b, "<h2>Counters</h2>\n<ul>\n")
		writeSortedCounterItems(&b, counters)
		builderWriteString(&b, "</ul>\n")
	}
	if len(gauges) == 0 && len(counters) == 0 {
		builderWriteString(&b, "<p>No metrics yet.</p>\n")
	}
	builderWriteString(&b, "</body>\n</html>\n")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(b.String())); err != nil {
		log.Error("write metrics index", zap.Error(err))
	}
}

func writeSortedGaugeItems(b *strings.Builder, gauges map[string]float64) {
	names := make([]string, 0, len(gauges))
	for n := range gauges {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		builderWriteString(b, "<li>")
		builderWriteString(b, n)
		builderWriteString(b, ": ")
		builderWriteString(b, strconv.FormatFloat(gauges[n], 'g', -1, 64))
		builderWriteString(b, "</li>\n")
	}
}

func writeSortedCounterItems(b *strings.Builder, counters map[string]int64) {
	names := make([]string, 0, len(counters))
	for n := range counters {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		builderWriteString(b, "<li>")
		builderWriteString(b, n)
		builderWriteString(b, ": ")
		builderWriteString(b, strconv.FormatInt(counters[n], 10))
		builderWriteString(b, "</li>\n")
	}
}

// strings.Builder.WriteString не возвращает реальных ошибок; явно игнорируем для revive.
func builderWriteString(b *strings.Builder, s string) {
	_, _ = b.WriteString(s)
}
