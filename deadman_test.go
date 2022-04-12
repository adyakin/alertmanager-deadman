package main

import (
	"testing"
	"time"

	"github.com/prometheus/common/promlog"
)

func TestDeadManDoesntTrigger(t *testing.T) {
	pinger := time.NewTicker(10 * time.Millisecond)
	defer pinger.Stop()

	called := false
	logger := promlog.New(promlog.AllowedLevel{})
	d := newDeadMan(pinger.C, 20*time.Millisecond, func() error {
		called = true
		return nil
	}, logger)

	go d.Run()
	defer d.Stop()

	time.Sleep(100 * time.Millisecond)
	if called == true {
		t.Fatal("deadman triggered!")
	}
}

func TestDeadManTriggers(t *testing.T) {
	pinger := time.NewTicker(30 * time.Millisecond)
	defer pinger.Stop()

	called := false
	logger := promlog.New(promlog.AllowedLevel{})
	d := newDeadMan(pinger.C, 20*time.Millisecond, func() error {
		called = true
		return nil
	}, logger)

	go d.Run()
	defer d.Stop()

	time.Sleep(100 * time.Millisecond)
	if called == false {
		t.Fatal("deadman did not trigger!")
	}
}
