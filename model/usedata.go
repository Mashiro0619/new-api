package model

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// QuotaData 柱状图数据
type QuotaData struct {
	Id        int    `json:"id"`
	UserID    int    `json:"user_id" gorm:"index;index:idx_qdt_user_created_at,priority:1"`
	Username  string `json:"username" gorm:"index:idx_qdt_model_user_name,priority:2;size:64;default:''"`
	ModelName string `json:"model_name" gorm:"index:idx_qdt_model_user_name,priority:1;size:64;default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index:idx_qdt_created_at,priority:2;index:idx_qdt_user_created_at,priority:2"`
	UseGroup  string `json:"use_group" gorm:"index;size:64;default:''"`
	TokenID   int    `json:"token_id" gorm:"index;default:0"`
	TokenName string `json:"token_name" gorm:"index;size:64;default:''"`
	ChannelID int    `json:"channel_id" gorm:"index;default:0"`
	NodeName  string `json:"node_name" gorm:"index;size:64;default:''"`
	TokenUsed int    `json:"token_used" gorm:"default:0"`
	Count     int    `json:"count" gorm:"default:0"`
	Quota     int    `json:"quota" gorm:"default:0"`

	InputTokens         int64 `json:"input_tokens" gorm:"type:bigint;default:0"`
	OutputTokens        int64 `json:"output_tokens" gorm:"type:bigint;default:0"`
	CacheCreationTokens int64 `json:"cache_creation_tokens" gorm:"type:bigint;default:0"`
	CacheReadTokens     int64 `json:"cache_read_tokens" gorm:"type:bigint;default:0"`
	TokenMetricsCount   int64 `json:"token_metrics_count" gorm:"type:bigint;default:0"`
}

// TokenMetrics contains normalized, non-overlapping token usage for one
// request. A nil *TokenMetrics means the request has no token semantics and
// must not make legacy/untracked dashboard data look like measured zeroes.
type TokenMetrics struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	CacheReadTokens     int64 `json:"cache_read_tokens"`
}

func NormalizeTokenMetrics(inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int64, inputAlreadySeparated bool) TokenMetrics {
	inputTokens = max(inputTokens, 0)
	outputTokens = max(outputTokens, 0)
	cacheCreationTokens = max(cacheCreationTokens, 0)
	cacheReadTokens = max(cacheReadTokens, 0)
	if !inputAlreadySeparated {
		if inputTokens <= cacheCreationTokens {
			inputTokens = 0
		} else {
			inputTokens -= cacheCreationTokens
		}
		if inputTokens <= cacheReadTokens {
			inputTokens = 0
		} else {
			inputTokens -= cacheReadTokens
		}
	}
	return TokenMetrics{
		InputTokens:         inputTokens,
		OutputTokens:        outputTokens,
		CacheCreationTokens: cacheCreationTokens,
		CacheReadTokens:     cacheReadTokens,
	}
}

type QuotaDataLogParams struct {
	UserID       int
	Username     string
	ModelName    string
	Quota        int
	CreatedAt    int64
	TokenUsed    int
	UseGroup     string
	TokenID      int
	TokenName    string
	ChannelID    int
	NodeName     string
	TokenMetrics *TokenMetrics
}

func UpdateQuotaData() {
	for {
		if common.DataExportEnabled {
			common.SysLog("正在更新数据看板数据...")
			SaveQuotaDataCache()
		}
		time.Sleep(time.Duration(common.DataExportInterval) * time.Minute)
	}
}

var CacheQuotaData = make(map[string]*QuotaData)
var CacheQuotaDataLock = sync.Mutex{}

