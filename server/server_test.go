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

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/chavamee/syndication/config"
	"github.com/chavamee/syndication/database"
	"github.com/chavamee/syndication/models"
	"github.com/chavamee/syndication/sync"
	"github.com/stretchr/testify/suite"
)

type (
	ServerTestSuite struct {
		suite.Suite

		db     database.DB
		sync   sync.Sync
		server Server
		user   models.User
		token  string
	}
)

func (suite *ServerTestSuite) SetupTest() {
	dbPath := "/tmp/syndication-test.db"
	dbType := "sqlite3"

	conf := config.NewDefaultConfig()

	var err error
	suite.db, err = database.NewDB(dbType, dbPath)
	suite.Nil(err)

	suite.sync = sync.NewSync(&suite.db)
	suite.server = NewServer(&suite.db, &suite.sync, conf.Server, conf.Security)
	suite.server.handle.HideBanner = true

	go suite.server.Start()

	resp, err := http.PostForm("http://localhost:8080/v1/register",
		url.Values{"username": {"GoTest"}, "password": {"testtesttest"}})
	suite.Nil(err)

	suite.Equal(resp.StatusCode, 204)

	users := suite.db.Users("username")
	suite.Len(users, 1)

	suite.Equal(users[0].Username, "GoTest")
	suite.NotEmpty(users[0].ID)
	suite.NotEmpty(users[0].UUID)

	resp.Body.Close()

	resp, err = http.PostForm("http://localhost:8080/v1/login",
		url.Values{"username": {"GoTest"}, "password": {"testtesttest"}})
	suite.Nil(err)

	defer resp.Body.Close()

	suite.Equal(resp.StatusCode, 200)

	type Token struct {
		Token string `json:"token"`
	}

	var t Token
	err = json.NewDecoder(resp.Body).Decode(&t)
	suite.Nil(err)
	suite.NotEmpty(t.Token)

	suite.token = t.Token
	suite.user, err = suite.db.UserFromName("GoTest")
	suite.Nil(err)
}

func (suite *ServerTestSuite) TearDownTest() {
	suite.server.Stop()

	os.Remove(suite.db.Path)
}

