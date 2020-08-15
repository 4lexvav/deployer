package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

const TOKEN_ENV = "DEPLOYER_TOKEN"

type DeployConfig struct {
	User    string
	Token   string
	Name    string
	Image   string
	Network string
	Env     map[string]string
	Volumes []VolumeConfig
	Tty     bool
}

type VolumeConfig struct {
	Source string
	Target string
}

func authMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		token := os.Getenv(TOKEN_ENV)

		if token == "" || auth != token {
			http.Error(w, "Unauthorized request", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	switch r.Method {
	case http.MethodGet: // handle get method: return info about running images
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containerList(ctx, cli, types.ContainerListOptions{}))

	case http.MethodPost: // handle post method: deploy new image version
		var config DeployConfig

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&config); err != nil {
			panic(err)
		}

		pullImage(ctx, cli, config.User, config.Token, config.Image)
		stopContainer(ctx, cli, config.Name)
		newContainer := createContainer(ctx, cli, config)
		cli.ContainerStart(ctx, newContainer.ID, types.ContainerStartOptions{})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newContainer)
	}
}

func containerList(ctx context.Context, cli *client.Client, options types.ContainerListOptions) []types.Container {
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		panic(err)
	}

	return containers
}

func pullImage(ctx context.Context, cli *client.Client, user string, token string, image string) {
	authJSON, err := json.Marshal(types.AuthConfig{Username: user, Password: token})
	if err != nil {
		panic(err)
	}

	reader, err := cli.ImagePull(ctx, image, types.ImagePullOptions{RegistryAuth: base64.URLEncoding.EncodeToString(authJSON)})
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, reader)
	defer reader.Close()
}

func stopContainer(ctx context.Context, cli *client.Client, name string) {
	args := filters.NewArgs()
	args.Add("name", name)
	containers := containerList(ctx, cli, types.ContainerListOptions{Filters: args, All: true})
	if len(containers) > 0 {
		oldContainer := containers[0]

		if err := cli.ContainerStop(ctx, oldContainer.ID, nil); err != nil {
			panic(err)
		}

		// Remove old container
		if err := cli.ContainerRemove(ctx, oldContainer.ID, types.ContainerRemoveOptions{}); err != nil {
			panic(err)
		}
	}
}

func createContainer(ctx context.Context, cli *client.Client, config DeployConfig) (createdContainer container.ContainerCreateCreatedBody) {
	env := []string{}
	for key, val := range config.Env {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}

	mounts := []mount.Mount{}
	for _, volume := range config.Volumes {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeBind,
			Source: volume.Source,
			Target: volume.Target,
		})
	}

	createdContainer, err := cli.ContainerCreate(ctx, &container.Config{
		Tty:   config.Tty,
		Image: config.Image,
		Env:   env,
	}, &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		NetworkMode:   container.NetworkMode(config.Network),
		Mounts:        mounts,
	}, nil, config.Name)

	if err != nil {
		panic(err)
	}

	return
}

func main() {
	token := os.Getenv(TOKEN_ENV)
	if token == "" {
		log.Fatalf("Define %s env variable before running script", TOKEN_ENV)
	}

	mux := http.NewServeMux()
	mux.Handle("/deploy/", authMiddleware(
		http.HandlerFunc(deployHandler),
	))

	log.Fatal(http.ListenAndServe(":3000", mux))
}
