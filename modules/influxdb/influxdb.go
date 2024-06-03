package influxdb

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// defaultImage {
const defaultImage = "influxdb:1.8"

// }

// Container represents the MySQL container type used in the module
type Container struct {
	*testcontainers.DockerContainer
}

// RunContainer creates an instance of the InfluxDB container type
func RunContainer(ctx context.Context, opts ...testcontainers.RequestCustomizer) (*Container, error) {
	req := testcontainers.Request{
		Image:        defaultImage,
		ExposedPorts: []string{"8086/tcp", "8088/tcp"},
		Env: map[string]string{
			"INFLUXDB_BIND_ADDRESS":          ":8088",
			"INFLUXDB_HTTP_BIND_ADDRESS":     ":8086",
			"INFLUXDB_REPORTING_DISABLED":    "true",
			"INFLUXDB_MONITOR_STORE_ENABLED": "false",
			"INFLUXDB_HTTP_HTTPS_ENABLED":    "false",
			"INFLUXDB_HTTP_AUTH_ENABLED":     "false",
		},
		WaitingFor: wait.ForListeningPort("8086/tcp"),
		Started:    true,
	}

	for _, opt := range opts {
		opt.Customize(&req)
	}

	hasInitDb := false

	for _, f := range req.Files {
		if f.ContainerFilePath == "/" && strings.HasSuffix(f.HostFilePath, "docker-entrypoint-initdb.d") {
			// Init service in container will start influxdb, run scripts in docker-entrypoint-initdb.d and then
			// terminate the influxdb server, followed by restart of influxdb.  This is tricky to wait for, and
			// in this case, we are assuming that data was added by init script, so we then look for an
			// "Open shard" which is the last thing that happens before the server is ready to accept connections.
			// This is probably different for InfluxDB 2.x, but that is left as an exercise for the reader.
			strategies := []wait.Strategy{
				req.WaitingFor,
				wait.ForLog("influxdb init process in progress..."),
				wait.ForLog("Server shutdown completed"),
				wait.ForLog("Opened shard"),
			}
			req.WaitingFor = wait.ForAll(strategies...)
			hasInitDb = true
			break
		}
	}

	if !hasInitDb {
		if lastIndex := strings.LastIndex(req.Image, ":"); lastIndex != -1 {
			tag := req.Image[lastIndex+1:]
			if tag == "latest" || tag[0] == '2' {
				req.WaitingFor = wait.ForLog(`Listening log_id=[0-9a-zA-Z_~]+ service=tcp-listener transport=http`).AsRegexp()
			}
		} else {
			req.WaitingFor = wait.ForLog("Listening for signals")
		}
	}

	container, err := testcontainers.New(ctx, req)
	if err != nil {
		return nil, err
	}

	return &Container{container}, nil
}

func (c *Container) MustConnectionUrl(ctx context.Context) string {
	connectionString, err := c.ConnectionUrl(ctx)
	if err != nil {
		panic(err)
	}
	return connectionString
}

func (c *Container) ConnectionUrl(ctx context.Context) (string, error) {
	containerPort, err := c.MappedPort(ctx, "8086/tcp")
	if err != nil {
		return "", err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s:%s", host, containerPort.Port()), nil
}

func WithUsername(username string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.Request) error {
		req.Env["INFLUXDB_USER"] = username
		return nil
	}
}

func WithPassword(password string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.Request) error {
		req.Env["INFLUXDB_PASSWORD"] = password
		return nil
	}
}

func WithDatabase(database string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.Request) error {
		req.Env["INFLUXDB_DATABASE"] = database
		return nil
	}
}

func WithConfigFile(configFile string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.Request) error {
		cf := testcontainers.ContainerFile{
			HostFilePath:      configFile,
			ContainerFilePath: "/etc/influxdb/influxdb.conf",
			FileMode:          0o755,
		}
		req.Files = append(req.Files, cf)
		return nil
	}
}

// WithInitDb will copy a 'docker-entrypoint-initdb.d' directory to the container.
// The secPath is the path to the directory on the host machine.
// The directory will be copied to the root of the container.
func WithInitDb(srcPath string) testcontainers.CustomizeRequestOption {
	return func(req *testcontainers.Request) error {
		cf := testcontainers.ContainerFile{
			HostFilePath:      path.Join(srcPath, "docker-entrypoint-initdb.d"),
			ContainerFilePath: "/",
			FileMode:          0o755,
		}
		req.Files = append(req.Files, cf)
		return nil
	}
}
