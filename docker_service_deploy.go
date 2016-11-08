package main

import (
	"errors"

	"github.com/docker/docker/api/types/swarm"
	"github.com/fsouza/go-dockerclient"
)

type DockerServiceDeploy struct {
	client *docker.Client
}

func (d DockerServiceDeploy) Deploy(payload *Payload) error {

	s, err := d.client.InspectService(payload.ServiceName)
	if err != nil {
		return err
	}

	if !d.isUpdatable(s) {
		return errors.New("Service is not labeled as being updatable: set deployer.allowUpdates=true as service label")
	}

	// Update needs to include the current full spec, as well as the current version number
	opts := docker.UpdateServiceOptions{
		ServiceSpec: s.Spec,
		Version:     s.Version.Index,
	}

	opts.ServiceSpec.TaskTemplate.ContainerSpec.Image = payload.Artifact

	// Update by service name doesn't work.
	if err := d.client.UpdateService(s.ID, opts); err != nil {
		return err
	}

	return nil
}

func (d DockerServiceDeploy) isUpdatable(s *swarm.Service) bool {
	return s.Spec.Labels["deployer.allowUpdates"] == "true"
}
