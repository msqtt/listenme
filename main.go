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
)


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
  log.Println("serve at:", fmt.Sprintf("http://%s:%s/?passwd=%s", ip, serverPort, passwd))

	go startServer(sampleRate, &buf, passwd)

	stream.Start()
	log.Println("Press any key to stop...")
	os.Stdin.Read([]byte{0})
	stream.Stop()
}
