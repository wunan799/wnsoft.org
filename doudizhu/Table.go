package doudizhu

import (
	"fmt"
	"math/rand"
	"time"
)

const INVALID_BYTE byte = 0xff

//当前回合信息
type Round struct {
	bombMultiple int     //倍数
	landScore    int     //底分
	firstCall    int     //首叫玩家
	banker       int     //庄家
	curPlayer    int     //当前玩家
	callInfo     [3]byte //叫分信息
	winner       int     //赢家
	backCards    [3]byte //底牌
}

//出牌信息
type PlayCard struct {
	cards [3][]byte //出牌数据
}

//手牌信息
type HandCard struct {
	cards [3][20]byte //手牌
	count [3]byte     //手牌数量
}

type Player struct {
	Id     int `json:"id"`
	Status int `json:"status"`
	Name string `json:"name"`
}

type Table struct {
	Id       int       `json:"id"`
	Player   [3]Player `json:"player"`
	State    int       `json:"state"`
	round    Round
	playCard PlayCard
	handCard HandCard
	ticker   *time.Ticker //定时器
	time     int          //当前状态计时
	limit    [STATE_GAMEOVER + 1]int          //当前状态限定时间
}

func (t *Table) init() {
	t.playCard.cards[0] = make([]byte, 20)
	t.playCard.cards[1] = make([]byte, 20)
	t.playCard.cards[2] = make([]byte, 20)
	t.reset()
	rand.Seed(time.Now().UnixNano())
	t.round.firstCall = rand.Intn(3)
	t.ticker = time.NewTicker(1 * time.Second)

	t.limit[STATE_GIVECARD] = 3
	t.limit[STATE_CALLING] = 5
	t.limit[STATE_ENDCALL] = 1
	t.limit[STATE_PLAYING] = 10
	t.limit[STATE_GAMEOVER] = 5

	go func() {
		for {
			if _, ok := <-t.ticker.C; ok {
				t.checkState()
			}
		}
	}()
}

func (t *Table) reset() {
	t.round.callInfo = [3]byte{INVALID_BYTE, INVALID_BYTE, INVALID_BYTE}
	t.round.bombMultiple = 0
	t.round.landScore = 0
	t.playCard.cards[0] = t.playCard.cards[0][0:0]
	t.playCard.cards[1] = t.playCard.cards[1][0:0]
	t.playCard.cards[2] = t.playCard.cards[2][0:0]
	t.Player[0].Status = 0
	t.Player[1].Status = 0
	t.Player[2].Status = 0
	t.changeState(STATE_WAITPLAYER)
}

func (t *Table) uninit() {
	if t.ticker != nil {
		t.ticker.Stop()
	}
}

func (t *Table) addPlayer(id int, name string) int {
	var index int

	for i, v := range t.Player {
		if v.Id == 0 {
			t.Player[i].Id = id
			t.Player[i].Name = name
			index = i
			break
		}
	}

	return index
}

func (t *Table) delPlayer(id int) {
	reset := false

	for i, v := range t.Player {
		if id == v.Id {
			t.Player[i].Id = 0;
			t.Player[i].Status = 0;
			fmt.Println("移除玩家：", id)
			reset = true
			break
		}
	}

	if reset {
		t.reset()
	}
}

func (t *Table) playerReady(id int) {
	index := t.getPlayerById(id)
	t.Player[index].Status = 1
}

func (t *Table) callScore(index int, score int) {
	if index != t.round.curPlayer {
		return
	}

	if t.round.callInfo[index] != INVALID_BYTE {
		return
	}

	if score != 0 {
		//只能叫更大的分
		if (t.round.callInfo[t.round.banker] != INVALID_BYTE) &&
			(score < int(t.round.callInfo[t.round.banker])) {
			return
		}

		t.round.banker = index
	}

	t.round.callInfo[index] = byte(score)
	broadcastCall(index, score, t)

	if t.checkCallEnd() {
		return
	}

	t.time = 0
	t.round.curPlayer = (t.round.curPlayer + 1) % 3
	fireState(t)
}

func (t *Table) sendCard(index int, cards []byte) {
	if index != t.round.curPlayer {
		fmt.Println("不是出牌顺序")
		return
	}

	handCards := t.handCard.cards[index]
	var kind int

	if kind = isValid(cards, &handCards, t.getLastCards(index),
		t.isFirstHand()); kind == CT_ERROR {
		fmt.Println("无效出牌")
		return
	}

	broadcastCard(index, cards, t, kind)

	if n := len(cards); n == 0 {
		t.playCard.cards[index] = t.playCard.cards[index][0:0]
	} else {
		t.handCard.cards[index] = handCards
		t.handCard.count[index] -= byte(n)
		t.playCard.cards[index] = t.playCard.cards[index][:n]

		for i, v := range cards {
			t.playCard.cards[index][i] = byte(v)
		}

		if (kind == CT_BOMB_CARD) || (kind == CT_MISSILE_CARD) {
			t.round.bombMultiple++
		}
	}

	if t.checkGameEnd() {
		return
	}

	t.time = 0
	t.round.curPlayer = (t.round.curPlayer + 1) % 3
	fireState(t)
}