func logQuotaDataCache(quotaData *QuotaData) {
	key := fmt.Sprintf("%d\x00%s\x00%s\x00%d\x00%s\x00%d\x00%s\x00%d\x00%s",
		quotaData.UserID,
		quotaData.Username,
		quotaData.ModelName,
		quotaData.CreatedAt,
		quotaData.UseGroup,
		quotaData.TokenID,
		quotaData.TokenName,
		quotaData.ChannelID,
		quotaData.NodeName,
	)
	count := quotaData.Count
	quota := quotaData.Quota
	tokenUsed := quotaData.TokenUsed
	cachedQuotaData, ok := CacheQuotaData[key]
	if ok {
		cachedQuotaData.Count += count
		cachedQuotaData.Quota += quota
		cachedQuotaData.TokenUsed += tokenUsed
		cachedQuotaData.InputTokens += quotaData.InputTokens
		cachedQuotaData.OutputTokens += quotaData.OutputTokens
		cachedQuotaData.CacheCreationTokens += quotaData.CacheCreationTokens
		cachedQuotaData.CacheReadTokens += quotaData.CacheReadTokens
		cachedQuotaData.TokenMetricsCount += quotaData.TokenMetricsCount
		quotaData = cachedQuotaData
	}
	CacheQuotaData[key] = quotaData
}

func LogQuotaData(params QuotaDataLogParams) {
	// 只精确到小时
	createdAt := params.CreatedAt - (params.CreatedAt % 3600)
	quotaData := &QuotaData{
		UserID:    params.UserID,
		Username:  params.Username,
		ModelName: params.ModelName,
		CreatedAt: createdAt,
		UseGroup:  params.UseGroup,
		TokenID:   params.TokenID,
		TokenName: params.TokenName,
		ChannelID: params.ChannelID,
		NodeName:  params.NodeName,
		Count:     1,
		Quota:     params.Quota,
		TokenUsed: params.TokenUsed,
	}
	if params.TokenMetrics != nil {
		quotaData.InputTokens = max(params.TokenMetrics.InputTokens, 0)
		quotaData.OutputTokens = max(params.TokenMetrics.OutputTokens, 0)
		quotaData.CacheCreationTokens = max(params.TokenMetrics.CacheCreationTokens, 0)
		quotaData.CacheReadTokens = max(params.TokenMetrics.CacheReadTokens, 0)
		quotaData.TokenMetricsCount = 1
	}

	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	logQuotaDataCache(quotaData)
}

func SaveQuotaDataCache() {
	CacheQuotaDataLock.Lock()
	defer CacheQuotaDataLock.Unlock()
	size := len(CacheQuotaData)
	// 如果缓存中有数据，就保存到数据库中
	// 1. 先查询数据库中是否有数据
	// 2. 如果有数据，就更新数据
	// 3. 如果没有数据，就插入数据
	for _, quotaData := range CacheQuotaData {
		quotaDataDB := &QuotaData{}
		DB.Table("quota_data").
			Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and token_name = ? and channel_id = ? and node_name = ?",
				quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.TokenName, quotaData.ChannelID, quotaData.NodeName).
			First(quotaDataDB)
		if quotaDataDB.Id > 0 {
			//quotaDataDB.Count += quotaData.Count
			//quotaDataDB.Quota += quotaData.Quota
			//DB.Table("quota_data").Save(quotaDataDB)
			increaseQuotaData(quotaData)
		} else {
			DB.Table("quota_data").Create(quotaData)
		}
	}
	CacheQuotaData = make(map[string]*QuotaData)
	common.SysLog(fmt.Sprintf("保存数据看板数据成功，共保存%d条数据", size))
}

