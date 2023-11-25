package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jfreymuth/pulse"
	"github.com/jfreymuth/pulse/proto"
	"github.com/noisetorch/pulseaudio"
)

var moduleId uint32

const moduleName = "listenme"

func initf() {
	mName := os.Getenv("MODULE_NAME")
	if mName == "" {
		mName = moduleName
	}
	c, err := pulseaudio.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	idx, err := c.LoadModule(mName, "sink_name="+mName+"Sink")
	if err != nil {
		log.Fatal(err)
	}
	moduleId = idx
}

func findModuleSink(c *pulse.Client) *pulse.Sink {
	sinks, err := c.ListSinks()
	if err != nil {
		log.Fatal(fmt.Errorf("get sink list:  %w", err))
	}
	for _, s := range sinks {
		if s.Name() == moduleName+"Sink" {
			return s
		}
	}
	return nil
}

func genRandomPasswd(k int) (ret string) {
  rand.Seed(time.Now().Unix())
	for k > 0 {
		ret += strconv.Itoa(rand.Intn(10))
		k--
	}
	return
}

func getIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
    log.Fatal(err)
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil &&
      (
        strings.Contains(ipnet.IP.String(), "125.") ||
        strings.Contains(ipnet.IP.String(), "10.")) {
        return ipnet.IP.String()
			}
		}
	}
  return "localhost"
}

func main() {
	client, err := pulse.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	recordOpts := make([]pulse.RecordOption, 0)
	// s := findModuleSink(client)
	// if s != nil {
	// }
	sampleRate := 96000
	recordOpts = append(recordOpts,
		pulse.RecordSampleRate(sampleRate),
		pulse.RecordStereo,
	)

	var buf bytes.Buffer

	w := pulse.NewWriter(&buf, proto.FormatInt32LE)
	stream, err := client.NewRecord(w, recordOpts...)
	if err != nil {
		log.Fatal(err)
	}

	passwd := os.Getenv("PASSWD")
	if passwd == "" {
		passwd = genRandomPasswd(6)
	}

  ip := getIP()
  log.Println("serve at:", fmt.Sprintf("http://%s:%s?passwd=%s", ip, serverPort, passwd))

	go startServer(sampleRate, &buf, passwd)

	stream.Start()
	log.Println("Press any key to stop...")
	os.Stdin.Read([]byte{0})
	stream.Stop()
}
