package pkg

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

type MetricsCache struct {
	cacheMutex *sync.Mutex
	entries    map[string]cacheEntry
	ttl        time.Duration
}

func NewMetricsCache(ttl time.Duration) *MetricsCache {
	return &MetricsCache{
		cacheMutex: &sync.Mutex{},
		entries:    map[string]cacheEntry{},
		ttl:        ttl,
	}
}

func getMetricHash(metric prometheus.Metric) string {
	var dto dto.Metric
	metric.Write(&dto)
	labelString := metric.Desc().String()

	for _, labelPair := range dto.GetLabel() {
		labelString = fmt.Sprintf("%s,%s,%s", labelString, labelPair.GetName(), labelPair.GetValue())
	}

	checksum := sha256.Sum256([]byte(labelString))
	return fmt.Sprintf("%x", checksum[:])
}

// AddMetric adds a metric to the cache
func (mc *MetricsCache) AddMetric(metric prometheus.Metric) {
	mc.cacheMutex.Lock()
	mc.entries[getMetricHash(metric)] = cacheEntry{
		creation: time.Now(),
		metric:   metric,
	}
	mc.cacheMutex.Unlock()
}

// GetAllMetrics Iterates over all cached metrics and discards expired ones.
func (mc *MetricsCache) GetAllMetrics() []prometheus.Metric {
	mc.cacheMutex.Lock()
	returnArr := make([]prometheus.Metric, 0)
	for k, v := range mc.entries {
		if time.Since(v.creation).Seconds() > mc.ttl.Seconds() {
			delete(mc.entries, k)
		} else {
			returnArr = append(returnArr, v.metric)
		}
	}
	mc.cacheMutex.Unlock()
	return returnArr
}

type cacheEntry struct {
	creation time.Time
	metric   prometheus.Metric
}
