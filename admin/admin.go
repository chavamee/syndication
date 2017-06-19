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

//TODO: Consider SO_PEERCRED with Unix Sockets

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"reflect"
	"sync"

	"github.com/chavamee/syndication/database"
)

const DefaultSocketPath = "/var/run/syndication/control"

var (
	commands = []string{
		"CreateUser",
		"DeleteUser",
		"NewUser",
	}
)

type State int

const (
	Started = iota
	Running
	Stopping
	Restarting
)

type (
	AdminSocket struct {
		ln       *net.UnixListener
		pathname string
		state    chan State
		stopping chan State
		db       *database.DB
		mux      sync.Mutex
	}

	Command struct {
		Cmd    string                 `json:"command"`
		Params map[string]interface{} `json:"params"`
	}

	Resp struct {
		Error string
	}
)

// NewAdminSocket creates a new admin socket and initialize it
func NewAdminSocket(db *database.DB, socketPath string) (socket AdminSocket, err error) {
	if socketPath != "" {
		socket.pathname = socketPath
	} else {
		socket.pathname = DefaultSocketPath
	}

	if _, err := os.Stat(socket.pathname); err == nil {
		os.Remove(socket.pathname)
	}

	socket.ln, err = net.ListenUnix("unixpacket", &net.UnixAddr{
		Name: socket.pathname,
		Net:  "unixpacket"})
	socket.state = make(chan State)
	socket.db = db

	if err != nil {
		log.Fatalln(err)
	}

	return
}

func (s *AdminSocket) Start() {
	go s.listen()
	s.state <- Running
}

func (s *AdminSocket) listen() {
	defer os.Remove(DefaultSocketPath)
	for {
		switch <-s.state {
		case Running:
			conn, err := s.ln.AcceptUnix()
			if err != nil {
				continue
			}

			if conn == nil {
				log.Fatal("Connection could not be accepted: ", err)
			}

			go func() {
				err := s.handleConnection(conn)
				if err != nil {
					_ = json.NewEncoder(conn).Encode(&Resp{
						Error: err.Error(),
					})
				} else {
					conn.Write([]byte("OK"))
				}
				conn.Close()
			}()
		case Stopping:
			return
		}
	}
}

func (s *AdminSocket) handleConnection(conn *net.UnixConn) error {
	s.state <- Running
	var cmd Command
	if err := json.NewDecoder(conn).Decode(&cmd); err != nil {
		return errors.New("Invalid cmd")
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	br := Bridge{s.db}

	method := reflect.ValueOf(&br).MethodByName(cmd.Cmd)
	if method.IsValid() {
		rtVals := method.Call([]reflect.Value{reflect.ValueOf(cmd.Params)})
		if !rtVals[0].IsNil() {
			return rtVals[0].Interface().(error)
		}

		return nil
	}

	return errors.New("Invalid Request")
}

func (s *AdminSocket) Stop() {
	s.ln.Close()
	s.state <- Stopping
}
