package dgraphdockerhelper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/test-go/testify/require"
)

const (
	DefaultDgraphImage = "dgraph/standalone:v21.03.2"
)

type DgraphConfig struct {
	Port        int
	ContainerID string
}

func (cfg *DgraphConfig) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", cfg.Port)
}

// DgraphStart creates and starts a dgraph container. You can specify a
// specific `image`. The image must be in your Docker cache (docker pull <image-name>).
func DgraphStart(t *testing.T, image string) *DgraphConfig {
	var (
		err    error
		config DgraphConfig
	)
	t.Log("Starting dgraph container")
	if image == "" {
		image = DefaultDgraphImage
	}
	require := require.New(t)
	config.Port, err = getFreePort()
	require.NoError(err)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(err)

	env := []string{}
	cfg := &container.Config{
		Image: image,
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", config.Port)): struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			"8080/tcp": []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", config.Port),
				},
			},
		},
	}

	cont, err := cli.ContainerCreate(
		context.Background(),
		cfg,
		hostConfig,
		nil,
		nil,
		"",
	)
	require.NoError(err)

	err = cli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	require.NoError(err)
	config.ContainerID = cont.ID

	// Wait for the container to start
	time.Sleep(time.Second * 1)

	// Wait for it to be ready
	for i := 0; i < 30; i++ {
		resp, err := http.Get(config.GetURL())
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(time.Second)
	}

	return &config
}

// DgraphStop stops and removes a Dgraph container.
func DgraphStop(t *testing.T, config *DgraphConfig) {
	t.Log("Stopping and removing dgraph container")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)
	err = cli.ContainerStop(context.Background(), config.ContainerID, nil)
	require.NoError(t, err)
	err = cli.ContainerRemove(context.Background(), config.ContainerID, types.ContainerRemoveOptions{})
	require.NoError(t, err)
}

type dgraphUpdateResponse struct {
	Data struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// DgraphLoadSchema loads a schema into dgraph.
func DgraphLoadSchema(t *testing.T, config *DgraphConfig, schema string) {
	t.Log("Loading schema")

	url := fmt.Sprintf("http://localhost:%d/admin/schema", config.Port)
	for i := 0; i < 30; i++ {
		var schemaUpdateResponse dgraphUpdateResponse
		buffer := bytes.NewBuffer([]byte(schema))
		resp, err := http.Post(url, "text/plain", buffer)
		require.NoError(t, err)
		require.Equal(t, resp.StatusCode, http.StatusOK)
		err = json.NewDecoder(resp.Body).Decode(&schemaUpdateResponse)
		require.NoError(t, err)
		resp.Body.Close()
		if schemaUpdateResponse.Data.Code == "Success" {
			time.Sleep(500 * time.Millisecond)
			return
		}
		if len(schemaUpdateResponse.Errors) > 0 {
			if strings.Contains(schemaUpdateResponse.Errors[0].Message, "not ready") {
				time.Sleep(time.Second)
				continue
			}
			t.Error(schemaUpdateResponse.Errors[0].Message)
			return
		}
		t.Logf("Warning, unexpected response %++v\n", schemaUpdateResponse)
		time.Sleep(time.Second)
	}
}

// DgraphDropData deletes all data from the graph. The schema is left intact.
func DgraphDropData(t *testing.T, config *DgraphConfig) {
	var opUpdateResponse dgraphUpdateResponse
	// curl -X POST localhost:8080/alter -d '{"drop_op": "DATA"}'
	url := fmt.Sprintf("http://localhost:%d/alter", config.Port)
	buffer := bytes.NewBuffer([]byte("{\"drop_op\": \"DATA\"}}"))
	resp, err := http.Post(url, "text/plain", buffer)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)
	err = json.NewDecoder(resp.Body).Decode(&opUpdateResponse)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, "Success", opUpdateResponse.Data.Code)
}

// DgraphDropAll drops all data AND the schema from the graph.
func DgraphDropAll(t *testing.T, config *DgraphConfig) {
	var opUpdateResponse dgraphUpdateResponse
	// curl -X POST localhost:8080/alter -d '{"drop_all": true}'
	url := fmt.Sprintf("http://localhost:%d/alter", config.Port)
	buffer := bytes.NewBuffer([]byte("{\"drop_all\": true}}"))
	resp, err := http.Post(url, "text/plain", buffer)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)
	err = json.NewDecoder(resp.Body).Decode(&opUpdateResponse)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, "Success", opUpdateResponse.Data.Code)
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