func (t *Table) getPlayerById(id int) int {
	var index int = -1

	for i, v := range t.Player {
		if v.Id == id {
			index = i
			break
		}
	}

	return index
}

func (t *Table) checkState() {
	t.time += 1

	switch t.State {
	case STATE_WAITPLAYER:
		t.checkReady()
	case STATE_ALLREADY:
		t.changeState(STATE_GIVECARD)
	case STATE_GIVECARD:
		if t.time > t.limit[STATE_GIVECARD] {
			t.round.curPlayer = t.round.firstCall
			t.changeState(STATE_CALLING)
		}
	case STATE_CALLING:
		t.checkCall()
	case STATE_ENDCALL:
		if t.time > t.limit[STATE_ENDCALL] {
			t.changeState(STATE_PLAYING)
		}
	case STATE_PLAYING:
		t.checkPlay()
	case STATE_GAMEOVER:
		if t.time > t.limit[STATE_GAMEOVER] {
			t.reset()
		}
	}
}

func (t *Table) checkReady() {
	state := STATE_ALLREADY

	for _, v := range t.Player {
		if (v.Id == 0) || (v.Status == 0) {
			state = t.State
			break
		}
	}

	t.changeState(state)
}

func (t *Table) checkCall() {
	if t.checkCallEnd() {
		t.changeState(STATE_ENDCALL)
		t.giveBackCards()
		return
	} else if t.time > t.limit[STATE_CALLING] {
		t.callScore(t.round.curPlayer, 0) //不叫
	}
}

func (t *Table) checkCallEnd() bool {
	if t.round.callInfo[t.round.banker] == 3 {
		t.round.curPlayer = t.round.banker
		t.round.landScore = 3
		return true
	}

	if (t.round.callInfo[0] != INVALID_BYTE) &&
		(t.round.callInfo[1] != INVALID_BYTE) &&
		(t.round.callInfo[2] != INVALID_BYTE) {
		t.round.curPlayer = t.round.banker
		t.round.landScore = int(t.round.callInfo[t.round.banker])
		return true
	}

	return false
}

func (t *Table) checkGameEnd() bool {
	b := false

	for i, v := range t.handCard.count {
		if v == 0 {
			b = true
			t.round.winner = i
			t.round.firstCall = i
			break
		}
	}

	return b
}

func (t *Table) checkPlay() {
	if t.checkGameEnd() {
		t.changeState(STATE_GAMEOVER)
	} else if t.time > t.limit[STATE_PLAYING] {
		t.autoPlay()
	}
}

func (t *Table) changeState(state int) {
	if t.State == state {
		return
	}

	t.State = state
	t.time = 0
	fireState(t)
	fmt.Println("Change State: ", state)
}

func (t *Table) giveBackCards() {
	t.handCard.cards[t.round.banker][17] = t.round.backCards[0]
	t.handCard.cards[t.round.banker][18] = t.round.backCards[1]
	t.handCard.cards[t.round.banker][19] = t.round.backCards[2]
	t.handCard.count[t.round.banker] += 3
}

func (t *Table) isFirstHand() bool {
	index := t.round.curPlayer
	return (len(t.playCard.cards[(index+1)%3]) == 0) &&
		(len(t.playCard.cards[(index+2)%3]) == 0)
}

func (t *Table) prepareCard() {
	randomCard(&t.handCard.cards, &t.round.backCards)
	t.handCard.count[0] = 17
	t.handCard.count[1] = 17
	t.handCard.count[2] = 17
}

func (t *Table) getLastCards(index int) []byte {
	if last := (index + 2) % 3; len(t.playCard.cards[last]) > 0 {
		return t.playCard.cards[last]
	}

	if last := (index + 1) % 3; len(t.playCard.cards[last]) > 0 {
		return t.playCard.cards[last]
	}

	return nil
}

func (t *Table) autoPlay() {
	if !t.isFirstHand() {
		t.sendCard(t.round.curPlayer, []byte{})
	} else {
		for _, v := range t.handCard.cards[t.round.curPlayer] {
			if v != INVALID_BYTE {
				t.sendCard(t.round.curPlayer, []byte{v})
				break
			}
		}
	}
}