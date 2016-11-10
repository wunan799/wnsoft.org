package doudizhu

import (
	"encoding/json"
	"fmt"
	"wnsoft.org/server"
	"wnsoft.org/user"
	"golang.org/x/net/websocket"
)

type JoinTable struct {
	UId   int `json:"uid"`
	Table int `json:"table"`
	Index int `json:"index"`
	Name string `json:"name"`
}

type LeaveTable struct {
	Id int `json:"id"`
	Table int `json:"table"`
}

type PlayerReady struct {
	UId   int `json:"uid"`
	Table int `json:"table"`
}

type TableState struct {
	State int    `json:"state"`
	Table int    `json:"table"`
	Data  string `json:"data"`
	Clock int `json:"clock"`
}

type CallState struct {
	Score int `json:"score"`
	Index int `json:"index"`
	Table int `json:"table"`
}

type CallEnd struct {
	Score     int     `json:"score"`
	Banker    int     `json:"banker"`
	BackCards [3]byte `json:"back_cards"`
}

type PlayState struct {
	Index int   `json:"index"`
	Cards []int `json:"cards, omitempty"`
	Table int   `json:"table"`
	First bool  `json:"first"`
	Kind int `json:"kind"`
}

type ScoreData struct {
	LandScore int `json:"land_score"`
	BombMultiple int `json:"bomb_multiple"`
	Banker int `json:"banker"`
	Winner int `json:"winner"`
	Spring int `json:"spring"`
}

const (
	STATE_WAITPLAYER = iota + 1
	STATE_ALLREADY
	STATE_GIVECARD
	STATE_CALLING
	STATE_ENDCALL
	STATE_PLAYING
	STATE_GAMEOVER
)

var tables map[int]*Table = make(map[int]*Table)

func Init() {
	server.RegHandler("join_table", joinTable)
	server.RegHandler("leave_table", leaveTable)
	server.RegHandler("player_ready", playerReady)
	server.RegHandler("table_state", stateChange)
	server.RegHandler("call_score", callScore)
	server.RegHandler("send_card", sendCard)
	user.Init()
	server.Start(10010)
}

func newTable(id int) *Table {
	t := &Table{Id: id, State: STATE_WAITPLAYER}
	tables[id] = t
	t.init()
	return t
}

func joinTable(pack *server.MsgPack) {
	var jt JoinTable

	if err := json.Unmarshal([]byte(pack.Data), &jt); err != nil {
		fmt.Println("加入牌桌解析错误：", err)
		server.Reply(pack, 1)
		return
	}

	t := tables[jt.Table]

	if t == nil {
		t = newTable(jt.Table)
	}

	jt.Index = t.addPlayer(jt.UId, jt.Name)
	server.HookClose(pack.Conn, breakOff(t))

	if b, err := json.Marshal(&jt); err == nil {
		pack.Data = string(b)
	}

	var tmp Table = *t
	server.Reply(pack, tmp)
	broadcast(t, pack)
}

func breakOff(t *Table) server.CloseHandler {
	return func(conn *websocket.Conn) {
		fmt.Println("移除玩家, break off")

		for _, v := range t.Player  {
			if u := user.GetUser(v.Id); u != nil {
				if u.Conn == conn {
					t.delPlayer(v.Id)
					break
				}
			}
		}
	}
}

func leaveTable(pack *server.MsgPack) {
	var leave LeaveTable

	if err := json.Unmarshal([]byte(pack.Data), &leave); err != nil {
		fmt.Println("离开消息解码错误：", err)
		return
	}

	t := tables[leave.Table]

	if t == nil {
		return
	}

	breakOff(t)(pack.Conn)
}

func playerReady(pack *server.MsgPack) {
	var pr PlayerReady

	if err := json.Unmarshal([]byte(pack.Data), &pr); err != nil {
		fmt.Println("玩家准备就绪：", err)
		server.Reply(pack, 1)
		return
	}

	t := tables[pr.Table]

	if t == nil {
		server.Reply(pack, 1)
		return
	}

	t.playerReady(pr.UId)
	server.Reply(pack, 0)
	broadcast(t, pack)
}

