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
package perfmetrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSiteModelTotalTokensFallsBackToKnownOutputTokens(t *testing.T) {
	require.EqualValues(t, 120, siteModelTotalTokens(counters{totalTokens: 120, outputTokens: 80}))
	require.EqualValues(t, 80, siteModelTotalTokens(counters{outputTokens: 80}))
	require.Zero(t, siteModelTotalTokens(counters{}))
}

func TestSiteCacheHitRateUsesNonOverlappingInputDenominator(t *testing.T) {
	require.InDelta(t, 25.0, siteCacheHitRate(300, 100), 0.0001)
	require.Zero(t, siteCacheHitRate(0, 0))
	require.Zero(t, siteCacheHitRate(-1, -1))
}
