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

package admin

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/chavamee/syndication/database"
)

type (
	AdminTestSuite struct {
		suite.Suite

		db         database.DB
		admin      AdminSocket
		conn       net.Conn
		socketPath string
	}
)

func (suite *AdminTestSuite) SetupTest() {
	var err error
	suite.db, err = database.NewDB("sqlite3", "/tmp/syndication-test.db")
	suite.Nil(err)

	suite.socketPath = "/tmp/syndication.socket"
	suite.admin, err = NewAdminSocket(&suite.db, suite.socketPath)
	suite.Nil(err)

	go suite.admin.Start()

	suite.conn, err = net.Dial("unixpacket", suite.socketPath)
	suite.Nil(err)
}

func (suite *AdminTestSuite) TearDownTest() {
	suite.admin.Stop()
	suite.conn.Close()
	suite.db.Close()

	os.Remove(suite.db.Path)
}

func (suite *AdminTestSuite) TestNewUser() {
	message := `
	{
		"command": "NewUser",
		"params": {"username":"GoTest",
							 "password":"testtesttest"}
	}
	`
	size, err := suite.conn.Write([]byte(message))
	suite.Nil(err)
	suite.Equal(len(message), size)

	buff := make([]byte, 256)
	size, err = suite.conn.Read(buff)

	users := suite.db.Users("username")
	suite.Len(users, 1)

	suite.Equal(users[0].Username, "GoTest")
	suite.NotEmpty(users[0].ID)
	suite.NotEmpty(users[0].UUID)
}

func TestAdminTestSuite(t *testing.T) {
	suite.Run(t, new(AdminTestSuite))
}
