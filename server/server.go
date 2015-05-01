package server

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/khlieng/name_pending/Godeps/_workspace/src/github.com/gorilla/websocket"
	"github.com/khlieng/name_pending/Godeps/_workspace/src/github.com/julienschmidt/httprouter"

	"github.com/khlieng/name_pending/storage"
)

var (
	channelStore *storage.ChannelStore
	sessions     map[string]*Session
	sessionLock  sync.Mutex
	fs           http.Handler
	files        []File

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type File struct {
	Path        string
	ContentType string
}

func Run(port int, development bool) {
	defer storage.Cleanup()

	channelStore = storage.NewChannelStore()
	sessions = make(map[string]*Session)
	fs = http.FileServer(BindataFileSystem{})

	files = []File{
		File{"/bundle.js", "text/javascript"},
		File{"/css/style.css", "text/css"},
		File{"/css/fontello.css", "text/css"},
		File{"/font/fontello.eot", "application/vnd.ms-fontobject"},
		File{"/font/fontello.svg", "image/svg+xml"},
		File{"/font/fontello.ttf", "application/x-font-ttf"},
		File{"/font/fontello.woff", "application/font-woff"},
	}

	if !development {
		reconnect()
	}

	router := httprouter.New()

	router.HandlerFunc("GET", "/ws", upgradeWS)
	router.NotFound = serveFiles

	log.Println("Listening on port", port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), router))
}

func upgradeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	handleWS(conn)
}

func serveFiles(w http.ResponseWriter, r *http.Request) {
	var ext string

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		ext = ".gz"
	}

	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		r.URL.Path = "/index.html" + ext
		fs.ServeHTTP(w, r)
		return
	}

	for _, file := range files {
		if strings.HasSuffix(r.URL.Path, file.Path) {
			w.Header().Set("Content-Type", file.ContentType)
			r.URL.Path = file.Path + ext
			fs.ServeHTTP(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	r.URL.Path = "/index.html" + ext

	fs.ServeHTTP(w, r)
}

func reconnect() {
	for _, user := range storage.LoadUsers() {
		session := NewSession()
		session.user = user
		sessions[user.UUID] = session
		go session.write()

		channels := user.GetChannels()

		for _, server := range user.GetServers() {
			irc := NewIRC(server.Nick, server.Username)
			irc.TLS = server.TLS
			irc.Password = server.Password
			irc.Realname = server.Realname

			go func() {
				err := irc.Connect(server.Address)
				if err != nil {
					log.Println(err)
				} else {
					session.setIRC(irc.Host, irc)

					go handleMessages(irc, session)

					var joining []string
					for _, channel := range channels {
						if channel.Server == server.Address {
							joining = append(joining, channel.Name)
						}
					}
					irc.Join(joining...)
				}
			}()
		}
	}
}