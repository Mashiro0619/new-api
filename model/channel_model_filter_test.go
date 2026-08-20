package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func useChannelModelFilterDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Channel{}))
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	initCol()
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
		initCol()
	})
	return db
}

func TestSplitChannelModelsTrimsAndDropsEmptyEntries(t *testing.T) {
	assert.Equal(t, []string{"gpt-4o", "claude-3"}, SplitChannelModels(", gpt-4o, ,claude-3 ,,"))
	assert.Empty(t, SplitChannelModels(" , , "))
}

func TestChannelModelFiltersMatchCompleteEntriesWithOrAndTextSearch(t *testing.T) {
	db := useChannelModelFilterDB(t)
	target := "mapped-target"
	channels := []Channel{
		{Name: "first", Key: "key-1", Models: "gpt-4o, gpt-4o-mini", Tag: stringPointer("tag-a"), ModelMapping: &target},
		{Name: "second", Key: "key-2", Models: "gpt-4o-mini", Tag: stringPointer("tag-a")},
		{Name: "third", Key: "key-3", Models: "gpt-4", Tag: stringPointer("tag-b")},
	}
	require.NoError(t, db.Create(&channels).Error)

	matched, err := SearchChannels("", "", "", []string{"gpt-4o"}, false)
	require.NoError(t, err)
	assert.Equal(t, []int{channels[0].Id}, channelIDs(matched))

	matched, err = SearchChannels("", "", "", []string{"gpt-4o", "gpt-4"}, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{channels[0].Id, channels[2].Id}, channelIDs(matched))

	matched, err = SearchChannels("", "", "", []string{"gpt-4o-mini"}, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{channels[0].Id, channels[1].Id}, channelIDs(matched))

	matched, err = SearchChannels("", "", "mini", []string{"gpt-4o"}, false)
	require.NoError(t, err)
	assert.Equal(t, []int{channels[0].Id}, channelIDs(matched))

	matched, err = SearchChannels("", "", "", []string{"gpt-4o"}, false)
	require.NoError(t, err)
	assert.NotContains(t, channelIDs(matched), channels[1].Id)

	all, err := SearchChannels("", "", "", nil, false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{channels[0].Id, channels[1].Id, channels[2].Id}, channelIDs(all))
}

func TestSearchTagsAppliesChannelModelFilter(t *testing.T) {
	db := useChannelModelFilterDB(t)
	tagA := "tag-a"
	tagB := "tag-b"
	channels := []Channel{
		{Name: "matching", Key: "key-1", Models: "gpt-4o", Tag: &tagA},
		{Name: "non-matching", Key: "key-2", Models: "gpt-4", Tag: &tagB},
	}
	require.NoError(t, db.Create(&channels).Error)

	tags, err := SearchTags("", "", "", []string{"gpt-4o"}, false)
	require.NoError(t, err)
	assert.Equal(t, []string{tagA}, stringPointers(tags))
}

func TestGetProvidedModelsDeduplicatesAndSortsConfiguredEntries(t *testing.T) {
	db := useChannelModelFilterDB(t)
	target := "mapped-only"
	require.NoError(t, db.Create(&[]Channel{
		{Key: "key-1", Models: "zeta, alpha, ,zeta", ModelMapping: &target},
		{Key: "key-2", Models: " beta,alpha "},
	}).Error)

	models, err := GetProvidedModels()
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta", "zeta"}, models)
}

func stringPointer(value string) *string {
	return &value
}

func channelIDs(channels []*Channel) []int {
	ids := make([]int, 0, len(channels))
	for _, channel := range channels {
		ids = append(ids, channel.Id)
	}
	return ids
}

func stringPointers(values []*string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}
