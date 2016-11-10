package server

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
)

type MsgPack struct {
	Type string          `json:"type"`
	Data string          `json:"data"`
	Conn *websocket.Conn `json:"-"`
}

type CloseHandler func(conn *websocket.Conn)

type WsConn struct {
	conn *websocket.Conn
	ch CloseHandler
}

var cm map[*websocket.Conn]*WsConn = make(map[*websocket.Conn]*WsConn)

func Start(port int) {
	fmt.Println("Starting WebSocket Server.")
	go dispatch()
	http.Handle("/", http.FileServer(http.Dir("."))) // <-- note this line
	http.Handle("/socket", websocket.Handler(addConn))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

	fmt.Println("WebSocket Server Stopped.")
}

func Stop() {
}

func addConn(ws *websocket.Conn) {
	wc := &WsConn{conn:ws}
	cm[ws] = wc
	recv(ws)
}

func delConn(ws *websocket.Conn) {
	fmt.Println("移除玩家：delConn")

	if wc, ok := cm[ws]; ok {
		if wc.ch != nil {
			wc.ch(ws)
		}

		delete(cm, ws)
	}
}

func HookClose(ws *websocket.Conn, ch CloseHandler) {
	if wc, ok := cm[ws]; ok {
		wc.ch = ch
	}
}

func recv(ws *websocket.Conn) {
	var err error

	for {
		var pack MsgPack

		if err = websocket.JSON.Receive(ws, &pack); err != nil {
			fmt.Println("Web Socket数据接收错误: ", err)
			delConn(ws)
			break
		}

		pack.Conn = ws
		msgCh <- &pack
	}
}

func Send(pack *MsgPack) {
	if err := websocket.JSON.Send(pack.Conn, pack); err != nil {
		fmt.Println("Web Socket发送数据错误：", err)
		delConn(pack.Conn)
	}
}

func SendData(t string, data interface{}, ws *websocket.Conn) {
	pack := MsgPack{Type: t, Conn: ws}

	if b, err := json.Marshal(data); err == nil {
		pack.Data = string(b)
	}

	Send(&pack)
}

func Reply(pack *MsgPack, data interface{}) {
	v, _ := json.Marshal(data)
	reply := MsgPack{Type: pack.Type + "_r", Data: string(v), Conn: pack.Conn}
	Send(&reply)
}