func (suite *ServerTestSuite) TestAddFeed() {
	payload := []byte(`{"title":"EFF", "subscription": "https://www.eff.org/rss/updates.xml"}`)
	req, err := http.NewRequest("POST", "http://localhost:8080/v1/feeds", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(201, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	suite.Require().Nil(err)

	suite.Require().NotEmpty(respFeed.UUID)
	suite.NotEmpty(respFeed.Title)

	dbFeed, err := suite.db.Feed(respFeed.UUID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(dbFeed.Title, respFeed.Title)

}

func (suite *ServerTestSuite) TestGetFeeds() {
	for i := 0; i < 5; i++ {
		feed := models.Feed{
			Title:        "Feed " + strconv.Itoa(i+1),
			Subscription: "http://example.com/feed",
		}
		err := suite.db.NewFeed(&feed, &suite.user)
		suite.Require().Nil(err)
		suite.Require().NotZero(feed.ID)
		suite.Require().NotEmpty(feed.UUID)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	suite.Require().Nil(err)
	suite.Len(respFeeds.Feeds, 5)
}

func (suite *ServerTestSuite) TestGetFeed() {
	feed := models.Feed{
		Title:        "EFF",
		Subscription: "https://www.eff.org/rss/updates.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds/"+feed.UUID, nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respFeed := new(models.Feed)
	err = json.NewDecoder(resp.Body).Decode(respFeed)
	suite.Require().Nil(err)

	suite.Equal(respFeed.Title, feed.Title)
	suite.Equal(respFeed.UUID, feed.UUID)
}

func (suite *ServerTestSuite) TestEditFeed() {
	feed := models.Feed{Title: "EFF", Subscription: "https://www.eff.org/rss/updates.xml"}
	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	payload := []byte(`{"title": "EFF Updates"}`)
	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/feeds/"+feed.UUID, bytes.NewBuffer(payload))
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	respFeed, err := suite.db.Feed(feed.UUID, &suite.user)
	suite.Nil(err)
	suite.Equal(respFeed.Title, "EFF Updates")
}

func (suite *ServerTestSuite) TestDeleteFeed() {
	feed := models.Feed{Title: "EFF", Subscription: "https://www.eff.org/rss/updates.xml"}
	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	req, err := http.NewRequest("DELETE", "http://localhost:8080/v1/feeds/"+feed.UUID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	_, err = suite.db.Feed(feed.UUID, &suite.user)
	suite.NotNil(err)
	suite.IsType(database.NotFound{}, err)
}

func (suite *ServerTestSuite) TestGetEntriesFromFeed() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:8080</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:8080/item_1</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:8080/item_2</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:8080/item_3</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:8080/item_4</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:8080/item_5</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	feed := models.Feed{
		Subscription: ts.URL,
		User:         suite.user,
		UserID:       suite.user.ID,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds/"+feed.UUID+"/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestMarkFeed() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:8080</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:8080/item_1</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:8080/item_2</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:8080/item_3</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:8080/item_4</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:8080/item_5</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	feed := models.Feed{
		Subscription: ts.URL,
		User:         suite.user,
		UserID:       suite.user.ID,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/feeds/"+feed.UUID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)
}

func (suite *ServerTestSuite) TestAddCategory() {
	payload := []byte(`{"name": "News"}`)
	req, err := http.NewRequest("POST", "http://localhost:8080/v1/categories", bytes.NewBuffer(payload))
	suite.Require().Nil(err)
	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(201, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	suite.Require().Nil(err)

	suite.Require().NotEmpty(respCtg.UUID)
	suite.NotEmpty(respCtg.Name)

	dbCtg, err := suite.db.Category(respCtg.UUID, &suite.user)
	suite.Nil(err)
	suite.Equal(dbCtg.Name, respCtg.Name)
}

func (suite *ServerTestSuite) TestGetCategories() {
	for i := 0; i < 5; i++ {
		ctg := models.Category{
			Name: "Category " + strconv.Itoa(i+1),
		}
		err := suite.db.NewCategory(&ctg, &suite.user)
		suite.Require().Nil(err)
		suite.Require().NotZero(ctg.ID)
		suite.Require().NotEmpty(ctg.UUID)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Categories struct {
		Categories []models.Category `json:"categories"`
	}

	respCtgs := new(Categories)
	err = json.NewDecoder(resp.Body).Decode(respCtgs)
	suite.Require().Nil(err)

	suite.Len(respCtgs.Categories, 7)
}

func (suite *ServerTestSuite) TestGetCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.UUID)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+ctg.UUID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respCtg := new(models.Category)
	err = json.NewDecoder(resp.Body).Decode(respCtg)
	suite.Require().Nil(err)

	suite.Equal(respCtg.Name, ctg.Name)
	suite.Equal(respCtg.UUID, ctg.UUID)
}

func (suite *ServerTestSuite) TestEditCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.UUID)

	payload := []byte(`{"name": "World News"}`)
	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/categories/"+ctg.UUID, bytes.NewBuffer(payload))
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	editedCtg, err := suite.db.Category(ctg.UUID, &suite.user)
	suite.Require().Nil(err)
	suite.Equal(editedCtg.Name, "World News")
}

func (suite *ServerTestSuite) TestDeleteCategory() {
	ctg := models.Category{Name: "News"}
	err := suite.db.NewCategory(&ctg, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(ctg.ID)
	suite.Require().NotEmpty(ctg.UUID)

	req, err := http.NewRequest("DELETE", "http://localhost:8080/v1/categories/"+ctg.UUID, nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	_, err = suite.db.Category(ctg.UUID, &suite.user)
	suite.NotNil(err)
	suite.IsType(database.NotFound{}, err)
}

func (suite *ServerTestSuite) TestGetFeedsFromCategory() {
	category := models.Category{
		Name: "News",
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)

	feed := models.Feed{
		Title:        "Test feed",
		Subscription: "http://localhost:8080",
		CategoryUUID: category.UUID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+category.UUID+"/feeds", nil)
	suite.Require().Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Feeds struct {
		Feeds []models.Feed `json:"feeds"`
	}

	respFeeds := new(Feeds)
	err = json.NewDecoder(resp.Body).Decode(respFeeds)
	suite.Require().Nil(err)
	suite.Len(respFeeds.Feeds, 1)
}

func (suite *ServerTestSuite) TestGetEntriesFromCategory() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:8080</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:8080/item_1</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:8080/item_2</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:8080/item_3</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:8080/item_4</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:8080/item_5</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	category := models.Category{
		Name:   "News",
		User:   suite.user,
		UserID: suite.user.ID,
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)
	suite.Require().NotZero(category.ID)

	feed := models.Feed{
		Subscription: ts.URL,
		User:         suite.user,
		UserID:       suite.user.ID,
		Category:     category,
		CategoryID:   category.ID,
		CategoryUUID: category.UUID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncCategory(&category)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+category.UUID+"/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestMarkCategory() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:8080</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:8080/item_1</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:8080/item_2</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:8080/item_3</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:8080/item_4</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:8080/item_5</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	category := models.Category{
		Name:   "News",
		User:   suite.user,
		UserID: suite.user.ID,
	}

	err := suite.db.NewCategory(&category, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotEmpty(category.UUID)

	feed := models.Feed{
		Subscription: ts.URL,
		User:         suite.user,
		UserID:       suite.user.ID,
		Category:     category,
		CategoryID:   category.ID,
		CategoryUUID: category.UUID,
	}

	err = suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncCategory(&category)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromCategory(category.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	entries, err = suite.db.EntriesFromCategory(category.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/categories/"+category.UUID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromCategory(category.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromCategory(category.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)
}

func (suite *ServerTestSuite) TestGetEntries() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:8080</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:8080/item_1</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:8080/item_2</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:8080/item_3</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:8080/item_4</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:8080/item_5</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	feed := models.Feed{
		Subscription: ts.URL,
		User:         suite.user,
		UserID:       suite.user.ID,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed)
	suite.Require().Nil(err)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/entries", nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	type Entries struct {
		Entries []models.Entry `json:"entries"`
	}

	respEntries := new(Entries)
	err = json.NewDecoder(resp.Body).Decode(respEntries)
	suite.Require().Nil(err)
	suite.Len(respEntries.Entries, 5)
}

func (suite *ServerTestSuite) TestGetEntry() {
	feed := models.Feed{
		Title:        "EFF",
		Subscription: "https://www.eff.org/rss/updates.xml",
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	entry := models.Entry{
		Title:    "The Espionage Acts Troubling Origins",
		Link:     "https://www.eff.org/deeplinks/2017/06/one-hundred-years-espionage-act",
		Feed:     feed,
		FeedID:   feed.ID,
		FeedUUID: feed.UUID,
	}

	err = suite.db.NewEntry(&entry, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(entry.ID)
	suite.Require().NotEmpty(entry.UUID)

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/entries/"+entry.UUID, nil)
	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respEntry := new(models.Entry)
	err = json.NewDecoder(resp.Body).Decode(respEntry)
	suite.Require().Nil(err)

	suite.Equal(entry.Title, respEntry.Title)
	suite.Equal(entry.UUID, respEntry.UUID)
}

func (suite *ServerTestSuite) TestMarkEntry() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<rss>
		<channel>
    <title>RSS Test</title>
    <link>http://localhost:8080</link>
    <description>Testing rss feeds</description>
    <language>en</language>
    <lastBuildDate></lastBuildDate>
    <item>
      <title>Item 1</title>
      <link>http://localhost:8080/item_1</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item1@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 2</title>
      <link>http://localhost:8080/item_2</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item2@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 3</title>
      <link>http://localhost:8080/item_3</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item3@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 4</title>
      <link>http://localhost:8080/item_4</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item4@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
    <item>
      <title>Item 5</title>
      <link>http://localhost:8080/item_5</link>
      <description>Single test item</description>
      <author>chavamee</author>
      <guid>item5@test</guid>
      <pubDate></pubDate>
      <source>http://localhost:8080/rss.xml</source>
    </item>
		</channel>
		</rss>`)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	feed := models.Feed{
		Subscription: ts.URL,
		User:         suite.user,
		UserID:       suite.user.ID,
	}

	err := suite.db.NewFeed(&feed, &suite.user)
	suite.Require().Nil(err)
	suite.Require().NotZero(feed.ID)
	suite.Require().NotEmpty(feed.UUID)

	err = suite.server.sync.SyncFeed(&feed)
	suite.Require().Nil(err)

	entries, err := suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 0)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 5)

	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/entries/"+entries[0].UUID+"/mark?as=read", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(204, resp.StatusCode)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Unread, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 4)

	entries, err = suite.db.EntriesFromFeed(feed.UUID, true, models.Read, &suite.user)
	suite.Require().Nil(err)
	suite.Require().Len(entries, 1)
}

func (suite *ServerTestSuite) TestGetStatsForFeed() {
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

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/feeds/"+feed.UUID+"/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestGetStats() {
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

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/entries/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestGetStatsForCategory() {
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

	req, err := http.NewRequest("GET", "http://localhost:8080/v1/categories/"+category.UUID+"/stats", nil)

	suite.Nil(err)

	req.Header.Set("Authorization", "Bearer "+suite.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	suite.Require().Nil(err)
	defer resp.Body.Close()

	suite.Equal(200, resp.StatusCode)

	respStats := new(models.Stats)
	err = json.NewDecoder(resp.Body).Decode(respStats)
	suite.Require().Nil(err)

	suite.Equal(7, respStats.Unread)
	suite.Equal(3, respStats.Read)
	suite.Equal(3, respStats.Saved)
	suite.Equal(10, respStats.Total)
}

func (suite *ServerTestSuite) TestAddFeedsToCategory() {

}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
