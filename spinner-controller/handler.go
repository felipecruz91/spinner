package function

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

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
	metricName := query.Get("metric_name")
	fmt.Printf("metricName: %s", metricName)

	metricThreshold := query.Get("metric_threshold")
	fmt.Printf("metricThreshold: %s", metricThreshold)
	mt, _ := strconv.Atoi(metricThreshold)

	lastMinutes := query.Get("last_minutes")
	fmt.Printf("lastMinutes: %s", lastMinutes)
	lm, _ := strconv.Atoi(lastMinutes)

	var message string
	var statusCode int

	client := hcloud.NewClient(hcloud.WithToken(token))

	fmt.Println("Checking running servers...")

	servers, err := client.Server.AllWithOpts(context.Background(), hcloud.ServerListOpts{
		Status:   []hcloud.ServerStatus{hcloud.ServerStatusRunning},
		ListOpts: hcloud.ListOpts{LabelSelector: "managed-by=spinner"},
	})
	if err != nil {
		message = fmt.Sprintf("error retrieving servers. Reason: %s", err.Error())
		statusCode = http.StatusInternalServerError
		w.WriteHeader(statusCode)
		w.Write([]byte(fmt.Sprintf(message)))
		return
	}

	if len(servers) == 0 {
		w.WriteHeader(statusCode)
		w.Write([]byte(fmt.Sprintf("No servers in running state.")))
		return
	}

	for _, server := range servers {
		metricIsBelow, err := metricIsBelow(client, server, metricName, mt, lm)
		if err != nil {
			message = fmt.Sprintf("unable to check metric. Reason: %s", err.Error())
			statusCode = http.StatusInternalServerError
			w.WriteHeader(statusCode)
			w.Write([]byte(fmt.Sprintf(message)))
			return
		}

		if metricIsBelow {
			_, err := client.Server.Delete(context.Background(), server)
			if err != nil {
				message = fmt.Sprintf("failed to delete server. Reason: %s", err.Error())
				statusCode = http.StatusInternalServerError
				w.WriteHeader(statusCode)
				w.Write([]byte(fmt.Sprintf(message)))
				return
			}
		}
	}

	w.WriteHeader(statusCode)
	w.Write([]byte(fmt.Sprintf("Done")))
}

// Check if a given metric is below a given threshold
func metricIsBelow(c *hcloud.Client, server *hcloud.Server, metricName string, metricThreshold int, lastMinutes int) (bool, error) {

	fmt.Printf("Checking metrics for server %s\n", server.Name)

	now := time.Now()

	serverMetrics, _, err := c.Server.GetMetrics(context.Background(), server, hcloud.ServerGetMetricsOpts{
		Types: []hcloud.ServerMetricType{hcloud.ServerMetricCPU},
		Start: now.Add(time.Duration(-lastMinutes) * time.Minute),
		End:   now,
	})
	if err != nil {
		log.Fatalf("error retrieving metrics for server: %s\n", err)
		return false, err
	}

	metricKeyPairs := serverMetrics.TimeSeries[metricName]

	var avg int
	sum := 0
	for _, metricKeyPair := range metricKeyPairs {
		fmt.Printf("Timestamp: %v, Value: %s\n", metricKeyPair.Timestamp, metricKeyPair.Value)
		val, err2 := strconv.Atoi(metricKeyPair.Value)
		if err2 != nil {
			return false, err2
		}
		sum += val
	}

	return avg < sum/len(metricKeyPairs), nil
}
