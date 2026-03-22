package model

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	RedemptionQuotaStatusPending = 1 // 等待中
	RedemptionQuotaStatusActive  = 2 // 激活中
	RedemptionQuotaStatusUsed    = 3 // 已用完
	RedemptionQuotaStatusExpired = 4 // 已过期
)

type RedemptionQuotaRecord struct {
	Id            int            `json:"id"`
	UserId        int            `json:"user_id" gorm:"index"`
	RedemptionId  int            `json:"redemption_id" gorm:"index"`
	Quota         int            `json:"quota"`
	UsedQuota     int            `json:"used_quota" gorm:"default:0"`
	Status        int            `json:"status" gorm:"default:1"`
	CreatedTime   int64          `json:"created_time" gorm:"bigint"`
	ActivatedTime int64          `json:"activated_time" gorm:"bigint"`
	ExpiredTime   int64          `json:"expired_time" gorm:"bigint"`
	QuotaDuration int64          `json:"quota_duration" gorm:"bigint"` // 有效期秒数，0=永不过期
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// GetActiveRecord returns the currently active quota record for a user.
func GetActiveRecord(userId int) (*RedemptionQuotaRecord, error) {
	var record RedemptionQuotaRecord
	err := DB.Where("user_id = ? AND status = ?", userId, RedemptionQuotaStatusActive).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetNextPendingRecord returns the next pending record in queue order.
func GetNextPendingRecord(userId int) (*RedemptionQuotaRecord, error) {
	var record RedemptionQuotaRecord
	err := DB.Where("user_id = ? AND status = ?", userId, RedemptionQuotaStatusPending).
		Order("id asc").
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ActivateRecord activates a pending record within a transaction.
func ActivateRecord(tx *gorm.DB, record *RedemptionQuotaRecord) error {
	now := common.GetTimestamp()
	record.ActivatedTime = now
	record.Status = RedemptionQuotaStatusActive
	if record.QuotaDuration > 0 {
		record.ExpiredTime = now + record.QuotaDuration
	} else {
		record.ExpiredTime = 0
	}
	return tx.Model(record).Select("status", "activated_time", "expired_time").Updates(record).Error
}

// ExpireActiveRecord marks an active record as expired, deducts remaining quota from user,
// and activates the next pending record if any.
func ExpireActiveRecord(record *RedemptionQuotaRecord) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Re-fetch with lock to get current used_quota
		var current RedemptionQuotaRecord
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&current, "id = ?", record.Id).Error; err != nil {
			return err
		}
		if current.Status != RedemptionQuotaStatusActive {
			return nil // already handled
		}
		remaining := current.Quota - current.UsedQuota
		if err := tx.Model(&current).Updates(map[string]interface{}{
			"status":     RedemptionQuotaStatusExpired,
			"used_quota": current.Quota,
		}).Error; err != nil {
			return err
		}
		// Deduct remaining from user quota
		if remaining > 0 {
			if err := tx.Model(&User{}).Where("id = ?", current.UserId).
				Update("quota", gorm.Expr("quota - ?", remaining)).Error; err != nil {
				return err
			}
		}
		// Activate next pending record
		next, err := GetNextPendingRecord(current.UserId)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if next != nil {
			if err := ActivateRecord(tx, next); err != nil {
				return err
			}
			// Add next record's quota to user
			if err := tx.Model(&User{}).Where("id = ?", current.UserId).
				Update("quota", gorm.Expr("quota + ?", next.Quota)).Error; err != nil {
				return err
			}
			InvalidateActiveQuotaCache(current.UserId)
		} else {
			InvalidateActiveQuotaCache(current.UserId)
		}
		return nil
	})
}

// MarkRecordUsed marks a record as fully used and activates the next pending record.
func MarkRecordUsedAndActivateNext(record *RedemptionQuotaRecord) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var current RedemptionQuotaRecord
		if err := tx.Set("gorm:query_option", "FOR UPDATE").
			First(&current, "id = ?", record.Id).Error; err != nil {
			return err
		}
		if current.Status != RedemptionQuotaStatusActive {
			return nil
		}
		if err := tx.Model(&current).Update("status", RedemptionQuotaStatusUsed).Error; err != nil {
			return err
		}
		// Activate next pending record
		next, err := GetNextPendingRecord(current.UserId)
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if next != nil {
			if err := ActivateRecord(tx, next); err != nil {
				return err
			}
			if err := tx.Model(&User{}).Where("id = ?", current.UserId).
				Update("quota", gorm.Expr("quota + ?", next.Quota)).Error; err != nil {
				return err
			}
			InvalidateActiveQuotaCache(current.UserId)
		} else {
			InvalidateActiveQuotaCache(current.UserId)
		}
		return nil
	})
}

