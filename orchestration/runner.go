package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	docker "github.com/bywan/go-dockercommand"
)

type Runner struct {
	BuildImages []BuiltImage
	Env         map[string]string
}

func (r *Runner) Run() (bool, error) {
	client, err := docker.NewDocker(DockerEndpoint)
	if err != nil {
		return false, err
	}

	errChanRun := make(chan error)
	successChanRun := make(chan bool)
	remainingRuns := len(r.BuildImages)
	for _, buildImage := range r.BuildImages {
		go runContainer(client, buildImage, r.Env, successChanRun, errChanRun)
	}

	success := false
	for {
		select {
		case result := <-successChanRun:
			success = success || result
			remainingRuns--
		case err := <-errChanRun:
			return false, err
		}

		if remainingRuns == 0 {
			break
		}
	}

	log.Printf("Dockerfiles builds finished\n")
	return success, nil
}

func runContainer(client *docker.Docker, buildImage BuiltImage, env map[string]string, successChan chan bool, errChan chan error) {
	success := true
	servicesFile := fmt.Sprintf("%s/work/%d/services", BazookaInput, buildImage.VariantID)

	servicesList, err := listServices(servicesFile)
	if err != nil {
		errChan <- err
		return
	}

	serviceContainers := []*docker.Container{}
	containerLinks := []string{}
	for _, service := range servicesList {
		name := fmt.Sprintf("service-%s-%s-%d", env[BazookaEnvProjectID], env[BazookaEnvJobID], buildImage.VariantID)
		containerLinks = append(containerLinks, fmt.Sprintf("%s:%s", name, service))
		serviceContainer, err := client.Run(&docker.RunOptions{
			Name:   name,
			Image:  service,
			Detach: true,
		})
		if err != nil {
			errChan <- err
			return
		}
		serviceContainers = append(serviceContainers, serviceContainer)
	}

	// TODO link containers
	container, err := client.Run(&docker.RunOptions{
		Image:  buildImage.Image,
		Links:  containerLinks,
		Detach: true,
	})
	if err != nil {
		errChan <- err
		return
	}

	container.Logs(buildImage.Image)

	exitCode, err := container.Wait()
	if err != nil {
		errChan <- err
		return
	}
	if exitCode != 0 {
		if exitCode == 42 {
			errChan <- fmt.Errorf("Run failed\n Check Docker container logs, id is %s\n", container.ID())
			return
		}
		success = false
	}
	err = container.Remove(&docker.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
	if err != nil {
		errChan <- err
		return
	}

	for _, serviceContainer := range serviceContainers {
		err = serviceContainer.Remove(&docker.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
		if err != nil {
			errChan <- err
			return
		}
	}
	successChan <- success
}

func listServices(servicesFile string) ([]string, error) {
	file, err := os.Open(servicesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var services []string
	for scanner.Scan() {
		services = append(services, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return services, nil
}
