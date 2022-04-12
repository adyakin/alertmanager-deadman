package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

var (
	ticksTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "deadman_ticks_total",
			Help: "The total ticks passed in this snitch",
		},
	)

	ticksNotified = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "deadman_ticks_notified",
			Help: "The number of ticks where notifications were sent.",
		},
	)

	failedNotifications = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "deadman_notifications_failed",
			Help: "The number of failed notifications.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		ticksTotal,
		ticksNotified,
		failedNotifications,
	)
}

func NewDeadMan(pinger <-chan time.Time, interval time.Duration, amURL string, labelConfig *Config, logger log.Logger) (*Deadman, error) {
	return newDeadMan(pinger, interval, amNotifier(amURL, labelConfig, logger), logger), nil
}

type Deadman struct {
	pinger   <-chan time.Time
	interval time.Duration
	ticker   *time.Ticker
	closer   chan struct{}

	notifier func() error

	logger log.Logger
}

func newDeadMan(pinger <-chan time.Time, interval time.Duration, notifier func() error, logger log.Logger) *Deadman {
	return &Deadman{
		pinger:   pinger,
		interval: interval,
		notifier: notifier,
		closer:   make(chan struct{}),
		logger:   logger,
	}
}

func (d *Deadman) Run() error {
	d.ticker = time.NewTicker(d.interval)

	skip := false

	for {
		select {
		case <-d.ticker.C:
			ticksTotal.Inc()

			if !skip {
				ticksNotified.Inc()
				if err := d.notifier(); err != nil {
					failedNotifications.Inc()
					d.logger.Printf("err: %v\n", err)
				}
			}
			skip = false

		case <-d.pinger:
			skip = true

		case <-d.closer:

		}
	}
}

func (d *Deadman) Stop() {
	if d.ticker != nil {
		d.ticker.Stop()
	}

	d.closer <- struct{}{}
}

func amNotifier(amURL string, cfg *Config, logger log.Logger) func() error {

	labels := model.LabelSet{}
	for k, v := range cfg.Labels {
		labels[model.LabelName(k)] = model.LabelValue(v)
	}

	annotations := model.LabelSet{}
	for k, v := range cfg.Annotations {
		annotations[model.LabelName(k)] = model.LabelValue(v)
	}
	alerts := []*model.Alert{{
		Labels:      labels,
		Annotations: annotations,
	}}
	logger.Printf("Using alerts labels: %v\n", alerts[len(alerts)-1].Labels)
	logger.Printf("Using annotations: %v\n", alerts[len(alerts)-1].Annotations)

	b, err := json.Marshal(alerts)
	if err != nil {
		logger.Printf("Failed to mashal alert: %v\n", err)
		os.Exit(2)
	}

	return func() error {
		logger.Printf("Sending notification to %s\n", amURL)
		client := &http.Client{}
		resp, err := client.Post(amURL, "application/json", bytes.NewReader(b))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("bad response status %v", resp.Status)
		}

		return nil
	}
}