// IncrActiveRecordUsedQuota increments used_quota on the active record by delta.
// Called after a settled quota deduction. Safe for both batch and non-batch paths.
func IncrActiveRecordUsedQuota(userId int, delta int) {
	if delta <= 0 {
		return
	}
	record, err := GetActiveRecord(userId)
	if err != nil {
		return
	}
	// Atomic increment, clamped to record.Quota
	result := DB.Model(record).
		Where("used_quota + ? <= quota", delta).
		Update("used_quota", gorm.Expr("used_quota + ?", delta))
	if result.Error != nil {
		return
	}
	// Re-fetch to check if fully consumed
	var updated RedemptionQuotaRecord
	if err := DB.First(&updated, "id = ?", record.Id).Error; err != nil {
		return
	}
	if updated.UsedQuota >= updated.Quota {
		MarkRecordUsedAndActivateNext(&updated)
	}
}

// GetUserQueueInfo returns queue summary for a user.
type UserQueueInfo struct {
	ActiveRecord   *RedemptionQuotaRecord   `json:"active_record"`
	PendingCount   int64                    `json:"pending_count"`
	PendingQuota   int64                    `json:"pending_quota"`
	PendingRecords []*RedemptionQuotaRecord `json:"pending_records"`
}

func GetUserQueueInfo(userId int) (*UserQueueInfo, error) {
	info := &UserQueueInfo{}

	active, err := GetActiveRecord(userId)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	info.ActiveRecord = active

	var pending []*RedemptionQuotaRecord
	if err := DB.Where("user_id = ? AND status = ?", userId, RedemptionQuotaStatusPending).
		Order("id asc").Find(&pending).Error; err != nil {
		return nil, err
	}
	info.PendingRecords = pending
	info.PendingCount = int64(len(pending))
	for _, r := range pending {
		info.PendingQuota += int64(r.Quota)
	}
	return info, nil
}

// activeQuotaRecordCacheKey returns the Redis key for a user's active quota record cache.
func activeQuotaRecordCacheKey(userId int) string {
	return fmt.Sprintf("user:%d:active_quota_record", userId)
}

// CachedActiveQuota holds the cached state of the active quota record.
type CachedActiveQuota struct {
	RecordId      int   `json:"record_id"`
	Remaining     int   `json:"remaining"`
	ExpiredTime   int64 `json:"expired_time"`
	QuotaDuration int64 `json:"quota_duration"`
}

// CacheActiveQuotaRecord stores the active quota record in Redis.
func CacheActiveQuotaRecord(userId int, c CachedActiveQuota) error {
	if !common.RedisEnabled {
		return nil
	}
	ttl := time.Duration(common.RedisKeyCacheSeconds()) * time.Second
	return common.RedisHSetObj(activeQuotaRecordCacheKey(userId), &c, ttl)
}

// GetCachedActiveQuotaRecord reads the active quota record from Redis.
func GetCachedActiveQuotaRecord(userId int) (*CachedActiveQuota, error) {
	if !common.RedisEnabled {
		return nil, fmt.Errorf("redis disabled")
	}
	var c CachedActiveQuota
	if err := common.RedisHGetObj(activeQuotaRecordCacheKey(userId), &c); err != nil {
		return nil, err
	}
	if c.RecordId == 0 {
		return nil, fmt.Errorf("no active record cached")
	}
	return &c, nil
}

// InvalidateActiveQuotaCache removes the active quota record cache for a user.
func InvalidateActiveQuotaCache(userId int) error {
	if !common.RedisEnabled {
		return nil
	}
	return common.RedisDelKey(activeQuotaRecordCacheKey(userId))
}

// GetExpiredActiveRecords returns all active records that have passed their expiry time.
func GetExpiredActiveRecords(limit int) ([]*RedemptionQuotaRecord, error) {
	now := common.GetTimestamp()
	var records []*RedemptionQuotaRecord
	err := DB.Where("status = ? AND expired_time > 0 AND expired_time < ?",
		RedemptionQuotaStatusActive, now).
		Limit(limit).Find(&records).Error
	return records, err
}
