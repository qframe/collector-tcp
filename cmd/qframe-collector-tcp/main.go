package main

import (
	"log"
	"fmt"
	"time"
	"github.com/zpatrick/go-config"
	"github.com/qnib/qframe-types"
	"github.com/qframe/collector-tcp"
	"github.com/qnib/qframe-collector-docker-events/lib"
	"github.com/qnib/qframe-filter-inventory/lib"
	"github.com/docker/docker/api/types"
)

const (
	dockerHost = "unix:///var/run/docker.sock"
	dockerAPI = "v1.29"
)

func Run(qChan qtypes.QChan, cfg config.Config, name string) {
	p, _ := qframe_collector_tcp.New(qChan, cfg, name)
	p.Run()
}


func main() {
	qChan := qtypes.NewQChan()
	qChan.Broadcast()
	cfgMap := map[string]string{
		"log.level": "info",
		"collector.tcp.port": "10001",
		"collector.tcp.docker-host": "unix:///var/run/docker.sock",
		"filter.inventory.inputs": "docker-events",
		"filter.inventory.ticker-ms": "2500",	}

	cfg := config.NewConfig(
		[]config.Provider{
			config.NewStatic(cfgMap),
		},
	)
	// Start docker-events
	pde, err := qframe_collector_docker_events.New(qChan, *cfg, "docker-events")
	if err != nil {
		log.Printf("[EE] Failed to create collector: %v", err)
		return
	}
	go pde.Run()
	pfi := qframe_filter_inventory.New(qChan, *cfg, "inventory")
	if err != nil {
		log.Printf("[EE] Failed to create filter: %v", err)
		return
	}
	go pfi.Run()
	time.Sleep(2*time.Second)
	p, err := qcollector_tcp.New(qChan, *cfg, "tcp")
	if err != nil {
		log.Printf("[EE] Failed to create collector: %v", err)
		return
	}
	go p.Run()
	time.Sleep(2*time.Second)
	bg := qChan.Data.Join()
	done := false
	for {
		select {
		case val := <- bg.Read:
			switch val.(type) {
			case qtypes.QMsg:
				qm := val.(qtypes.QMsg)
				if qm.Source == "tcp" {
					switch qm.Data.(type) {
					case types.ContainerJSON:
						cnt := qm.Data.(types.ContainerJSON)
						p.Log("info", fmt.Sprintf("Got inventory response for msg: '%s'", qm.Msg))
						p.Log("info", fmt.Sprintf("        Container{Name:%s, Image: %s}", cnt.Name, cnt.Image))
						done = true

					}
				}
			}
		}
		if done {
			break
		}
	}
}
