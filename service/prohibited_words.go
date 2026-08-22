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
package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

// CheckProhibitedText returns each configured keyword at most once per request.
func CheckProhibitedText(text string) []string {
	words := setting.GetProhibitedWordsCopy()
	if len(words) == 0 || strings.TrimSpace(text) == "" {
		return nil
	}

	byLower := make(map[string]string, len(words))
	for _, word := range words {
		byLower[strings.ToLower(word)] = word
	}
	_, hits := AcSearch(strings.ToLower(text), words, false)
	seen := make(map[string]struct{}, len(hits))
	result := make([]string, 0, len(hits))
	for _, hit := range hits {
		key := strings.ToLower(hit)
		word, ok := byLower[key]
		if !ok {
			word = hit
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, word)
	}
	return result
}

func RecordProhibitedWordHits(userID int, keywords []string) error {
	return model.IncrementProhibitedWordHits(userID, keywords)
}