func stateChange(pack *server.MsgPack) {
	var ts TableState

	if err := json.Unmarshal([]byte(pack.Data), &ts); err != nil {
		return
	}

	t := tables[ts.Table]

	if t.State == STATE_GIVECARD {
		t.prepareCard()

		for i, v := range t.Player {
			if u := user.GetUser(v.Id); u != nil {
				if b, err := json.Marshal(t.handCard.cards[i]); err == nil {
					ts.Data = string(b)
					server.SendData(pack.Type, &ts, u.Conn)
				}
			}
		}
	} else if t.State == STATE_CALLING {
		call := CallState{Index: int(t.round.curPlayer)}
		broadcastState(&ts, &call, t)
	} else if t.State == STATE_ENDCALL {
		end := CallEnd{
			Score:     int(t.round.callInfo[t.round.banker]),
			Banker:    int(t.round.banker),
			BackCards: t.round.backCards,
		}

		broadcastState(&ts, &end, t)
	} else if t.State == STATE_PLAYING {
		play := PlayState{Index: t.round.curPlayer, First: t.isFirstHand()}
		broadcastState(&ts, &play, t)
	} else if t.State == STATE_GAMEOVER {
		score := ScoreData{LandScore:t.round.landScore,
			BombMultiple:t.round.bombMultiple,
			Winner:t.round.winner,
			Banker:t.round.banker,
		}

		broadcastState(&ts, &score, t)
	} else {
		broadcast(t, pack)
	}
}

func callScore(pack *server.MsgPack) {
	var call CallState

	if err := json.Unmarshal([]byte(pack.Data), &call); err != nil {
		fmt.Println("叫分解析错误：", err)
		return
	}

	t := tables[call.Table]

	if t == nil {
		return
	}

	t.callScore(call.Index, call.Score)
	//server.Reply(pack, 0)
	//broadcast(t, pack)
}

func sendCard(pack *server.MsgPack) {
	var send PlayState

	if err := json.Unmarshal([]byte(pack.Data), &send); err != nil {
		fmt.Println("出牌解析错误：", err)
		return
	}

	t := tables[send.Table]

	if t == nil {
		return
	}

	cards := make([]byte, len(send.Cards))

	for i, v := range send.Cards {
		cards[i] = byte(v)
	}

	t.sendCard(send.Index, cards)
}

func broadcast(t *Table, pack *server.MsgPack) {
	for _, v := range t.Player {
		if v.Id != 0 {
			if u := user.GetUser(v.Id); u != nil {
				pack.Conn = u.Conn
				server.Send(pack)
			}
		}
	}
}

func broadcastCall(index int, score int, t *Table) {
	call := &CallState{Index: index, Score: score, Table: t.Id}

	for _, v := range t.Player {
		if u := user.GetUser(v.Id); u != nil {
			server.SendData("call_score", call, u.Conn)
		}
	}
}

func broadcastCard(index int, cards []byte, t *Table, kind int) {
	playState := &PlayState{Index: index, Table: t.Id, Kind:kind, First:t.isFirstHand()}
	playState.Cards = make([]int, len(cards))

	for i, v := range cards {
		playState.Cards[i] = int(v)
	}

	for _, v := range t.Player {
		if u := user.GetUser(v.Id); u != nil {
			server.SendData("send_card", playState, u.Conn)
		}
	}
}

func fireState(t *Table) {
	var ts TableState
	ts.Table = t.Id
	ts.State = t.State
	ts.Clock = t.limit[t.State]

	if b, err := json.Marshal(ts); err == nil {
		pack := server.MsgPack{Type: "table_state", Data: string(b)}
		server.FireMsg(&pack)
	}
}

func broadcastState(ts *TableState, data interface{}, t *Table) {
	if b, err := json.Marshal(data); err == nil {
		ts.Data = string(b)
	}

	for _, v := range t.Player {
		if u := user.GetUser(v.Id); u != nil {
			server.SendData("table_state", &ts, u.Conn)
		}
	}
}
