package doudizhu

import (
	"math/rand"
	"time"
	"fmt"
)

const (
	CT_ERROR               = iota //错误类型
	CT_SINGLE                     //单牌类型
	CT_DOUBLE                     //对牌类型
	CT_THREE                      //三条类型
	CT_THREE_TAKE_ONE			  //三带一单
	CT_THREE_TAKE_TWO             //三带一对
	CT_SINGLE_LINE                //单连类型
	CT_DOUBLE_LINE                //对连类型
	CT_THREE_LINE                 //三连类型
	CT_THREE_LINE_TAKE_ONE        //连三带单
	CT_THREE_LINE_TAKE_TWO        //连三带对
	CT_FOUR_LINE_TAKE_ONE         //四带两单
	CT_FOUR_LINE_TAKE_TWO         //四带两对
	CT_BOMB_CARD                  //炸弹类型
	CT_MISSILE_CARD               //火箭类型
	CT_PASS               //过牌
)

const (
	CC_FANGKUAI = iota //方块
	CC_CAOHUA          //梅花
	CC_HONGTAO         //红桃
	CC_HEITAO          //黑桃
)

func randomCard(cards *[3][20]byte, back *[3]byte) {
	var m [54]int
	leftNum := len(m)

	for i := 0; i < leftNum; i++ {
		m[i] = i
	}

	rand.Seed(time.Now().UnixNano())

	var doRand = func(c *[20]byte) {
		for i := 0; i < 17; i++ {
			index := rand.Intn(leftNum)
			c[i] = byte(m[index])
			m[index] = m[leftNum-1]
			leftNum--
		}

		c[17] = INVALID_BYTE
		c[18] = INVALID_BYTE
		c[19] = INVALID_BYTE
	}

	doRand(&cards[0])
	doRand(&cards[1])
	doRand(&cards[2])
	back[0] = byte(m[0])
	back[1] = byte(m[1])
	back[2] = byte(m[2])
}

/**
* cards - 带出的牌
* handCards - 手牌
* lastCards - 上家出的牌
* first 是否第一首牌
**/
func isValid(cards []byte, handCards *[20]byte, lastCards []byte, first bool) int {
	total := len(cards)

	if ((total == 0) && !first) {
		return CT_PASS
	}

	var b bool

	//判断手里是否有要出的牌
	for _, sv := range cards {
		b = false

		for i, hv := range handCards {
			if sv == hv {
				handCards[i] = INVALID_BYTE
				b = true
				break
			}
		}

		if !b {
			return CT_ERROR
		}
	}

	myType, myKind := getCardKind(cards)
	fmt.Println("牌型：", myType, myKind, first)

	if myType == CT_ERROR {
		return CT_ERROR
	}

	if first {
		return int(myType)
	}

	lastTotal := len(lastCards)
	lastType, lastKind := getCardKind(lastCards)

	if compareCards(myType, lastType, myKind, lastKind, total, lastTotal) {
		return int(myType)
	}

	return CT_ERROR
}

func getCardColor(card byte) byte {
	return card % 4
}

func getCardValue(card byte) byte {
	return card / 4 + 3
}

func getCardLogicValue(card byte) byte {
	color := getCardColor(card)
	value := getCardValue(card)

	//大小鬼
	if value > 15 {
		value += color
	}

	return value
}

func getCardKind(cards []byte) (byte, *[4][]byte) {
	n := len(cards)

	if n == 0 {
		return CT_ERROR, nil
	}

	if (n == 2) && (cards[0] == 52) && (cards[1] == 53) {
		return CT_MISSILE_CARD, nil
	}

	cardType := analyzeCards(cards)

	if r, b := judgeFour(cardType, n); b {
		return r, cardType
	}

	if r, b := judgeThree(cardType, n); b {
		return r, cardType
	}

	if r, b := judgeDouble(cardType, n); b {
		return r, cardType
	}

	if r, b := judgeSingle(cardType, n); b {
		return r, cardType
	}

	return CT_ERROR, nil
}

//返回值是牌型数组，0-single 1-double 2-three 4-four
func analyzeCards(cards []byte) *[4][]byte {
	var r [4][]byte
	n := len(cards)

	for i := 0; i < n; {
		same := 1
		value := getCardLogicValue(cards[i])

		for j := i + 1; j < n; j++ {
			if getCardLogicValue(cards[j]) != value {
				break
			}

			same++
		}

		//取牌型相应数组
		cardType := &r[same - 1]

		for k := 0; k < same; k++ {
			*cardType = append(*cardType, cards[i + k])
		}

		i += same
	}

	return &r
}

