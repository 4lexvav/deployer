package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// State struct used as reply to GET /deploy/ API.
type State struct {
	Containers []string
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	// @todo: auth request by checking Authorization Header

	switch r.Method {
	case "GET":
		// handle get method: return info about current image version
		cli, err := client.NewEnvClient()
		if err != nil {
			panic(err)
		}

		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containers)
	case "POST":
		// handle post method: deploy new image' version
	}
}

func main() {
	http.HandleFunc("/deploy/", deployHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
