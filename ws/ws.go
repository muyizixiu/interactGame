package ws

import (
	"encoding/json"
	"fmt"
	"groups/log"
	"strconv"
	"sync"

	"golang.org/x/net/websocket"
)
import "errors"

const IdMask = 0x0f

var RoomNotExist = errors.New("room not exist")
var IdNum = 0
var room map[int]*Room

type Key struct {
	RoomId int
	Name   string
}

func init() {
	room = make(map[int]*Room, 10)
	room0 := initARoom("chess")
	room1 := initARoom("chess")
	room[room0.Id] = room0
	room[room1.Id] = room1
}

type Conn struct {
	Conn       *websocket.Conn
	Id         int
	Mask       int
	Sid        int
	Name       string
	ClientType int
	Room       *Room
}

func (c *Conn) Write(d Data) error {
	fmt.Println(d, string(d.Conntent))
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
	con.ClientType = rid
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
	Id            int
	ClientsLeft   []*Conn
	Clients_r     map[int]map[int]*Conn
	GameName      string //房间所属游戏的名字
}

func (r Room) IsOverLimit(t int) bool {
	if m, ok := r.Clients_r[t]; ok {
		return IsOverLimit(r.GameName, t, len(m))
	}
	log.Log("no such limit at " + r.GameName + " at " + strconv.Itoa(t))
	print("room")
	return true
}

//房间配置信息,决定房间有多少人，房间基础信息
type RoomConfig struct {
	GameName string
	Limit    map[int]int
}

func (r RoomConfig) IsOverLimit(t int, total int) bool {
	if limit, ok := r.Limit[t]; ok {
		fmt.Print("total limit", total, limit)
		return total >= limit
	} //！ok 应该做错误日志
	log.Log("fatal: no such game limit: " + r.GameName + " at " + strconv.Itoa(t)) //这个日志意味着有程序内部错误
	return true
}

var roomConfig map[string]RoomConfig

func IsOverLimit(name string, t, total int) bool {
	if r, ok := roomConfig[name]; ok {
		return r.IsOverLimit(t, total)
	}
	log.Log("no such game :" + name)
	return true
}

func init() {
	roomConfig = make(map[string]RoomConfig, 4)
	roomConfig["chess"] = RoomConfig{GameName: "chess", Limit: map[int]int{0: 2, 1: 10}} //做个自动生成模块
}

var roomId = struct {
	Id     int
	locker sync.Mutex
}{Id: 0, locker: sync.Mutex{}}

func GetRoomId() int {
	roomId.locker.Lock()
	defer roomId.locker.Unlock()
	roomId.Id++
	return roomId.Id
}

//初始化一个房间，读取json配置，载入房间配置
func initARoom(gName string) *Room {
	r := &Room{Clients_r: make(map[int]map[int]*Conn), SharedDataQue: make(chan Data, 10)}
	r.Clients_r[0] = make(map[int]*Conn, 10) //@todo against config rules to initiate map
	r.Clients_r[1] = make(map[int]*Conn, 10) //@todo against config rules to initiate map
	r.Id = GetRoomId()
	r.GameName = gName
	room[r.Id] = r
	go r.initChan()
	return r
}

//put client into room against game rule defined by game name
func (r *Room) Add(c *Conn) error {
	if r.IsOverLimit(c.ClientType) {
		return errors.New("over limit against game rules")
	}
	r.Clients_r[c.ClientType][c.Id] = c //map 未初始化,没有锁@todo
	return nil
}
func (r *Room) Del(c *Conn) {
	delete(r.Clients, c.Id)
}
func (r *Room) Receive(d Data) {
	r.SharedDataQue <- d
}
func (r *Room) initChan() {
	for v := range r.SharedDataQue {
		for _, b := range r.Clients_r {
			for i, d := range b {
				println(i, v.From.Id)
				if i == v.From.Id {
					continue
				}
				d.Write(v)
			}
		}
	}
}

type Data struct {
	Type     int
	Conntent []byte
	From     *Conn
	Time     int64
}

var WsHandler websocket.Handler = func(con *websocket.Conn) {
	c, err := InitAConn(con)
	if err != nil {
		con.Write([]byte("wrong happen")) //@todo something can be parsed by client and ready to close connection
		fmt.Println(err)
		return
	}
	con.Write([]byte("{\"rid\":" + strconv.Itoa(c.Room.Id) + "}"))
	switch con.Request().URL.Path {
	case "/":
		var buffer []byte
		for {
			msg := make([]byte, 1024)
			n, err := con.Read(msg)
			if err != nil {
				println(err.Error())
				return
			}
			buffer = append(buffer, msg[:n]...)
			if n >= 1024 {
				continue
			}
			r := c.Room
			r.Receive(Data{Type: 1, Conntent: append([]byte{}, buffer...), From: c})
			buffer = nil
		}
	case "/gobang":
		msg := make([]byte, 1024)
		n, err := con.Read(msg)
		if err != nil {
			println(err.Error())
			return
		}
		roomId := getRoomId(msg[:n])
		c := NewConn(con, roomId)
		for {
			msg := make([]byte, 1024)
			n, err := con.Read(msg)
			if err != nil {
				println(err.Error())
				return
			}
			if r, ok := room[1]; ok {
				r.Receive(Data{Type: 1, Conntent: msg[:n], From: c})
			}
		}
	}
}

func getRoomId(msg []byte) int {
	key := &Key{}
	if json.Unmarshal(msg, key) != nil {
		return 0
	}
	if key.RoomId > 0 && key.RoomId < 1000000 {
		return key.RoomId
	}
	return 0
}

//初始化一个连接，配置好进入游戏房间，加载房间配置,解析客户端请求,向客户端写入初始化数据
func InitAConn(c *websocket.Conn) (*Conn, error) {
	r := c.Request()
	r.ParseForm()
	roomId := r.FormValue("roomId")
	name := r.FormValue("name")
	gName := r.FormValue("gName")
	clientType := r.FormValue("clientType")
	sidStr := r.FormValue("sid")
	rid, err := strconv.ParseInt(roomId, 10, 64)
	if err != nil {
		rid = 0
	}
	cType, err := strconv.ParseInt(clientType, 10, 64)
	if err != nil {
		cType = 0
	}
	sid, err := strconv.ParseInt(sidStr, 10, 64)
	if err != nil {
		sid = 0
	}
	var gameRoom *Room
	if roomId != "" {
		if c, ok := room[int(rid)]; !ok {
			return nil, RoomNotExist
		} else {
			gameRoom = c
		}
	} else {
		gameRoom = initARoom(gName)
	}
	client := NewConn(c, int(cType))
	client.Name = name
	client.Sid = int(sid)
	client.Room = gameRoom
	return client, gameRoom.Add(client)
}
