package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	kingpin "github.com/alecthomas/kingpin/v2"
)

func main() {
	cfg := struct {
		amURL      string
		interval   model.Duration
		configPath string
		port       uint
	}{}

	app := kingpin.New(filepath.Base(os.Args[0]), "A deadman's snitch for Prometheus Alertmanager compatible notifications.")
	app.HelpFlag.Short('h')

	app.Flag("am.url", "The URL to POST alerts to.").
		Default("http://localhost:9093/api/v1/alerts").StringVar(&cfg.amURL)
	app.Flag("deadman.interval", "The heartbeat interval. An alert is sent if no heartbeat is sent.").
		Default("30s").SetValue(&cfg.interval)
	app.Flag("config", "Path to config file.").Default("/config.yml").StringVar((&cfg.configPath))
	app.Flag("port", "Listen port").Default("9050").UintVar(&cfg.port)

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)

	_, err := app.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing command line arguments.")
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

	labelConfig, err := NewConfig(cfg.configPath)
	if err != nil {
		log.Fatalf("Cannot load %s: %v\n", cfg.configPath, err)
		os.Exit(1)
	}

	pinger := make(chan time.Time)
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", simpleHandler(pinger))
	http.Handle("/health", http.HandlerFunc(healthHandler))

	logger.Printf("Waiting %s for watchdog alerts. Configured alertmanager for notifications: %s\n", cfg.interval, cfg.amURL)
	logger.Printf("Listening for connections on :%d\n", cfg.port)

	go http.ListenAndServe(fmt.Sprintf(":%d", cfg.port), nil)

	d, err := NewDeadMan(pinger, time.Duration(cfg.interval), cfg.amURL, labelConfig, *logger)
	if err != nil {
		log.Fatalf("err: %v\n", err)
		os.Exit(2)
	}

	d.Run()
}

func healthHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Alive")
}

func simpleHandler(pinger chan<- time.Time) http.HandlerFunc {

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	logger.Print("Waiting for alerts")
	return func(w http.ResponseWriter, r *http.Request) {
		pinger <- time.Now()
		body, _ := ioutil.ReadAll(r.Body)
		logger.Printf("Got Watchdog alert: %s", body)
		fmt.Fprint(w, "")
	}
}
