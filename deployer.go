package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// State struct used as reply to GET /deploy/ API.
type State struct {
	Containers []string
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	// @TODO: auth request by checking Authorization Header

	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	switch r.Method {
	case "GET": // handle get method: return info about current image version
		containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containers)
	case "POST": // handle post method: deploy new image' version
		name := r.FormValue("name")
		imageName := r.FormValue("image")
		token := r.FormValue("token")
		user := r.FormValue("user")
		net := r.FormValue("network")

		authJSON, err := json.Marshal(types.AuthConfig{Username: user, Password: token})
		if err != nil {
			panic(err)
		}

		// Pull new image version
		reader, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{RegistryAuth: base64.URLEncoding.EncodeToString(authJSON)})
		if err != nil {
			panic(err)
		}

		io.Copy(os.Stdout, reader)
		defer reader.Close()

		args := filters.NewArgs()
		args.Add("name", name)

		containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: args, All: true})
		if err != nil {
			panic(err)
		}

		if len(containers) > 0 {
			oldContainer := containers[0]

			if err = cli.ContainerStop(ctx, oldContainer.ID, nil); err != nil {
				panic(err)
			}

			// Remove old container
			if err = cli.ContainerRemove(ctx, oldContainer.ID, types.ContainerRemoveOptions{}); err != nil {
				panic(err)
			}
		}

		// Run new container
		newContainer, err := cli.ContainerCreate(ctx, &container.Config{
			Tty:        true,
			Image:      imageName,
			WorkingDir: "/var/www",
			Env:        []string{"SERVICE_NAME=app", "SERVICE_TAGS=dev"},
			Cmd:        []string{"/usr/bin/supervisord", "-n", "-c", "/etc/supervisord.conf"},
			ExposedPorts: nat.PortSet{
				"9000/tcp": struct{}{},
			},
		}, &container.HostConfig{
			RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		}, &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				net: {},
			},
		}, name)

		if err != nil {
			panic(err)
		}

		cli.ContainerStart(ctx, newContainer.ID, types.ContainerStartOptions{})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newContainer)
	}
}

func main() {
	http.HandleFunc("/deploy/", deployHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
