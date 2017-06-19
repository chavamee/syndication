/*
  Copyright (C) 2017 Jorge Martinez Hernandez

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package database

import (
	uuid "github.com/satori/go.uuid"
	"os"
	"strconv"
	"testing"

	"github.com/chavamee/syndication/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type (
	DatabaseTestSuite struct {
		suite.Suite

		db   DB
		user models.User
	}
)

const TestDatabasePath = "/tmp/syndication-test.db"

func (suite *DatabaseTestSuite) SetupTest() {
	var err error
	suite.db, err = NewDB("sqlite3", TestDatabasePath)
	suite.Require().Nil(err)

	err = suite.db.NewUser("test", "golang")
	suite.Require().Nil(err)

	suite.user, err = suite.db.UserFromName("test")
	suite.Require().Nil(err)
}

func (suite *DatabaseTestSuite) TearDownTest() {
	suite.db.Close()
	os.Remove(suite.db.Path)
}

func (suite *DatabaseTestSuite) TestNewCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(ctg.UUID)
	suite.NotZero(ctg.ID)
	suite.NotZero(ctg.UserID)
	suite.NotZero(ctg.CreatedAt)
	suite.NotZero(ctg.UpdatedAt)
	suite.NotZero(ctg.UserID)

	query, err := suite.db.Category(ctg.UUID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Name)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)
}

func (suite *DatabaseTestSuite) TestCategories() {
	for i := 0; i < 5; i++ {
		ctg := models.Category{
			Name: "Test Category " + strconv.Itoa(i),
		}

		err := suite.db.NewCategory(&ctg, &suite.user)
		suite.Require().Nil(err)
	}

	ctgs := suite.db.Categories(&suite.user)
	suite.Len(ctgs, 7)
}

func (suite *DatabaseTestSuite) TestEditCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Category(ctg.UUID, &suite.user)
	suite.Nil(err)
	suite.Equal(query.Name, "News")

	ctg.Name = "World News"
	suite.db.EditCategory(&ctg, &suite.user)

	query, err = suite.db.Category(ctg.UUID, &suite.user)
	suite.Nil(err)
	suite.Equal(ctg.ID, query.ID)
	suite.Equal(query.Name, "World News")
}

func (suite *DatabaseTestSuite) TestEditNonExistingCategory() {
	err := suite.db.EditCategory(&models.Category{}, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Category(ctg.UUID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.UUID)

	err = suite.db.DeleteCategory(ctg.UUID, &suite.user)
	suite.Nil(err)

	_, err = suite.db.Category(ctg.UUID, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteNonExistingCategory() {
	err := suite.db.DeleteCategory(uuid.NewV4().String(), &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteSystemCategory() {
	err := suite.db.DeleteCategory(suite.user.SavedCategoryUUID, &suite.user)
	suite.IsType(BadRequest{}, err)
}

func (suite *DatabaseTestSuite) TestNewFeedWithDefaults() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.UUID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)

	suite.NotZero(query.Category.ID)
	suite.NotEmpty(query.Category.UUID)
	suite.Equal(query.Category.Name, models.Uncategorized)

	feeds, err := suite.db.FeedsFromCategory(&suite.user, query.Category.UUID)
	suite.Nil(err)
	suite.NotEmpty(feeds)
	suite.Equal(feeds[0].Title, feed.Title)
	suite.Equal(feeds[0].ID, feed.ID)
	suite.Equal(feeds[0].UUID, feed.UUID)
}

func (suite *DatabaseTestSuite) TestNewFeedWithCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(ctg.UUID)
	suite.NotZero(ctg.ID)
	suite.Empty(ctg.Feeds)

	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     ctg,
		CategoryID:   ctg.ID,
		CategoryUUID: ctg.UUID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.UUID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)
	suite.NotZero(query.CreatedAt)
	suite.NotZero(query.UpdatedAt)
	suite.NotZero(query.UserID)

	suite.NotZero(query.Category.ID)
	suite.NotEmpty(query.Category.UUID)
	suite.Equal(query.Category.Name, "News")

	feeds, err := suite.db.FeedsFromCategory(&suite.user, ctg.UUID)
	suite.Nil(err)
	suite.NotEmpty(feeds)
	suite.Equal(feeds[0].Title, feed.Title)
	suite.Equal(feeds[0].ID, feed.ID)
	suite.Equal(feeds[0].UUID, feed.UUID)
}

func (suite *DatabaseTestSuite) TestNewFeedWithNonExistingCategory() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		CategoryUUID: uuid.NewV4().String(),
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.IsType(BadRequest{}, err)
}

func (suite *DatabaseTestSuite) TestFeedsFromNonExistingCategory() {
	_, err := suite.db.FeedsFromCategory(&suite.user, uuid.NewV4().String())
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestChangeFeedCategory() {
	first_ctg := models.Category{
		Name: "News",
	}

	second_ctg := models.Category{
		Name: "Tech",
	}

	err := suite.db.NewCategory(&first_ctg, &suite.user)
	err = suite.db.NewCategory(&second_ctg, &suite.user)

	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     first_ctg,
		CategoryID:   first_ctg.ID,
		CategoryUUID: first_ctg.UUID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	feeds, err := suite.db.FeedsFromCategory(&suite.user, first_ctg.UUID)
	suite.Nil(err)
	suite.Require().Len(feeds, 1)
	suite.Equal(feeds[0].UUID, feed.UUID)
	suite.Equal(feeds[0].Title, feed.Title)

	feeds, err = suite.db.FeedsFromCategory(&suite.user, second_ctg.UUID)
	suite.Nil(err)
	suite.Empty(feeds)

	err = suite.db.ChangeFeedCategory(feed.UUID, second_ctg.UUID, &suite.user)
	suite.Nil(err)

	feeds, err = suite.db.FeedsFromCategory(&suite.user, first_ctg.UUID)
	suite.Nil(err)
	suite.Empty(feeds)

	feeds, err = suite.db.FeedsFromCategory(&suite.user, second_ctg.UUID)
	suite.Nil(err)
	suite.Require().Len(feeds, 1)
	suite.Equal(feeds[0].UUID, feed.UUID)
	suite.Equal(feeds[0].Title, feed.Title)
}

func (suite *DatabaseTestSuite) TestFeeds() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			Title:        "Test site " + strconv.Itoa(i),
			Subscription: "http://example.com",
		}

		err := suite.db.NewFeed(&feed, &suite.user)
		suite.Require().Nil(err)
	}

	feeds := suite.db.Feeds(&suite.user)
	suite.Len(feeds, 5)
}

func (suite *DatabaseTestSuite) TestEditFeed() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.UUID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.Title)
	suite.NotZero(query.ID)

	feed.Title = "Testing New Name"
	feed.Subscription = "http://example.com/feed"

	err = suite.db.EditFeed(&feed, &suite.user)
	suite.Nil(err)

	query, err = suite.db.Feed(feed.UUID, &suite.user)
	suite.Nil(err)
	suite.Equal(feed.Title, "Testing New Name")
	suite.Equal(feed.Subscription, "http://example.com/feed")
}

func (suite *DatabaseTestSuite) TestEditNonExistingFeed() {
	err := suite.db.EditFeed(&models.Feed{}, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestConflictingNewCategory() {
	ctg := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)

	err = suite.db.NewCategory(&ctg, &suite.user)
	suite.IsType(Conflict{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteFeed() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	query, err := suite.db.Feed(feed.UUID, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(query.UUID)

	err = suite.db.DeleteFeed(feed.UUID, &suite.user)
	suite.Nil(err)

	_, err = suite.db.Feed(feed.UUID, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestDeleteNonExistingFeed() {
	err := suite.db.DeleteFeed(uuid.NewV4().String(), &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestNewEntry() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotZero(feed.ID)
	suite.NotEmpty(feed.UUID)

	entry := models.Entry{
		Title:       "Test Entry",
		Description: "Testing entry",
		Author:      "chavamee",
		Link:        "http://example.com",
		Mark:        models.Unread,
		FeedUUID:    feed.UUID,
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)
	suite.NotZero(entry.ID)
	suite.NotEmpty(entry.UUID)

	query, err := suite.db.Entry(entry.UUID, &suite.user)
	suite.Nil(err)
	suite.NotZero(query.FeedID)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Len(entries, 1)
	suite.Equal(entries[0].ID, entry.ID)
	suite.Equal(entries[0].Title, entry.Title)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeedWithNonExistenFeed() {
	_, err := suite.db.EntriesFromFeed(uuid.NewV4().String(), true, models.Unread, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestNewEntryWithEmptyFeed() {
	entry := models.Entry{
		Title:       "Test Entry",
		Link:        "http://example.com",
		Description: "Testing entry",
		Author:      "chavamee",
		Mark:        models.Unread,
		FeedUUID:    "",
	}

	err := suite.db.NewEntry(&entry, &suite.user)
	suite.IsType(BadRequest{}, err)
	suite.Zero(entry.ID)
	suite.Empty(entry.UUID)

	query, err := suite.db.Entry(entry.UUID, &suite.user)
	suite.NotNil(err)
	suite.Zero(query.FeedID)
}

func (suite *DatabaseTestSuite) TestNewEntryWithBadFeed() {
	entry := models.Entry{
		Title:       "Test Entry",
		Description: "Testing entry",
		Author:      "chavamee",
		Mark:        models.Unread,
		FeedUUID:    uuid.NewV4().String(),
	}

	err := suite.db.NewEntry(&entry, &suite.user)
	suite.NotNil(err)
	suite.Zero(entry.ID)
	suite.Empty(entry.UUID)

	query, err := suite.db.Entry(entry.UUID, &suite.user)
	suite.NotNil(err)
	suite.Zero(query.FeedID)
}

func (suite *DatabaseTestSuite) TestEntriesFromFeed() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(feed.UUID)

	entry := models.Entry{
		Title:       "Test Entry",
		Description: "Testing entry",
		Author:      "chavamee",
		Link:        "http://example.com",
		Mark:        models.Unread,
		FeedUUID:    feed.UUID,
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Equal(entries[0].ID, entry.ID)
	suite.Equal(entries[0].Title, entry.Title)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Nil(err)
	suite.Empty(entries)
}

func (suite *DatabaseTestSuite) TestEntryWithGUIDExists() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entry := models.Entry{
		Title:    "Test Entry",
		GUID:     "entry@test",
		Feed:     feed,
		FeedID:   feed.ID,
		FeedUUID: feed.UUID,
	}

	err = suite.db.NewEntry(&entry, &suite.user)

	suite.True(suite.db.EntryWithGUIDExists(entry.GUID, &suite.user))
}

func (suite *DatabaseTestSuite) TestEntryWithGUIDDoesNotExists() {
	feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entry := models.Entry{
		Title:    "Test Entry",
		Feed:     feed,
		FeedID:   feed.ID,
		FeedUUID: feed.UUID,
	}

	err = suite.db.NewEntry(&entry, &suite.user)

	suite.False(suite.db.EntryWithGUIDExists("item@test", &suite.user))
}

func (suite *DatabaseTestSuite) TestEntriesFromCategory() {
	first_ctg := models.Category{
		Name: "News",
	}

	second_ctg := models.Category{
		Name: "Tech",
	}

	err := suite.db.NewCategory(&first_ctg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(first_ctg.UUID)

	err = suite.db.NewCategory(&second_ctg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(second_ctg.UUID)

	first_feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     first_ctg,
		CategoryID:   first_ctg.ID,
		CategoryUUID: first_ctg.UUID,
	}

	second_feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     second_ctg,
		CategoryID:   second_ctg.ID,
		CategoryUUID: second_ctg.UUID,
	}

	third_feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     second_ctg,
		CategoryID:   second_ctg.ID,
		CategoryUUID: second_ctg.UUID,
	}

	err = suite.db.NewFeed(&first_feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(first_feed.UUID)

	err = suite.db.NewFeed(&second_feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(second_feed.UUID)

	err = suite.db.NewFeed(&third_feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(second_feed.UUID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:       "First Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "chavamee",
				Link:        "http://example.com",
				Mark:        models.Unread,
				FeedUUID:    first_feed.UUID,
			}
		} else {
			if i < 7 {
				entry = models.Entry{
					Title:       "Second Feed Test Entry " + strconv.Itoa(i),
					Description: "Testing entry " + strconv.Itoa(i),
					Author:      "chavamee",
					Link:        "http://example.com",
					Mark:        models.Unread,
					FeedUUID:    second_feed.UUID,
				}
			} else {
				entry = models.Entry{
					Title:       "Third Feed Test Entry " + strconv.Itoa(i),
					Description: "Testing entry " + strconv.Itoa(i),
					Author:      "chavamee",
					Link:        "http://example.com",
					Mark:        models.Unread,
					FeedUUID:    third_feed.UUID,
				}
			}
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	entries, err := suite.db.EntriesFromCategory(first_ctg.UUID, false, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Len(entries, 5)
	suite.Equal(entries[0].Title, "First Feed Test Entry 0")

	entries, err = suite.db.EntriesFromCategory(second_ctg.UUID, true, models.Unread, &suite.user)
	suite.Nil(err)
	suite.NotEmpty(entries)
	suite.Len(entries, 5)
	suite.Equal(entries[0].Title, "Third Feed Test Entry 9")
	suite.Equal(entries[len(entries)-1].Title, "Second Feed Test Entry 5")
}

func (suite *DatabaseTestSuite) TestEntriesFromNonExistingCategory() {
	_, err := suite.db.EntriesFromCategory(uuid.NewV4().String(), true, models.Unread, &suite.user)
	suite.IsType(NotFound{}, err)
}

func (suite *DatabaseTestSuite) TestMarkCategory() {
	first_ctg := models.Category{
		Name: "News",
	}

	second_ctg := models.Category{
		Name: "Tech",
	}

	err := suite.db.NewCategory(&first_ctg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(first_ctg.UUID)

	err = suite.db.NewCategory(&second_ctg, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(second_ctg.UUID)

	first_feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     first_ctg,
		CategoryID:   first_ctg.ID,
		CategoryUUID: first_ctg.UUID,
	}

	second_feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
		Category:     second_ctg,
		CategoryID:   second_ctg.ID,
		CategoryUUID: second_ctg.UUID,
	}

	err = suite.db.NewFeed(&first_feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(first_feed.UUID)

	err = suite.db.NewFeed(&second_feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(second_feed.UUID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:       "First Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "chavamee",
				Link:        "http://example.com",
				Mark:        models.Unread,
				FeedUUID:    first_feed.UUID,
			}
		} else {
			entry = models.Entry{
				Title:       "Second Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "chavamee",
				Link:        "http://example.com",
				Mark:        models.Read,
				FeedUUID:    second_feed.UUID,
			}
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	suite.Require().Equal(suite.db.db.Model(&suite.user).Association("Entries").Count(), 10)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err = suite.db.MarkCategory(first_ctg.UUID, models.Read, &suite.user)
	suite.Nil(err)

	entries, err := suite.db.EntriesFromCategory(first_ctg.UUID, true, models.Any, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Read)
	}

	err = suite.db.MarkCategory(second_ctg.UUID, models.Unread, &suite.user)
	suite.Nil(err)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries, err = suite.db.EntriesFromCategory(second_ctg.UUID, true, models.Any, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Unread)
	}
}

func (suite *DatabaseTestSuite) TestMarkFeed() {
	first_feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	second_feed := models.Feed{
		Title:        "Test site",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&first_feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(first_feed.UUID)

	err = suite.db.NewFeed(&second_feed, &suite.user)
	suite.Require().Nil(err)
	suite.NotEmpty(second_feed.UUID)

	for i := 0; i < 10; i++ {
		var entry models.Entry
		if i <= 4 {
			entry = models.Entry{
				Title:       "First Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "chavamee",
				Link:        "http://example.com",
				Mark:        models.Unread,
				FeedUUID:    first_feed.UUID,
			}
		} else {
			entry = models.Entry{
				Title:       "Second Feed Test Entry " + strconv.Itoa(i),
				Description: "Testing entry " + strconv.Itoa(i),
				Author:      "chavamee",
				Link:        "http://example.com",
				Mark:        models.Read,
				FeedUUID:    second_feed.UUID,
			}
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	suite.Require().Equal(suite.db.db.Model(&suite.user).Association("Entries").Count(), 10)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 5)
	suite.Require().Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	err = suite.db.MarkFeed(first_feed.UUID, models.Read, &suite.user)
	suite.Nil(err)

	entries, err := suite.db.EntriesFromFeed(first_feed.UUID, true, models.Any, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Read).Association("Entries").Count(), 10)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Read)
	}

	err = suite.db.MarkFeed(second_feed.UUID, models.Unread, &suite.user)
	suite.Nil(err)

	suite.Equal(suite.db.db.Model(&suite.user).Where("mark = ?", models.Unread).Association("Entries").Count(), 5)

	entries, err = suite.db.EntriesFromFeed(second_feed.UUID, true, models.Any, &suite.user)
	suite.Nil(err)
	suite.Len(entries, 5)

	for _, entry := range entries {
		suite.EqualValues(entry.Mark, models.Unread)
	}
}

func (suite *DatabaseTestSuite) TestMarkEntry() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://localhost/news",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	entry := models.Entry{
		Title:    "Article",
		Feed:     feed,
		FeedID:   feed.ID,
		FeedUUID: feed.UUID,
		Mark:     models.Unread,
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 1)

	err = suite.db.MarkEntry(entry.UUID, models.Read, &suite.user)
	suite.Require().Nil(err)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 1)
}

func (suite *DatabaseTestSuite) TestStats() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:    "Item",
			Link:     "http://example.com",
			Feed:     feed,
			FeedID:   feed.ID,
			FeedUUID: feed.UUID,
			Mark:     models.Read,
			Saved:    true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:    "Item",
			Link:     "http://example.com",
			Feed:     feed,
			FeedID:   feed.ID,
			FeedUUID: feed.UUID,
			Mark:     models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	stats := suite.db.Stats(&suite.user)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestFeedStats() {
	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:    "Item",
			Link:     "http://example.com",
			Feed:     feed,
			FeedID:   feed.ID,
			FeedUUID: feed.UUID,
			Mark:     models.Read,
			Saved:    true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:    "Item",
			Link:     "http://example.com",
			Feed:     feed,
			FeedID:   feed.ID,
			FeedUUID: feed.UUID,
			Mark:     models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	stats, err := suite.db.FeedStats(feed.UUID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}

func (suite *DatabaseTestSuite) TestCategoryStats() {
	category := models.Category{
		Name: "World",
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)

	feed := models.Feed{
		Title:        "News",
		Subscription: "http://example.com",
		Category:     category,
		CategoryID:   category.ID,
		CategoryUUID: category.UUID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(feed.UUID)

	for i := 0; i < 3; i++ {
		entry := models.Entry{
			Title:    "Item",
			Link:     "http://example.com",
			Feed:     feed,
			FeedID:   feed.ID,
			FeedUUID: feed.UUID,
			Mark:     models.Read,
			Saved:    true,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	for i := 0; i < 7; i++ {
		entry := models.Entry{
			Title:    "Item",
			Link:     "http://example.com",
			Feed:     feed,
			FeedID:   feed.ID,
			FeedUUID: feed.UUID,
			Mark:     models.Unread,
		}

		err = suite.db.NewEntry(&entry, &suite.user)
		suite.Require().Nil(err)
	}

	stats, err := suite.db.CategoryStats(category.UUID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(7, stats.Unread)
	suite.Equal(3, stats.Read)
	suite.Equal(3, stats.Saved)
	suite.Equal(10, stats.Total)
}
func TestNewDB(t *testing.T) {
	_, err := NewDB("sqlite3", TestDatabasePath)
	assert.Nil(t, err)
	defer os.Remove(TestDatabasePath)
}

func TestNewDBWithBadOptions(t *testing.T) {
	_, err := NewDB("bogus", TestDatabasePath)
	assert.NotNil(t, err)
	defer os.Remove(TestDatabasePath)
}

func TestNewUser(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)
	defer os.Remove(TestDatabasePath)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	user, err := db.UserFromName("test")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)
}

func TestUsers(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)
	defer os.Remove(TestDatabasePath)

	err = db.NewUser("test_one", "golang")
	assert.Nil(t, err)

	err = db.NewUser("test_two", "password")
	assert.Nil(t, err)

	users := db.Users()
	assert.Len(t, users, 2)
}

func TestUsersWithFields(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)
	defer os.Remove(TestDatabasePath)

	err = db.NewUser("test_one", "golang")
	assert.Nil(t, err)

	err = db.NewUser("test_two", "password")
	assert.Nil(t, err)

	users := db.Users("uncategorized_category_uuid", "saved_category_uuid")
	assert.Len(t, users, 2)
	assert.NotEmpty(t, users[0].SavedCategoryUUID)
	assert.NotEmpty(t, users[0].UncategorizedCategoryUUID)
}

func TestNewConflictingUsers(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)
	defer os.Remove(TestDatabasePath)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	err = db.NewUser("test", "password")
	assert.IsType(t, Conflict{}, err)
}

func TestSuccessfulAuthentication(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)
	defer os.Remove(TestDatabasePath)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	user, err := db.UserFromName("test")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	user, err = db.Authenticate("test", "golang")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)
}

func TestBadPasswordAuthentication(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)
	defer os.Remove(TestDatabasePath)

	err = db.NewUser("test", "golang")
	assert.Nil(t, err)

	user, err := db.UserFromName("test")
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	user, err = db.Authenticate("test", "badpass")
	assert.IsType(t, Unauthorized{}, err)
}

func TestBadUserAuthentication(t *testing.T) {
	db, err := NewDB("sqlite3", TestDatabasePath)
	require.Nil(t, err)
	defer os.Remove(TestDatabasePath)

	_, err = db.Authenticate("test", "golang")
	assert.IsType(t, NotFound{}, err)
}

func TestDatabaseTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseTestSuite))
}
