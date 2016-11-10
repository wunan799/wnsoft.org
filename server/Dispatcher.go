package server

import (
	"fmt"
)

type MsgHandler func(*MsgPack)

var msgCh chan *MsgPack = make(chan *MsgPack, 500)
var handlerMap map[string]MsgHandler = make(map[string]MsgHandler)

func RegHandler(mtype string, handler MsgHandler) {
	handlerMap[mtype] = handler
}

func UnregHandler(mtype string) {
	handlerMap[mtype] = nil
}

func FireMsg(pack *MsgPack) {
	msgCh <- pack
}

func dispatch() {
	for pack := range msgCh {
		fmt.Println("收到消息：", pack)

		if h, ok := handlerMap[pack.Type]; ok {
			h(pack)
		}
	}
}
