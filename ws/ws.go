package ws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	//"groups/log"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)
import "errors"

const BEGIN = 0
const RUNNING = 1
const WAITING = 0
const CLOSING = 2
const ENDGAME = 3
const IdMask = 0x0f
const GAMECONFIGDIR = "game"

var RoomNotExist = errors.New("room not exist")
var IdNum = 0
var roomManager *RoomManager

type RoomManager struct {
	Rooms     map[int]*Room
	Match     map[string][]*Room
	ConfigDir string
	locker    sync.Mutex
}

func (m RoomManager) SeeEveryRoom() {
	var oldStr string
	for {
		time.Sleep(2 * time.Second)
		var str string
		for i, v := range m.Rooms {
			for j, k := range v.Clients_r {
				if len(k) > 0 {
					str += fmt.Sprintf("at room \033[31m%d\033[0m there are \033[31m%d \033[0mclients with type \033[31m%d \033[0m \n", i, len(k), j)
				}
			}
		}
		if str != oldStr {
			oldStr = str
			fmt.Println(oldStr)
		}
	}
}

func init() {
	roomManager = &RoomManager{Rooms: make(map[int]*Room, 10), Match: make(map[string][]*Room, 10), locker: sync.Mutex{}}
	go roomManager.SeeEveryRoom()
}
func (m *RoomManager) CloseARoom(r Room) {
	delete(m.Rooms, r.Id)
}
func (m *RoomManager) Add(r *Room) {
	m.locker.Lock()
	defer m.locker.Unlock()
	m.Rooms[r.Id] = r
	if r.Status == WAITING {
		m.Match[r.GameName] = append(m.Match[r.GameName], r)
	}
}
func (m *RoomManager) GetAvaliableRoom(game string) *Room {
	m.locker.Lock()
	defer m.locker.Unlock()
	if list, ok := m.Match[game]; ok {
		for i, v := range list {
			if list[i].Status == WAITING {
				m.Match[game] = m.Match[game][i:]
				return v
			}
		}
	}
	return nil
}

type Key struct {
	RoomId int
	Name   string
}

func init() {
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
	(&d).putType()
	switch d.Type {
	default:
		c.Conn.Write(d.Content)
		return nil
	}
}

func (c *Conn) Close() {
	if c.Room != nil {
		c.Room.DeleteConn(c)
	}
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
	Status        int
	SharedDataQue chan Data
	Id            int
	ClientsLeft   []*Conn
	Clients_r     map[int]map[int]*Conn
	GameName      string //房间所属游戏的名字
}

func (r *Room) ShouldStart() bool {
	m := make(map[int]int)
	for i, v := range r.Clients_r {
		m[i] = len(v)
	}
	for i, v := range roomConfig[r.GameName].Start {
		if m[i] < v {
			return false
		}
	}
	return true
}
func (r Room) IsOverLimit(t int) bool {
	if m, ok := r.Clients_r[t]; ok {
		return IsOverLimit(r.GameName, t, len(m))
	}
	//log.Log("no such limit at " + r.GameName + " at " + strconv.Itoa(t))
	return true
}

func (r *Room) DeleteConn(c *Conn) {
	if m, ok := r.Clients_r[c.ClientType]; ok {
		delete(m, c.Id)
		r.EndGame()
	}
}
func (r *Room) EndGame() {
	if r.SatisfyEndRule() {
		r.Notify(ENDGAME)
	}
}
func (r *Room) SatisfyEndRule() bool {
	for i, v := range roomConfig[r.GameName].End {
		if len(r.Clients_r[i]) <= v {
			return true
		}
	}
	return false
}

//房间配置信息,决定房间有多少人，房间基础信息
type RoomConfig struct {
	GameName string
	L        map[string]int `json:"limit"`
	Limit    map[int]int    `json:"l"`
	S        map[string]int `json:"start"` //the rules of start a game
	Start    map[int]int    `json:"f"`
	E        map[string]int `json:"end"` // the rules of end a game
	End      map[int]int    `json:"g"`
}

func (r *RoomConfig) ParseIntoIntRule() error {
	r.Limit = make(map[int]int)
	r.Start = make(map[int]int)
	r.End = make(map[int]int)
	for i, v := range r.L {
		k, err := strconv.ParseInt(i, 10, 64)
		r.Limit[int(k)] = v
		if err != nil {
			return err
		}
	}
	for i, v := range r.S {
		k, err := strconv.ParseInt(i, 10, 64)
		r.Start[int(k)] = v
		if err != nil {
			return err
		}
	}
	for i, v := range r.E {
		k, err := strconv.ParseInt(i, 10, 64)
		r.End[int(k)] = v
		if err != nil {
			return err
		}
	}
	return nil
}

func InitConfig() error {
	finfo, err := os.Stat(GAMECONFIGDIR)
	if err != nil {
		return err
	}
	if !finfo.IsDir() {
		return errors.New("configpath is not a directory")
	}
	f, err := os.Open(GAMECONFIGDIR)
	if err != nil {
		return err
	}
	defer f.Close()
	fi, err := f.Readdir(-1)
	if err != nil {
		return err
	}
	for _, v := range fi {
		if v.IsDir() {
			continue
		}
		name := v.Name()
		if ok, err := regexp.Match(".json$", []byte(name)); !ok || err != nil {
			continue
		}
		f, err := os.Open(GAMECONFIGDIR + "/" + name)
		if err != nil {
			fmt.Println("open GAMECONFIGDIR game file", err.Error())
			continue
		}
		byt, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Println("read config file", err.Error())
			continue
		}
		rc := RoomConfig{GameName: name[:len(name)-5]}
		if err = json.Unmarshal(byt, &rc); err != nil {
			fmt.Println("read config", err.Error())
			continue
		}
		if err = (&rc).ParseIntoIntRule(); err != nil {
			return err
		}
		fmt.Println(rc)
		roomConfig[rc.GameName] = rc
	}
	return nil
}

