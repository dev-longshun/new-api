package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
)

type ChannelHealthLog struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	ChannelId    int    `json:"channel_id" gorm:"index:idx_chl_channel_created,priority:1"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index:idx_chl_channel_created,priority:2"`
	ResponseTime int    `json:"response_time"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message" gorm:"type:varchar(512)"`
}

func InsertChannelHealthLog(channelId int, responseTime int, success bool, errMsg string) {
	if len(errMsg) > 512 {
		errMsg = errMsg[:512]
	}
	log := &ChannelHealthLog{
		ChannelId:    channelId,
		CreatedAt:    common.GetTimestamp(),
		ResponseTime: responseTime,
		Success:      success,
		ErrorMessage: errMsg,
	}
	err := DB.Create(log).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("记录渠道健康日志失败: %s", err))
	}
}

func GetChannelHealthHistory(channelId int, limit int) ([]*ChannelHealthLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var logs []*ChannelHealthLog
	err := DB.Where("channel_id = ?", channelId).
		Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}

func CleanupOldHealthLogs(retentionDays int) {
	cutoff := common.GetTimestamp() - int64(retentionDays*86400)
	result := DB.Where("created_at < ?", cutoff).Delete(&ChannelHealthLog{})
	if result.Error != nil {
		common.SysLog(fmt.Sprintf("清理渠道健康日志失败: %s", result.Error))
	} else if result.RowsAffected > 0 {
		common.SysLog(fmt.Sprintf("清理渠道健康日志成功，共清理%d条数据", result.RowsAffected))
	}
}
