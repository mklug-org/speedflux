package main

import (
	"context"
	"fmt"
	"github.com/influxdata/influxdb-client-go"
	"github.com/johnsto/speedtest"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {

	location := getEnvVariable("SPEEDFLUX_LOCATION")
	userName := getEnvVariable("SPEEDFLUX_USER")
	password := getEnvVariable("SPEEDFLUX_PASS")
	protocol := getEnvVariable("SPEEDFLUX_PROTOCOL")
	host := getEnvVariable("SPEEDFLUX_HOST")
	port := getEnvVariable("SPEEDFLUX_PORT")
	db := getEnvVariable("SPEEDFLUX_DB")
	interval, err := strconv.Atoi(getEnvVariable("SPEEDFLUX_INTERVAL"))
	if err != nil {
		log.Fatalf("Interval: %d could not successfully be parsed: %v", interval, err)
	}

	log.Printf("starting measurements for location %s every %d minute(s)", location, interval)

	for true {
		go measure(location, protocol, host, port, userName, password, db)
		log.Printf("starting measurement and next iteration in %d minutes", interval)
		time.Sleep(time.Duration(interval) * time.Minute)
	}
}

func measure(location string, protocol string, host string, port string, userName string, password string, db string) {
	down, up := testSpeed()
	sendToInflux(down, up, location, fmt.Sprintf("%s://%s:%s", protocol, host, port), db, fmt.Sprintf("%s:%s", userName, password))
}

func sendToInflux(down, up int, location, serverUrl, db, authToken string) {
	log.Printf("sending data to influxdb server %s", serverUrl)

	client := influxdb2.NewClient(serverUrl, authToken)

	writeAPI := client.WriteAPIBlocking("", db)

	measurement := influxdb2.NewPoint("speed",
		map[string]string{
			"location": location,
		},
		map[string]interface{}{
			"down": down,
			"up":   up,
		},
		time.Now())

	err := writeAPI.WritePoint(context.Background(), measurement)
	if err != nil {
		log.Printf("Write error: %s\n", err.Error())
	}
	log.Print("Data sent to influxdb successfully")
}

func testSpeed() (int, int) {

	const timeout = 10 * time.Second

	log.Print("starting speedtest exporter")

	log.Print("fetching server list")
	settings, err := speedtest.FetchSettings()
	if err != nil {
		log.Printf("error: %v", err)
		return 0, 0
	}

	log.Printf("%v found.\n", len(settings.Servers))

	log.Printf("Fetching config...\n")
	config, err := speedtest.FetchConfig()
	if err != nil {
		log.Printf("Couldn't read config: %v", err)
		return 0, 0
	}
	settings.UpdateDistances(config.Client.Lat, config.Client.Lon)

	log.Printf("  ISP: %v\n", config.Client.IspName)
	log.Printf("  Location: %v, %v\n\n", config.Client.Lat, config.Client.Lon)

	var server speedtest.Server
	settings.Servers.SortByDistance()
	server = settings.Servers[0]

	log.Printf("Using server %d. %v, %v, %v (%dkm)\n",
		server.ID, server.Sponsor, server.Name, server.Country, int(server.Distance))

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, timeout)
			},
		},
	}

	benchmarkDown := speedtest.NewDownloadBenchmark(client, server)
	log.Print("Testing download speed... ")
	rateDown := speedtest.RunBenchmark(benchmarkDown, 4, 16, timeout)
	log.Println(speedtest.NiceRate(rateDown))

	benchmarkUp := speedtest.NewUploadBenchmark(client, server)
	log.Print("Testing upload speed... ")
	rateUp := speedtest.RunBenchmark(benchmarkUp, 4, 16, timeout)
	log.Println(speedtest.NiceRate(rateUp))

	return rateDown, rateUp
}

func getEnvVariable(v string) string {
	value, found := os.LookupEnv(v)
	if !found {
		log.Fatalf("ENV variable %s is not set. Quitting", v)
	}
	return value
}
