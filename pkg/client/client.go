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

package client

import (
	"fmt"
	"io"
	"net/http"
)

// TaskClient is used to do request to the server.
type TaskClient struct {
	config *Config
	client *http.Client
}

// NewTaskClient returns a new TaskClient.
func NewTaskClient(config *Config) *TaskClient {

	return &TaskClient{
		config: config,
		// in future there will be more
		// options
		client: &http.Client{},
	}
}

func (c *TaskClient) request(method, postfix string,
	expectedStatus int, body io.ReadCloser) (*http.Response, error) {

	uri := fmt.Sprintf("http://%s:%d/%s",
		c.config.ServerAddr, c.config.ServerPort, postfix)

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != expectedStatus {
		return nil, err
	}

	return resp, nil
}
