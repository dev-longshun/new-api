package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	quotaExpiryTickInterval = 1 * time.Minute
	quotaExpiryBatchSize    = 200
)

var (
	quotaExpiryOnce    sync.Once
	quotaExpiryRunning atomic.Bool
)

func StartQuotaExpiryTask() {
	quotaExpiryOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("quota expiry task started: tick=%s", quotaExpiryTickInterval))
			ticker := time.NewTicker(quotaExpiryTickInterval)
			defer ticker.Stop()

			runQuotaExpiryOnce()
			for range ticker.C {
				runQuotaExpiryOnce()
			}
		})
	})
}

func runQuotaExpiryOnce() {
	if !quotaExpiryRunning.CompareAndSwap(false, true) {
		return
	}
	defer quotaExpiryRunning.Store(false)

	ctx := context.Background()
	total := 0
	for {
		records, err := model.GetExpiredActiveRecords(quotaExpiryBatchSize)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("quota expiry task: failed to fetch expired records: %v", err))
			return
		}
		if len(records) == 0 {
			break
		}
		for _, record := range records {
			if err := model.ExpireActiveRecord(record); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("quota expiry task: failed to expire record %d: %v", record.Id, err))
			} else {
				total++
			}
		}
		if len(records) < quotaExpiryBatchSize {
			break
		}
	}
	if common.DebugEnabled && total > 0 {
		logger.LogDebug(ctx, "quota expiry task: expired %d records", total)
	}
}
