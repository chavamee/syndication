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
	"errors"
	"reflect"

	"github.com/chavamee/syndication/database"
)

type (
	Bridge struct {
		db *database.DB
	}
)

func (br *Bridge) NewUser(args map[string]interface{}) error {
	var username string
	var password string

	aVal := reflect.ValueOf(args["username"])
	if aVal.IsValid() && aVal.Kind() == reflect.String {
		username = aVal.String()
	} else {
		return errors.New("Invalid Param")
	}

	bVal := reflect.ValueOf(args["password"])
	if bVal.IsValid() && bVal.Kind() == reflect.String {
		password = bVal.String()
	} else {
		return errors.New("Invalid Param")
	}

	return br.db.NewUser(username, password)
}

func (br *Bridge) DeleteUser(args map[string]interface{}) error {
	var userID string

	aVal := reflect.ValueOf(args["userID"])
	if aVal.IsValid() && aVal.Kind() == reflect.String {
		userID = aVal.String()
	} else {
		return errors.New("Invalid Param")
	}

	return br.db.DeleteUser(userID)
}

func (br *Bridge) ChangeUserName(args map[string]interface{}) error {
	return nil
}

func (br *Bridge) ChangeUserPassword(args map[string]interface{}) error {
	return nil
}

func (br *Bridge) GetUsers(args map[string]interface{}) error {
	return nil
}

func (br *Bridge) GetUser(args map[string]interface{}) error {
	return nil
}

func (br *Bridge) AssignNewAPIKey(args map[string]interface{}) error {
	return nil
}

func getUser(args map[string]interface{}) (string, error) {
	aVal := reflect.ValueOf(args["username"])
	if aVal.IsValid() && aVal.Kind() == reflect.String {
		return aVal.String(), nil
	} else {
		return "", errors.New("Invalid Param")
	}
}
