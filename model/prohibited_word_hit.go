/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package model

import (
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProhibitedWordHit struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"not null;uniqueIndex:uk_prohibited_word_user_keyword,priority:1"`
	Keyword   string `json:"keyword" gorm:"size:255;not null;uniqueIndex:uk_prohibited_word_user_keyword,priority:2"`
	HitCount  int64  `json:"hit_count" gorm:"not null;default:0"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ProhibitedWordHit) TableName() string { return "prohibited_word_hits" }

func IncrementProhibitedWordHits(userID int, keywords []string) error {
	if userID <= 0 || len(keywords) == 0 {
		return nil
	}
	unique := make(map[string]struct{}, len(keywords))
	normalized := make([]string, 0, len(keywords))
	for _, raw := range keywords {
		keyword := strings.TrimSpace(raw)
		if keyword == "" {
			continue
		}
		key := strings.ToLower(keyword)
		if _, ok := unique[key]; ok {
			continue
		}
		unique[key] = struct{}{}
		normalized = append(normalized, keyword)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)

	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, keyword := range normalized {
			hit := &ProhibitedWordHit{
				UserId:    userID,
				Keyword:   strings.ToLower(keyword),
				HitCount:  1,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}, {Name: "keyword"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"hit_count":  gorm.Expr("prohibited_word_hits.hit_count + ?", 1),
					"updated_at": now,
				}),
			}).Create(hit).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type ProhibitedWordUser struct {
	UserId   int    `json:"user_id"`
	Username string `json:"username"`
}

type ProhibitedWordSummaryItem struct {
	UserId   int              `json:"user_id"`
	Username string           `json:"username"`
	Counts   map[string]int64 `json:"counts"`
}

type ProhibitedWordSummaryPage struct {
	Items    []ProhibitedWordSummaryItem `json:"items"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

func GetProhibitedWordSummary(page, pageSize int, keywords []string) (ProhibitedWordSummaryPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	var total int64
	if err := DB.Model(&User{}).Count(&total).Error; err != nil {
		return ProhibitedWordSummaryPage{}, err
	}
	var users []ProhibitedWordUser
	if err := DB.Model(&User{}).
		Select("id as user_id, username").
		Order("id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&users).Error; err != nil {
		return ProhibitedWordSummaryPage{}, err
	}

	counts := make(map[int]map[string]int64, len(users))
	if len(users) > 0 && len(keywords) > 0 {
		userIDs := make([]int, 0, len(users))
		keywordKeys := make([]string, 0, len(keywords))
		keywordLabels := make(map[string]string, len(keywords))
		for _, keyword := range keywords {
			key := strings.ToLower(strings.TrimSpace(keyword))
			if key != "" {
				keywordKeys = append(keywordKeys, key)
				keywordLabels[key] = keyword
			}
		}
		for _, user := range users {
			userIDs = append(userIDs, user.UserId)
			counts[user.UserId] = make(map[string]int64, len(keywords))
		}
		var hits []ProhibitedWordHit
		if err := DB.Where("user_id IN ? AND keyword IN ?", userIDs, keywordKeys).Find(&hits).Error; err != nil {
			return ProhibitedWordSummaryPage{}, err
		}
		for _, hit := range hits {
			label := keywordLabels[hit.Keyword]
			if label != "" {
				counts[hit.UserId][label] = hit.HitCount
			}
		}
	}

	items := make([]ProhibitedWordSummaryItem, 0, len(users))
	for _, user := range users {
		userCounts := counts[user.UserId]
		if userCounts == nil {
			userCounts = make(map[string]int64, len(keywords))
		}
		for _, keyword := range keywords {
			if _, ok := userCounts[keyword]; !ok {
				userCounts[keyword] = 0
			}
		}
		items = append(items, ProhibitedWordSummaryItem{
			UserId:   user.UserId,
			Username: user.Username,
			Counts:   userCounts,
		})
	}
	return ProhibitedWordSummaryPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func ClearProhibitedWordHits() error {
	return DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ProhibitedWordHit{}).Error
}
