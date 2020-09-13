package main

import (
	"context"
	"fmt"
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/johnsto/speedtest"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {

	down, up := testSpeed()
	sendToInflux(down, up)

}

func sendToInflux(down, up int) {
	userName := "kii"
	password := "influx4cloudupload"

	client := influxdb2.NewClient("https://influxdb.cloud.mklug.at:443", fmt.Sprintf("%s:%s", userName, password))

	writeAPI := client.WriteAPIBlocking("", "speedtest")

	measurement := influxdb2.NewPoint("home",
		map[string]string{
			"location": "home",
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
}

func testSpeed() (int, int) {

	const timeout = 1 * time.Second

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
