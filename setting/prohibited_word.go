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
package setting

import (
	"sort"
	"strings"
	"sync"
)

var prohibitedWordsMu sync.RWMutex
var ProhibitedWords []string

func NormalizeProhibitedWords(words []string) []string {
	seen := make(map[string]struct{}, len(words))
	result := make([]string, 0, len(words))
	for _, raw := range words {
		word := strings.TrimSpace(raw)
		if word == "" {
			continue
		}
		key := strings.ToLower(word)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, word)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}

func ProhibitedWordsToString() string {
	prohibitedWordsMu.RLock()
	defer prohibitedWordsMu.RUnlock()
	return strings.Join(ProhibitedWords, "\n")
}

func ProhibitedWordsFromString(value string) {
	words := NormalizeProhibitedWords(strings.Split(value, "\n"))
	prohibitedWordsMu.Lock()
	ProhibitedWords = words
	prohibitedWordsMu.Unlock()
}

func GetProhibitedWordsCopy() []string {
	prohibitedWordsMu.RLock()
	defer prohibitedWordsMu.RUnlock()
	return append([]string(nil), ProhibitedWords...)
}
