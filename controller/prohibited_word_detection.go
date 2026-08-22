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
package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

type prohibitedWordConfigRequest struct {
	Keywords []string `json:"keywords"`
}

func GetProhibitedWordConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"keywords": setting.GetProhibitedWordsCopy(),
		},
	})
}

func UpdateProhibitedWordConfig(c *gin.Context) {
	var request prohibitedWordConfigRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid keywords"})
		return
	}
	keywords := setting.NormalizeProhibitedWords(request.Keywords)
	for _, keyword := range keywords {
		if len(keyword) > 255 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "keyword must be 255 bytes or fewer"})
			return
		}
	}
	if err := model.UpdateOption("ProhibitedWords", strings.Join(keywords, "\n")); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"keywords": keywords}})
}

func GetProhibitedWordSummary(c *gin.Context) {
	page, pageSize := parseProhibitedWordPage(c)
	keywords := setting.GetProhibitedWordsCopy()
	result, err := model.GetProhibitedWordSummary(page, pageSize, keywords)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func ClearProhibitedWordStats(c *gin.Context) {
	if err := model.ClearProhibitedWordHits(); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func parseProhibitedWordPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.Query("p"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
