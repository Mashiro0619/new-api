/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
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
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/site_model_calls"
	"github.com/gin-gonic/gin"
)

func siteModelCallsAllowed(c *gin.Context) bool {
	return c.GetInt("role") >= common.RoleAdminUser || site_model_calls.GetConfig().Enabled
}

func rejectSiteModelCallsIfDisabled(c *gin.Context) bool {
	if siteModelCallsAllowed(c) {
		return false
	}
	c.JSON(http.StatusForbidden, gin.H{
		"success": false,
		"message": "全站模型调用未开放",
	})
	return true
}

func GetSiteModelCallsSummary(c *gin.Context) {
	if rejectSiteModelCallsIfDisabled(c) {
		return
	}
	config := site_model_calls.GetConfig()
	result, err := perfmetrics.QuerySiteSummary(config.Models)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func GetSiteModelCallModels(c *gin.Context) {
	if rejectSiteModelCallsIfDisabled(c) {
		return
	}
	actual, err := perfmetrics.QuerySiteModelNames()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	seen := make(map[string]struct{}, len(actual))
	for _, name := range actual {
		seen[name] = struct{}{}
	}
	for _, pricing := range model.GetPricing() {
		if name := strings.TrimSpace(pricing.ModelName); name != "" {
			seen[name] = struct{}{}
		}
	}
	for _, name := range site_model_calls.GetConfig().Models {
		if name = strings.TrimSpace(name); name != "" {
			seen[name] = struct{}{}
		}
	}
	models := make([]string, 0, len(seen))
	for name := range seen {
		models = append(models, name)
	}
	sort.Strings(models)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": models})
}
