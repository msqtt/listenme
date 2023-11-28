package main

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type onlineSet struct {
	set  map[*websocket.Conn]struct{}
	lock sync.Mutex
}

var userSet onlineSet
var serverPort = "4321"
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
} // use default options

func init() {
	port := os.Getenv("PORT")
	if port != "" {
		serverPort = port
	}
	userSet = onlineSet{set: make(map[*websocket.Conn]struct{})}
}

func dealWithMessage(host string, conn *websocket.Conn, ackCh, closeCh chan struct{}) {
	for {
	}
}

func readAudioChunk(sampleRate int, reader io.Reader) []byte {
	size := sampleRate << 1
	bytes := make([]byte, size)
	idx := 0
	for idx < size {
		n, err := reader.Read(bytes[idx:])
		idx += n
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
	}
	return bytes[:idx]
}

func serverStream(sampleRate int, reader io.Reader, passwd string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("passwd") != passwd {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
		}
		defer conn.Close()

		userSet.lock.Lock()
		userSet.set[conn] = struct{}{}
		userSet.lock.Unlock()

		// closeCh := make(chan struct{})

		agent := strings.Join(r.Header["User-Agent"], " ")
		log.Println(agent, "entering...")

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("cannot read from client message: ", string(msg))
				log.Printf("connection from %s closing...\n", agent)
				conn.Close()
				return
			}
			switch string(msg) {
			case "Close!":
				userSet.lock.Lock()
				delete(userSet.set, conn)
				userSet.lock.Unlock()

				conn.Close()
				log.Printf("connection from %s closing...\n", agent)
				return
			default:
				conn.WriteMessage(websocket.TextMessage, msg)
			}
		}
	}
}

func audioServer(sampleRate int, audio io.Reader) {
	for {
		time.Sleep(150 * time.Millisecond)
		userSet.lock.Lock()
		b := readAudioChunk(sampleRate, audio)
		for c := range userSet.set {
			err := c.WriteMessage(websocket.BinaryMessage, b)
			if err != nil {
				c.Close()
				delete(userSet.set, c)
				log.Println("cannot send audio message, closing...")
			}
		}
		userSet.lock.Unlock()
	}
}

//go:embed web
var web embed.FS

func startServer(sampleRate int, audio io.Reader, passwd string) {
	go audioServer(sampleRate, audio)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFS(web, "web/index.html")
		// ParseFiles("web/index.html")
		if err != nil {
			log.Fatal(err)
		}
		pswd := r.URL.Query().Get("passwd")
		t.Execute(w, map[string]any{"auth": pswd == passwd, "sampleRate": sampleRate})
	})
	subWeb, _ := fs.Sub(web, "web")
	http.Handle("/js/", http.FileServer(http.FS(subWeb)))
	http.HandleFunc("/listen", serverStream(sampleRate, audio, passwd))
	log.Fatal(http.ListenAndServe(":"+serverPort, nil))
}