func increaseQuotaData(quotaData *QuotaData) {
	err := DB.Table("quota_data").
		Where("user_id = ? and username = ? and model_name = ? and created_at = ? and use_group = ? and token_id = ? and token_name = ? and channel_id = ? and node_name = ?",
			quotaData.UserID, quotaData.Username, quotaData.ModelName, quotaData.CreatedAt, quotaData.UseGroup, quotaData.TokenID, quotaData.TokenName, quotaData.ChannelID, quotaData.NodeName).
		Updates(map[string]interface{}{
			"count":                 gorm.Expr("count + ?", quotaData.Count),
			"quota":                 gorm.Expr("quota + ?", quotaData.Quota),
			"token_used":            gorm.Expr("token_used + ?", quotaData.TokenUsed),
			"input_tokens":          gorm.Expr("input_tokens + ?", quotaData.InputTokens),
			"output_tokens":         gorm.Expr("output_tokens + ?", quotaData.OutputTokens),
			"cache_creation_tokens": gorm.Expr("cache_creation_tokens + ?", quotaData.CacheCreationTokens),
			"cache_read_tokens":     gorm.Expr("cache_read_tokens + ?", quotaData.CacheReadTokens),
			"token_metrics_count":   gorm.Expr("token_metrics_count + ?", quotaData.TokenMetricsCount),
		}).Error
	if err != nil {
		common.SysLog(fmt.Sprintf("increaseQuotaData error: %s", err))
	}
}

func GetQuotaDataByUsername(username string, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").
		Select("user_id, username, model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("username = ? and created_at >= ? and created_at <= ?", username, startTime, endTime).
		Group("user_id, username, model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataByUserId(userId int, startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	err = DB.Table("quota_data").
		Select("user_id, username, model_name, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("user_id = ? and created_at >= ? and created_at <= ?", userId, startTime, endTime).
		Group("user_id, username, model_name, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetQuotaDataGroupByUser(startTime int64, endTime int64) (quotaData []*QuotaData, err error) {
	var quotaDatas []*QuotaData
	err = DB.Table("quota_data").
		Select("username, created_at, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Group("username, created_at").
		Find(&quotaDatas).Error
	return quotaDatas, err
}

func GetAllQuotaDates(startTime int64, endTime int64, username string) (quotaData []*QuotaData, err error) {
	if username != "" {
		return GetQuotaDataByUsername(username, startTime, endTime)
	}
	var quotaDatas []*QuotaData
	// 从quota_data表中查询数据
	// only select model_name, sum(count) as count, sum(quota) as quota, model_name, created_at from quota_data group by model_name, created_at;
	//err = DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", startTime, endTime).Find(&quotaDatas).Error
	err = DB.Table("quota_data").Select("model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at").Where("created_at >= ? and created_at <= ?", startTime, endTime).Group("model_name, created_at").Find(&quotaDatas).Error
	return quotaDatas, err
}

type QuotaDataQuery struct {
	StartTime int64
	EndTime   int64
	UserID    int
	Username  string
	ModelName string
	TokenName string
	UseGroup  string
	ChannelID int
}

func GetQuotaDatesByFilter(query QuotaDataQuery) (quotaData []*QuotaData, err error) {
	rows := make([]*QuotaData, 0)
	tx := DB.Table("quota_data").Where("created_at >= ? and created_at <= ?", query.StartTime, query.EndTime)
	if query.UserID > 0 {
		tx = tx.Where("user_id = ?", query.UserID)
	}
	if query.Username != "" {
		tx = tx.Where("username = ?", query.Username)
	}
	if query.ModelName != "" {
		tx = tx.Where("model_name = ?", query.ModelName)
	}
	if query.TokenName != "" {
		tx = tx.Where("token_name = ?", query.TokenName)
	}
	if query.UseGroup != "" {
		tx = tx.Where("use_group = ?", query.UseGroup)
	}
	if query.ChannelID > 0 {
		tx = tx.Where("channel_id = ?", query.ChannelID)
	}

	selectColumns := "model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at"
	groupColumns := "model_name, created_at"
	if query.UserID > 0 || query.Username != "" {
		selectColumns = "user_id, username, model_name, sum(count) as count, sum(quota) as quota, sum(token_used) as token_used, created_at"
		groupColumns = "user_id, username, model_name, created_at"
	}
	err = tx.Select(selectColumns).Group(groupColumns).Find(&rows).Error
	return rows, err
}