//判断是否有四张同牌
func judgeFour(cards *[4][]byte, total int) (byte, bool) {
	count := len(cards[3]) / 4

	if count < 1 {
		 return INVALID_BYTE, false
	}

	if count > 1 {
		return CT_ERROR, true
	}

	if total == 4 {
		return CT_BOMB_CARD, true
	}

	if (total == 6) && (len(cards[0]) == 2) {
		return CT_FOUR_LINE_TAKE_ONE, true
	}

	if (total == 8) && (len(cards[1]) / 2 == 2) {
		return CT_FOUR_LINE_TAKE_TWO, true
	}

	return CT_ERROR, true
}

//判断是否有三张同牌
func judgeThree(cards *[4][]byte, total int) (byte, bool) {
	count := len(cards[2]) / 3

	if count < 1 {
		 return INVALID_BYTE, false
	}

	if count == 1 {
		if total == 3 {
			return CT_THREE, true
		}

		if total == 4 {
			return CT_THREE_TAKE_ONE, true
		}

		if (total == 5) && (len(cards[1]) == 2) {
			return CT_THREE_TAKE_TWO, true
		}
	}

	first := getCardLogicValue(cards[2][0])

	//A以上的牌无法凑成3张飞机
	if (first > 14) && (count > 1) {
		return CT_ERROR, true
	}

	//是否连牌判断
	for i := 1; i < count; i++ {
		if getCardLogicValue(cards[2][i * 3]) != (first + byte(i)) {
			return CT_ERROR, true
		}
	}

	if (count * 3) == total {
		return CT_THREE_LINE, true
	}

	if (count * 4) == total {
		return CT_THREE_LINE_TAKE_ONE, true
	}

	if ((count * 5) == total) && (len(cards[1]) / 2 == count) {
		return CT_THREE_LINE_TAKE_TWO, true
	}

	return CT_ERROR, true
}

func judgeDouble(cards *[4][]byte, total int) (byte, bool) {
	count := len(cards[1]) / 2

	if (count * 2) != total {
		return INVALID_BYTE, false
	}

	if count == 1 {
		return CT_DOUBLE, true
	}

	if count < 3 {
		return INVALID_BYTE, false
	}

	first := getCardLogicValue(cards[1][0])

	//K以上的牌无法凑成2张飞机
	if first > 13 {
		 return CT_ERROR, true
	}

	//是否连牌判断
	for i := 1; i < count; i++ {
		if getCardLogicValue(cards[1][i * 2]) != (first + byte(i)) {
			return CT_ERROR, true
		}
	}

	return CT_DOUBLE_LINE, true
}

func judgeSingle (cards *[4][]byte, total int) (byte, bool) {
	count := len(cards[0])

	if count != total {
		return INVALID_BYTE, false
	}

	if count == 1 {
		return CT_SINGLE, true
	}

	if count < 5 {
		return INVALID_BYTE, false
	}

	first := getCardLogicValue(cards[0][0])

	//J以上的牌无法凑成1张飞机
	if first > 11 {
		return CT_ERROR, true
	}

	//是否连牌判断
	for i := 1; i < count; i++ {
		if getCardLogicValue(cards[0][i]) != (first + byte(i)) {
			return CT_ERROR, true
		}
	}

	return CT_SINGLE_LINE, true
}

func compareCards(myType byte, lastType byte, myKind *[4][]byte,
	lastKind *[4][]byte, myTotal int, lastTotal int) bool {
	//火箭判断
	if myType == CT_MISSILE_CARD {
		return true
	}

	if lastType == CT_MISSILE_CARD {
		return false
	}

	//炸弹判断
	if (myType == CT_BOMB_CARD) && (lastType != CT_BOMB_CARD) {
		return true
	}

	if (lastType == CT_BOMB_CARD) && (myType != CT_BOMB_CARD) {
		return false
	}

	//规则判断
	if (myType != lastType) || (myTotal != lastTotal) {
		return false
	}

	for i := 3; i >=0; i-- {
		if len(myKind[i]) == 0 {
			continue
		}

		return myKind[i][0] > lastKind[i][0]
	}

	return false
}