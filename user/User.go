package user

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"wnsoft.org/server"
)

type User struct {
	Id   int             `json:"id"`
	Name string          `json:"name"`
	Conn *websocket.Conn `json:"-"`
}

var userMap map[int]*User = make(map[int]*User)
var userId int = 1

func Init() {
	server.RegHandler("login", loginHandler)
}

func loginHandler(pack *server.MsgPack) {
	var user User

	if err := json.Unmarshal([]byte(pack.Data), &user); err != nil {
		fmt.Println("JSON解析错误，登录失败:", err)
		return
	}

	if user.Id == 0 {
		user.Id = userId
		userId++
	}

	user.Conn = pack.Conn
	userMap[user.Id] = &user
	pack.Type = "login_r"
	v, _ := json.Marshal(&user)
	pack.Data = string(v)
	server.Send(pack)
}

func GetUser(id int) *User {
	return userMap[id]
}
