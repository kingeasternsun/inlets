package util

import (
	"sync"
	"time"
)

type Order struct {
	OrderID string
}

type HitFunc func(info Order) (ret int, err error)
type OneHit struct {
	mp map[string]HitOrder
	// hit map[string]int
	sync.RWMutex
	MaxHit  int           // 订单被触发几次，进行优先推送
	Dur     time.Duration //秒为单位，周期检查是否有订单要优先推送
	Handle  HitFunc
	OrderCh chan string //no found触发达到阈值的订单放在这里
}

type HitOrder struct {
	Info Order
	Hit  int
}

func NewOneHit(maxHit int, dur time.Duration, fn HitFunc) OneHit {

	return OneHit{mp: make(map[string]HitOrder, 0),
		// hit:    make(map[string]int, 0),
		MaxHit:  maxHit,
		Dur:     dur,
		Handle:  fn,
		OrderCh: make(chan string, 10),
	}

}

func (m *OneHit) Add(order Order) {

	if order.OrderID == "" {
		return
	}

	m.Lock()

	tmp, ok := m.mp[order.OrderID]
	//不存在
	if !ok {
		Debug("add new order to pri pull", order.OrderID)
		m.mp[order.OrderID] = HitOrder{Info: order, Hit: 1}
		m.Unlock()
		return
	}

	//存在就更新次数
	Debugf("order(%v) hits:%v", order.OrderID, tmp.Hit)

	if tmp.Hit >= m.MaxHit {
		select {
		case m.OrderCh <- tmp.Info.OrderID:
			Debug("TO add order to pri pull", tmp.Info.OrderID)
			tmp.Hit = -10000000
		default:

		}
	}

	tmp.Hit = tmp.Hit + 1

	m.mp[order.OrderID] = tmp

	m.Unlock()

}

//license拉取成功后，删除
func (m *OneHit) Del(orderID string) {
	m.Lock()
	defer m.Unlock()
	delete(m.mp, orderID)
}

func (m *OneHit) Get(orderID string) (info HitOrder, ok bool) {

	m.Lock()
	info, ok = m.mp[orderID]
	m.Unlock()
	return
}

func (m *OneHit) StartCheck() {

	select {
	case orderID := <-m.OrderCh:
		v, ok := m.Get(orderID)
		if ok {
			Debugf("put order(%v) in to priority pull", v.Info.OrderID)
			m.Handle(v.Info)
		}

	}
}
