package gcloud

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RunFirestoreContainer creates an instance of the GCloud container type for Firestore
func RunFirestoreContainer(ctx context.Context, opts ...testcontainers.RequestCustomizer) (*Container, error) {
	req := testcontainers.Request{
		Image:        "gcr.io/google.com/cloudsdktool/cloud-sdk:367.0.0-emulators",
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForLog("running"),
		Started:      true,
	}

	settings, err := applyOptions(&req, opts)
	if err != nil {
		return nil, err
	}

	req.Cmd = []string{
		"/bin/sh",
		"-c",
		"gcloud beta emulators firestore start --host-port 0.0.0.0:8080 " + fmt.Sprintf("--project=%s", settings.ProjectID),
	}

	container, err := testcontainers.New(ctx, req)
	if err != nil {
		return nil, err
	}

	return newGCloudContainer(ctx, 8080, container, settings)
}
