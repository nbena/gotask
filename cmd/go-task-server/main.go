// go-task, a simple client-server task runner
// Copyright (C) 2018 nbena
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"

	"github.com/nbena/gotask/pkg/server"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Missing path to configuration file\n")
		os.Exit(-1)
	}

	configFile := os.Args[1]

	config, err := server.ReadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in config: %s\n", err.Error())
		os.Exit(-2)
	}

	server, err := server.NewServer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error in start server: %s\n", err.Error())
		os.Exit(-2)
	}

	server.Run()
}
