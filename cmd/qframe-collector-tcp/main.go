package main

import (
	"log"
	"fmt"
	"time"
	"github.com/zpatrick/go-config"
	"github.com/qnib/qframe-types"
	"github.com/qframe/collector-tcp"
	"github.com/docker/docker/api/types"
	"github.com/qframe/cache-inventory"
	"github.com/qframe/collector-docker-events"
	"github.com/qframe/types/qchannel"
)

const (
	dockerHost = "unix:///var/run/docker.sock"
	dockerAPI = "v1.30"
)


func main() {
	qChan := qtypes_qchannel.NewQChan()
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
	pde, err := qcollector_docker_events.New(qChan, *cfg, "docker-events")
	if err != nil {
		log.Printf("[EE] Failed to create collector: %v", err)
		return
	}
	go pde.Run()
	pci, err := qcache_inventory.New(qChan, *cfg, "inventory")
	if err != nil {
		log.Printf("[EE] Failed to create filter: %v", err)
		return
	}
	go pci.Run()
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
