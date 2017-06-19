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

type DatabaseError interface {
	String() string
	Code() int
	Error() string
}

type (
	Conflict struct {
		msg string
	}

	NotFound struct {
		msg string
	}

	BadRequest struct {
		msg string
	}

	Unauthorized struct {
		msg string
	}
)

func (e Conflict) Error() string {
	return e.msg
}

func (e Conflict) String() string {
	return "Conflict"
}

func (e Conflict) Code() int {
	return 409
}

func (e NotFound) Error() string {
	return e.msg
}

func (e NotFound) String() string {
	return "NotFound"
}

func (e NotFound) Code() int {
	return 404
}

func (e BadRequest) Error() string {
	return e.msg
}

func (e BadRequest) String() string {
	return "BadRequest"
}

func (e BadRequest) Code() int {
	return 400
}

func (e Unauthorized) Error() string {
	return e.msg
}

func (e Unauthorized) String() string {
	return "Unauthorized"
}

func (e Unauthorized) Code() int {
	return 401
}
