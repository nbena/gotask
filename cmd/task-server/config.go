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

package server

// Config is the configuration used by the server.
type Config struct {
	ListenAddr string `json:"listenAddr"`
	ListenPort int    `json:"listenPort"`
	TaskFile   string `json:"taskFile"`
	AllowVars  bool   `json:"allowVars"`

	UseTLS      bool   `json:"useTLS"`
	TLSKeyPath  string `json:"tlsKeyPath"`
	TLSCertPath string `json:"tlsCertPath"`

	LogRequests bool `json:"logRequests"`
}

// RuntimeConfig keeps only the info we need at
// runtime.
type RuntimeConfig struct {
	taskFilePath string
	logRequests  bool
}