func (r RoomConfig) IsOverLimit(t int, total int) bool {
	if limit, ok := r.Limit[t]; ok {
		fmt.Print("total limit", total, limit)
		return total >= limit
	} //！ok 应该做错误日志
	//log.Log("fatal: no such game limit: " + r.GameName + " at " + strconv.Itoa(t)) //这个日志意味着有程序内部错误
	return true
}

var roomConfig map[string]RoomConfig

func IsOverLimit(name string, t, total int) bool {
	if r, ok := roomConfig[name]; ok {
		return r.IsOverLimit(t, total)
	}
	//log.Log("no such game :" + name)
	return true
}

func init() {
	roomConfig = make(map[string]RoomConfig, 4)
	InitConfig()
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
	roomManager.Add(r)
	go r.initChan()
	go r.run()
	return r
}

//put client into room against game rule defined by game name
func (r *Room) Add(c *Conn) error {
	if r.IsOverLimit(c.ClientType) {
		return errors.New("over limit against game rules")
	}
	r.Clients_r[c.ClientType][c.Id] = c //map 未初始化,没有锁@todo
	r.ShouldStart()
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

func (r *Room) run() { //@todo a better way to achieve the feature?
	r.Status = WAITING
	for {
		time.Sleep(1 * time.Second)
		if r.ShouldStart() && r.Status == WAITING {
			r.Status = RUNNING
			r.Notify(BEGIN)
		}
		if r.IsEmpty() {
			r.Status = CLOSING
			r.Close()
			return
		}
	}
}
func (r Room) Notify(flag int) {
	switch flag {
	case BEGIN: //represent the start of game
		sData := GameStartData{User: make(map[string]int, 10), Start: true}
		for i, v := range r.Clients_r {
			for id, _ := range v {
				sData.User[strconv.Itoa(id)] = i
			}
		}
		msg, err := json.Marshal(sData)
		if err != nil {
			println(err.Error())
			return
		}
		d := Data{Type: 200, Content: msg, Time: time.Now().Unix()} //200 游戏开始，游戏正常信号
		r.Broadcast(d)
	case ENDGAME:
		d := Data{Type: 300, Content: []byte("{\"end\":true}"), Time: time.Now().Unix()} //300 游戏结束，游戏异常信号
		r.Broadcast(d)
	}
}

func (r Room) IsEmpty() bool {
	for _, v := range r.Clients_r {
		if len(v) != 0 {
			return false
		}
	}
	return true
}
func (r Room) Close() {
	close(r.SharedDataQue)
	roomManager.CloseARoom(r)
}
func (r *Room) Broadcast(d Data) {
	fmt.Println("data", d)
	for _, v := range r.Clients_r {
		for _, con := range v {
			con.Write(d)
		}
	}
}

type Data struct {
	Type    int
	Content []byte
	From    *Conn
	Time    int64
}

func (d *Data) putType() {
	str := strings.Replace(string(d.Content), "\"T\"", "\"_t\"", -1) // escape T
	str = strings.Replace(str, "'T'", "\"_t\"", -1)
	d.Content = []byte(str)
	l := len(d.Content)
	if l > 2 {
		if d.From == nil {
			d.Content = append(d.Content[:l-1], []byte(",\"T\":"+strconv.Itoa(d.Type)+",\"id\":"+strconv.Itoa(1)+"}")...)
		} else {
			d.Content = append(d.Content[:l-1], []byte(",\"T\":"+strconv.Itoa(d.Type)+",\"id\":"+strconv.Itoa(d.From.Id)+"}")...)
		}
		return
	}
	d.Content = []byte("{\"T\":" + strconv.Itoa(d.Type) + "}")
}

type GameStartData struct {
	User  map[string]int `json:"user"` //@todo should be user infomation?
	Start bool           `json:"start"`
}

var WsHandler websocket.Handler = func(con *websocket.Conn) {
	c, err := InitAConn(con)
	if err != nil {
		Content := []byte("{\"error\":\"wrong happen\",\"T\":100}") //@todo something can be parsed by client and ready to close connection
		con.Write(Content)
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
				c.Close()
				return
			}
			buffer = append(buffer, msg[:n]...)
			if n >= 1024 {
				continue
			}
			r := c.Room
			r.Receive(Data{Type: 1, Content: append([]byte{}, buffer...), From: c})
			buffer = nil
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
	act := r.FormValue("act")
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
		if c, ok := roomManager.Rooms[int(rid)]; !ok {
			return nil, RoomNotExist
		} else {
			gameRoom = c
		}
	} else {
		println(act)
		switch act {
		case "join":
			gameRoom = roomManager.GetAvaliableRoom(gName)
		default:
			gameRoom = initARoom(gName)
		}
	}
	if gameRoom == nil {
		return nil, errors.New("no such game room")
	}
	client := NewConn(c, int(cType))
	client.Name = name
	client.Sid = int(sid)
	client.Room = gameRoom
	return client, gameRoom.Add(client)
}
