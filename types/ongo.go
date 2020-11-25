package types

import (
	"net/http"
	"sync"
	"time"
)

type OnGO struct {
	//维护下游client server的的列表，通过这个map可以将用户的请求req发送clientID对应的发送通道中
	//然后相应的goroutine从通道中取出来通过ws发送到下游server
	Clients map[string]Client
	Mutex   *sync.RWMutex

	Labels     map[string]string
	LabelMutex *sync.RWMutex
}

type Client struct { //在构建每个websocket链接的时候，会为对应client维护Client这样的一个结构
	Data     chan *http.Request `json:"data,omitempty"`
	Created  *time.Time         `json:"created,omitempty"`
	Addr     string             `json:"addr,omitempty"`
	ClientID string             `json:"client_id,omitempty"`
}

func NewOnGo() *OnGO {
	return &OnGO{
		Clients: make(map[string]Client),
		Mutex:   &sync.RWMutex{},

		Labels: make(map[string]string),

		LabelMutex: &sync.RWMutex{},
	}
}

//获取当前的client列表
func (b *OnGO) ClientList() []Client {
	keys := []Client{}

	b.Mutex.RLock()
	defer b.Mutex.RUnlock()

	for cID, v := range b.Clients {
		v.ClientID = cID
		label := b.GetLabel(cID)
		if label != "" {
			v.Addr = label
		}

		keys = append(keys, v)

	}

	return keys
}

//注册当前ws的对端
func (b *OnGO) Subscribe(id string, addr string) *Client {
	now := time.Now()
	sub := Client{
		Data:    make(chan *http.Request),
		Created: &now,
		Addr:    addr,
	}

	b.Mutex.Lock()
	b.Clients[id] = sub
	b.Mutex.Unlock()

	return &sub
}

//将请求放入对应的client
func (b *OnGO) Send(id string, req *http.Request) {
	var ok bool

	b.Mutex.RLock()
	defer b.Mutex.RUnlock()
	_, ok = b.Clients[id]

	if !ok {
		return
	}

	b.Clients[id].Data <- req
}

//对端断开链接
func (b *OnGO) Unsubscribe(id string) {

	b.Mutex.RLock()

	sub, ok := b.Clients[id]

	b.Mutex.RUnlock()

	if ok {
		close(sub.Data)

		b.Mutex.Lock()
		defer b.Mutex.Unlock()

		delete(b.Clients, id)
	}

	b.DelLabel(id)
}

//判断id是否存在
func (b *OnGO) Exist(id string) (ok bool) {

	b.Mutex.RLock()
	defer b.Mutex.RUnlock()
	_, ok = b.Clients[id]

	if !ok {
		return
	}

	return
}

//添加标签
func (b *OnGO) AddLabel(id string, label string) {

	b.LabelMutex.Lock()
	b.Labels[id] = label
	b.LabelMutex.Unlock()

	return
}

//删除标签
func (b *OnGO) DelLabel(id string) {

	b.LabelMutex.Lock()
	delete(b.Labels, id)
	b.LabelMutex.Unlock()

	return
}

//获取标签
func (b *OnGO) GetLabel(id string) string {

	var label string
	b.LabelMutex.RLock()
	label = b.Labels[id]
	b.LabelMutex.RUnlock()

	return label
}
