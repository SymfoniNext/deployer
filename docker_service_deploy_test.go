package main

import (
	"testing"

	"github.com/fsouza/go-dockerclient"
)

func TestDockerServiceDeploy(t *testing.T) {
	t.Skip()

	endpoint := "unix:///var/run/docker.sock"

	c, _ := docker.NewClient(endpoint)

	deployer := DockerServiceDeploy{
		client: c,
	}

	if err := deployer.Deploy(&Payload{"redis", "redis:3.2-alpine"}); err != nil {
		t.Errorf("%v", err)
	}
}
