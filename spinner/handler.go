package function

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	guuid "github.com/google/uuid"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

func getAPISecret(secretName string) (secretBytes []byte, err error) {
	// read from the openfaas secrets folder
	secretBytes, err = ioutil.ReadFile("/var/openfaas/secrets/" + secretName)
	if err != nil {
		// read from the original location for backwards compatibility with openfaas <= 0.8.2
		secretBytes, err = ioutil.ReadFile("/run/secrets/" + secretName)
	}

	return secretBytes, err
}

func validAuth(apiKeyHeader string) (bool, error) {

	apiSecret, err := getAPISecret("secret-api-key")
	// fmt.Printf("string(apiSecret): %s\n", string(apiSecret))
	if err != nil {
		fmt.Printf(err.Error())
		return false, err
	}

	if apiKeyHeader != "" && apiKeyHeader == string(apiSecret) {
		fmt.Println("Authorization succeded.")
		return true, nil
	}

	fmt.Println("Authorization failed.")
	return false, nil
}

// Handle will process incoming HTTP requests
func Handle(w http.ResponseWriter, r *http.Request) {

	token := r.Header.Get("X-Api-Key")
	// fmt.Printf("X-Api-Key: %s\n", token)

	authenticated, err := validAuth(token)
	if err != nil || !authenticated {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("Authorization failed. Token is not valid.")))
		return
	}

	query := r.URL.Query()
	serverType := query.Get("server_type")
	fmt.Printf("server type: %s", serverType)

	imageName := query.Get("image_name")
	fmt.Printf("image name: %s", imageName)

	location := query.Get("location")
	fmt.Printf("location: %s", location)

	var message string
	var statusCode int

	client := hcloud.NewClient(hcloud.WithToken(token))

	available := checkAvailableServers(client)
	if !available {
		server, err := spinUpServer(client, serverType, imageName, location)
		if err != nil {
			message = fmt.Sprintf("Failed when spinning up a new server. Reason: %s", err.Error())
			statusCode = http.StatusInternalServerError
		} else {
			message = fmt.Sprintf("Server '%s' created", server.Name)
			statusCode = http.StatusCreated
		}
	} else {
		message = fmt.Sprintf("No need to spin up any new servers.")
		statusCode = http.StatusNoContent
	}

	w.WriteHeader(statusCode)
	w.Write([]byte(fmt.Sprintf(message)))
}

// Checks whether there are any available Servers ready to serve requests or not
func checkAvailableServers(c *hcloud.Client) bool {

	fmt.Println("Checking available servers...")

	servers, err := c.Server.AllWithOpts(context.Background(), hcloud.ServerListOpts{
		Status:   []hcloud.ServerStatus{hcloud.ServerStatusRunning},
		ListOpts: hcloud.ListOpts{LabelSelector: "managed-by=spinner"},
	})
	if err != nil {
		log.Fatalf("error retrieving servers: %s\n", err)
		return false
	}

	for _, server := range servers {
		if server != nil {
			fmt.Printf("Server %q is running.\n", server.Name)
		} else {
			fmt.Println("Something went wrong when iterating over servers.")
		}
	}

	return len(servers) > 0
}

// Spins up a new Server in the cloud provider
// Check out server types: https://docs.hetzner.cloud/#server-types
func spinUpServer(c *hcloud.Client, serverType string, imageName string, location string) (*hcloud.Server, error) {

	fmt.Println("Spinning up a new server...")
	serverName := guuid.New().String()

	if serverType == "" {
		serverType = "cx11" // smallest one
	}

	if imageName == "" {
		imageName = "ubuntu-20.04"
	}

	if location == "" {
		location = "nbg1" // Nuremberg
	}

	start := time.Now()

	result, _, err := c.Server.Create(context.Background(), hcloud.ServerCreateOpts{
		Name:       serverName,
		ServerType: &hcloud.ServerType{Name: serverType},
		Image:      &hcloud.Image{Name: imageName},
		Location:   &hcloud.Location{Name: location},
		Labels:     map[string]string{"managed-by": "spinner"},
	})

	if err != nil {
		fmt.Printf("Server.Create failed: %s\n", err)
		return nil, err
	}

	// Wait until status is running
	isRunning := false

	for !isRunning {

		servers, _, lerr := c.Server.List(context.Background(), hcloud.ServerListOpts{
			Name:     serverName,
			Status:   []hcloud.ServerStatus{hcloud.ServerStatusRunning},
			ListOpts: hcloud.ListOpts{LabelSelector: "managed-by=spinner"},
		})

		if lerr != nil {
			return nil, lerr
		}

		isRunning = len(servers) == 1

		fmt.Println("waiting for server status to be 'running' ...")

		time.Sleep(1 * time.Second)
	}

	t := time.Now()
	elapsed := t.Sub(start)

	fmt.Printf("server state is 'running' after %v milliseconds\n", elapsed.Milliseconds())

	return result.Server, nil
}
