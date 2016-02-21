package ws

import "golang.org/x/net/websocket"
import "errors"

const IdMask = 0x0f

var IdNum = 0
var room0 = &Room{Clients: make(map[int]*Conn), SharedDataQue: make(chan Data, 10)}
var room1 = &Room{Clients: make(map[int]*Conn), SharedDataQue: make(chan Data, 10)}
var room []*Room

func init() {
	go room1.initChan()
	go room0.initChan()
	room = []*Room{room0, room1}
}

type Conn struct {
	Conn *websocket.Conn
	Id   int
}

func (c *Conn) Write(d Data) error {
	switch d.Type {
	case 1:
		c.Conn.Write(d.Conntent)
		return nil
	}
	return errors.New("not found")
}

func NewConn(c *websocket.Conn, rid int) *Conn {
	IdNum++ //parallel bug
	con := &Conn{Id: ((rid << 8) + IdNum), Conn: c}
	room[rid].Add(con)
	return con
}
func (c Conn) GetClientId() int {
	return c.Id & IdMask
}
func (c Conn) GetRoomId() int {
	return c.Id >> 8
}

type Room struct {
	Clients       map[int]*Conn
	SharedDataQue chan Data
}

func (r *Room) Add(c *Conn) {
	r.Clients[c.Id] = c
}
func (r *Room) Del(c *Conn) {
	delete(r.Clients, c.Id)
}
func (r *Room) Receive(d Data) {
	r.SharedDataQue <- d
}
func (r *Room) initChan() {
	for v := range r.SharedDataQue {
		for i, d := range r.Clients {
			println(i, v.From)
			if i == v.From {
				continue
			}
			d.Write(v)
		}
	}
}

type Data struct {
	Type     int
	Conntent []byte
	From     int
	Time     int64
}

var WsHandler websocket.Handler = func(con *websocket.Conn) {
	switch con.Request().URL.Path {
	case "/":
		c := NewConn(con, 0)
		for {
			msg := make([]byte, 1024)
			n, err := con.Read(msg)
			if err != nil {
				println(err.Error())
				return
			}
			room0.Receive(Data{Type: 1, Conntent: msg[:n], From: c.Id})
		}
	case "/mv":
		c := NewConn(con, 1)
		for {
			msg := make([]byte, 1024)
			n, err := con.Read(msg)
			if err != nil {
				println(err.Error())
				return
			}
			room1.Receive(Data{Type: 1, Conntent: msg[:n], From: c.Id})
		}
	}
}
