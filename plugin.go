package qcollector_tcp

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"net"
	"reflect"

	"github.com/qframe/cache-inventory"
	"github.com/qframe/types/plugin"
	"github.com/qframe/types/qchannel"
	"github.com/zpatrick/go-config"
	"github.com/qframe/types/messages"
	"io"
	"bufio"
)

const (
	version = "0.2.6"
	pluginTyp = "collector"
	pluginPkg = "tcp"
)

type Plugin struct {
	*qtypes_plugin.Plugin
	buffer chan interface{}
}



func New(qChan qtypes_qchannel.QChan, cfg *config.Config, name string) (Plugin, error) {
	var err error
	p := Plugin{
		Plugin: qtypes_plugin.NewNamedPlugin(qChan, cfg, pluginTyp, pluginPkg, name, version),
		buffer: make(chan interface{}, 1000),
	}
	return p, err
}

func (p *Plugin) HandleInventoryRequest(qm qtypes_messages.Message) {
	if _, ok := qm.Tags["host"]; !ok {
		p.Log("error", fmt.Sprintf("Got msg '%s' without host key!", qm))
		qm.SourceSuccess = false
		p.QChan.Data.Send(qm)
		return
	}
	tags := []string{""}
	for k, v := range qm.Tags {
		tags = append(tags, fmt.Sprintf("%s=%s", k,v))
	}
	p.Log("trace", fmt.Sprintf("Got msg from %s: %s", strings.Join(tags, ","), qm.Message))
	req := qcache_inventory.NewIPContainerRequest(strings.Join(qm.SourcePath,","), qm.Tags["host"])
	tout := p.CfgIntOr("inventory-timeout-ms", 2000)
	timeout := time.NewTimer(time.Duration(tout)*time.Millisecond).C
	p.QChan.Data.Send(req)
	select {
	case resp := <- req.Back:
		if resp.Error != nil {
			p.Log("error", resp.Error.Error())
			qm.SourceSuccess = false
		} else {
			cm := qtypes_messages.NewContainerMessage(qm.Base, resp.Container, qm.Message)
			cm.AddEngineInfo(resp.Engine)
			p.Log("trace", fmt.Sprintf("Got InventoryResponse: ContainerName:%s | Image:%s", cm.GetContainerName(), cm.Container.Config.Image))
			p.QChan.Data.Send(cm)
			return
		}
	case <- timeout:
		p.Log("debug", fmt.Sprintf("Experience timeout for IP %s... continue w/o Container info (SourcePath: %s)", qm.Tags["host"], strings.Join(qm.SourcePath, ",")))
	}
	p.QChan.Data.Send(qm)
}

func (p *Plugin) Run() {
	p.Log("notice", fmt.Sprintf("Start collector v%s", p.Version))
	host := p.CfgStringOr("bind-host", "0.0.0.0")
	port := p.CfgStringOr("bind-port", "11001")
	// Listen for incoming connections.
	addr :=  host+":"+port
	l, err := net.Listen("tcp",addr)
	defer l.Close()
	if err != nil {
		p.Log("error", fmt.Sprintln("Error listening:", err.Error()))
		return
	}
	// Close the listener when the application closes.

	p.Log("info", fmt.Sprintf("Listening on %s", addr))
	go p.handleRequests(l)
	for {
		select {
		case msg := <- p.buffer:
			switch msg.(type) {
			case IncommingMsg:
				im := msg.(IncommingMsg)
				var b bytes.Buffer
				b.WriteString(im.Msg)
				scanner := bufio.NewScanner(&b)
				for scanner.Scan() {
					base := qtypes_messages.NewTimedBase(p.Name, time.Now())
					qm := qtypes_messages.NewMessage(base, scanner.Text())
					qm.Tags["host"] = im.Host
					go p.HandleInventoryRequest(qm)
				}
			default:
				p.Log("warn", fmt.Sprintf("Unkown data type: %s", reflect.TypeOf(msg)))
			}
		}
	}

}

func (p *Plugin) handleRequests(l net.Listener) {
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			continue
		}
		// Handle connections in a new goroutine.
		go p.handleRequest(conn)
	}
}

type IncommingMsg struct {
	Msg string
	Host string
}

func (p *Plugin) handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1048576)
	// Read the incoming connection into the buffer.
	_, err := conn.Read(buf)
	if err == io.EOF {
		return
	}
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	} else {
		n := bytes.Index(buf, []byte{0})
		addrTuple := strings.Split(conn.RemoteAddr().String(), ":")
		host := addrTuple[0]
		if strings.Count(conn.RemoteAddr().String(), ":") != 1 {
			if conn.RemoteAddr().String()[1:4] == "::1" {
				host = "localhost"
			} else {
				host = conn.RemoteAddr().String()
			}
		}
		im := IncommingMsg{
			// TODO: Is -1 wrong here?
			//Msg: string(buf[:n-1]),
			Msg: string(buf[:n]),
			Host: host,
		}
		p.Log("trace", fmt.Sprintf("Received Raw TCP message '%s' from '%s'", im.Msg, im.Host))
		p.buffer <- im
	}
	// Close the connection when you're done with it.
	conn.Close()
}
