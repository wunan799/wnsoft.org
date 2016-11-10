package main

import (
	"fmt"
	"wnsoft.org/doudizhu"
	"wnsoft.org/server"
)

type MsgType struct {
	T    string
	Data interface{}
}

type MyMsg struct {
	Id   int
	Name string
}

//func Echo(ws *websocket.Conn) {
//	var err error
//
//	for {
//
//		var reply string
//
//		if err = websocket.Message.Receive(ws, &reply); err != nil {
//			fmt.Println("Can't receive")
//			break
//		}
//
//		var msgType MsgType
//		json.Unmarshal([]byte(reply), &msgType)
//
//		if msgType.T == "login" {
//			var m MyMsg
//			json.Unmarshal([]byte(msgType.Data), &m)
//			fmt.Println(m)
//		}
//
//		fmt.Println("Received back from client: " + reply)
//
//		msg := "Received:  " + reply
//		fmt.Println("Sending to client: " + msg)
//
//		if err = websocket.Message.Send(ws, msg); err != nil {
//			fmt.Println("Can't send")
//			break
//		}
//	}
//}

//func main() {
//	fmt.Println("begin")
//	http.Handle("/", http.FileServer(http.Dir("."))) // <-- note this line
//
//	http.Handle("/socket", websocket.Handler(Echo))
//
//	if err := http.ListenAndServe(":1234", nil); err != nil {
//		log.Fatal("ListenAndServe:", err)
//	}
//
//	fmt.Println("end")
//}

func handle(pack *server.MsgPack) {
	fmt.Println("处理消息：", pack)
	reply := server.MsgPack{Type: "login_r", Data: "hello, jack!", Conn: pack.Conn}
	server.Send(&reply)
}

func main() {
	doudizhu.Init()
}
