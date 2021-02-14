package app

import (
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mercedtime/api/catalog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type hub struct {
	clients map[net.Addr]*websocket.Conn
	mu      sync.Mutex
}

func (h *hub) get(key net.Addr) *websocket.Conn {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.clients[key]
}

func (a *App) wsPublisher(ch chan<- interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		r := make([]*catalog.Entry, 0)
		if err := c.BindJSON(&r); err != nil {
			senderr(c, err, 400)
			log.Println(err)
			return
		}
		ch <- r
		c.Status(200)
	}
}

func (a *App) wsSub(ch <-chan interface{}) gin.HandlerFunc {
	var (
		// TODO Using the client address as the key limits the number of sockets
		// to one per user. This makes it so that only one tab will have accesss
		// to real time data.
		clients = make(map[net.Addr]*websocket.Conn)
		mu      sync.Mutex
	)

	// TODO Each client only cares about one semester at a time and will not want data from
	// a bulk update from a different semester.
	go func() {
		for {
			up := <-ch
			mu.Lock()
			for _, c := range clients {
				c.WriteJSON(up)
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(500, Error{Msg: err.Error(), Status: 500})
			return
		}
		remote := conn.RemoteAddr()

		mu.Lock()
		clients[remote] = conn
		mu.Unlock()
		conn.SetCloseHandler(func(code int, text string) error {
			println("closing")
			log.Printf("closing websocket connection to %v\n", remote)
			mu.Lock()
			delete(clients, remote)
			mu.Unlock()
			return conn.Close()
		})
		for {
			_, _, err = conn.NextReader()
			if err != nil {
				log.Println(err)
				conn.Close()
				break
			}
		}
	}
}
