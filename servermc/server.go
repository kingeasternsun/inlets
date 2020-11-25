//支持多个客户端

package servermc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kingeastern/inlets/util"

	"github.com/gorilla/websocket"
	"github.com/kingeastern/inlets/transport"
	"github.com/kingeastern/inlets/types"
	"github.com/twinj/uuid"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 6 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Server for the exit-node of inlets
type Server struct {
	GatewayTimeout  time.Duration //server下发到下游的client后，等待响应的超时时间
	CollectInterval time.Duration
	Port            int
	Token           string
}

//GarbageCollectBus 对于一些上游的http请求，经有server转发到client后，可能由于一些错误或异常没有及时处理，超时了，就要进行清理，避免map过大
func GarbageCollectBus(bus *types.Bus, interval time.Duration, expiry time.Duration) {
	ticker := time.NewTicker(interval)
	select {
	case <-ticker.C:
		list := bus.SubscriptionList()
		for _, item := range list {
			if bus.Expired(item, expiry) {
				bus.Unsubscribe(item)
			}
		}
		break
	}
}

func handleErrCient(w http.ResponseWriter) {

	var resInfo struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}

	resInfo.Status = -1
	resInfo.Msg = "x-clients-id 不存在或链接已经断开"

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(resInfo)
}

// ProxyHandler 上游用户的每一条 http请求，经由 ProxyHandler,根据请求头中的clientID转发相应的OnGo中，
func ProxyHandler(ongo *types.OnGO, bus *types.Bus, gatewayTimeout time.Duration) func(w http.ResponseWriter, r *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		// log.SetReportCaller(true)

		//获取clientID。上游用户首先通过当前server的http接口获取到下游可用的client列表，然后选择要请求命令到哪个client
		clientID := r.Header.Get(transport.ClientsHeader)
		if clientID == "" {
			handleErrCient(w)
			return
		}

		if ok := ongo.Exist(clientID); ok == false {
			handleErrCient(w)
			return
		}

		//生成每次request的唯一ID
		inletsID := uuid.Formatter(uuid.NewV4(), uuid.FormatHex)

		sub := bus.Subscribe(inletsID)

		defer func() {
			bus.Unsubscribe(inletsID)
		}()

		util.Debugf("[%s] proxy %s %s %s", inletsID, r.Host, r.Method, r.URL.String())
		r.Header.Set(transport.InletsHeader, inletsID)

		if r.Body != nil {
			defer r.Body.Close()
		}

		body, _ := ioutil.ReadAll(r.Body)

		qs := ""
		if len(r.URL.RawQuery) > 0 {
			qs = "?" + r.URL.RawQuery
		}

		req, _ := http.NewRequest(r.Method, fmt.Sprintf("http://%s%s%s", r.Host, r.URL.Path, qs),
			bytes.NewReader(body))

		transport.CopyHeaders(req.Header, &r.Header)

		wg := sync.WaitGroup{}
		wg.Add(2)

		//将用户的http请求根据 clientID 放入到对应的数据通道中，等待 ws连接上的goroutine读取下发到下游的client
		go func() {

			clientID := req.Header.Get(transport.ClientsHeader)
			ongo.Send(clientID, req)
			// outgoing <- req
			wg.Done()
		}()

		//等待接收下游的client的响应：下游client的相应经过ws连接发送上来后，经有ws连接上的goroutine读取后放入到 inletsID 对应的数据通道上
		go func() {
			util.Debugf("[%s] waiting for response", inletsID)

			select {
			case res, ok := <-sub.Data:

				util.Debugf("%+v", res)
				if res == nil {

					util.Debug("ok", ok)
					//TODO
					// w.WriteHeader(http.StatusBadGateway)
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"msg":"请求出错"}`))
				} else {

					innerBody, _ := ioutil.ReadAll(res.Body)

					transport.CopyHeaders(w.Header(), &res.Header)
					w.WriteHeader(res.StatusCode)
					w.Write(innerBody)
					util.Debugf("[%s] wrote %d bytes", inletsID, len(innerBody))

				}

				wg.Done()
				break
			case <-time.After(gatewayTimeout):
				util.Debugf("[%s] timeout after %f secs\n", inletsID, gatewayTimeout.Seconds())

				w.WriteHeader(http.StatusGatewayTimeout)
				wg.Done()
				break
			}
		}()

		wg.Wait()
	}
}

// ServerWs 下游client通过该接口 注册到当前server中，维护ws长连接和相应的两个goroutine，
// 一个goroutine读取当前连接对应的channel上的数据然后发往下游，一个gotine读取ws上下游发送过来的二进制响应数据，根据 inletID 发送对应的channel
func ServerWs(ongo *types.OnGO, bus *types.Bus, token string) func(w http.ResponseWriter, r *http.Request) {

	// log.SetReportCaller(true)

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := authorized(token, r)

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
		}

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if _, ok := err.(websocket.HandshakeError); !ok {
				util.Error(err)
			}
			return
		}

		// util.Debug("ws.RemoteAddr().Network()", ws.RemoteAddr().Network())
		util.Debug("ws.RemoteAddr().add()", ws.RemoteAddr().String())
		// util.Debugf("Connecting websocket on %s:", ws.RemoteAddr())

		clientID := uuid.Formatter(uuid.NewV4(), uuid.FormatHex)
		util.Debug("clientID:", clientID)

		//下游的server连接过来，作为websocket的client端。 当前websocket维护不同的wsClient端和数据接收channel的映射关系
		client := ongo.Subscribe(clientID, ws.RemoteAddr().String())

		defer func() {
			ongo.Unsubscribe(clientID)
		}()

		/*
			pongs := make(chan string, 1)
			ws.SetReadDeadline(time.Now().Add(pongWait))
			ws.SetPingHandler(func(m string) error {
				// ws.SetReadDeadline(time.Now().Add(pongWait));
				util.Debug(" receive pong")
				select {
				case pongs <- m:
				default:
				}

				return nil
			})

		*/
		// connectionDone := make(chan struct{})

		// cLock := sync.Mutex{}
		wg := sync.WaitGroup{}
		wg.Add(2)

		go func() {
			// wg.Add(1)

			defer func() {

				// cLock.Lock()
				// _, ok := <-connectionDone
				// if ok {
				// 	cLock.Unlock()
				// } else {
				// 	close(connectionDone)
				// 	cLock.Unlock()
				// }
				wg.Done()

			}()

			for {

				// select {
				// case m := <-pongs:
				// 	util.Debug("to send pong")
				// 	if err := ws.WriteControl(websocket.PongMessage, []byte(m), time.Now().Add(pongWait)); err != nil {
				// 		util.Error("pong:", err)
				// 	}
				// default:
				// }

				// ws.SetReadDeadline(time.Now().Add(pongWait))
				msgType, message, err := ws.ReadMessage()
				if err != nil {

					ongo.Unsubscribe(clientID)
					util.Error("read:", err)
					return
				}

				if msgType == websocket.TextMessage {
					util.Debugf("TextMessage: %s", message)
					ongo.AddLabel(clientID, string(message))
					a := struct {
						Host string
						Age  int
					}{
						"hostback",
						4,
					}

					ws.WriteJSON(a)
				} else if msgType == websocket.BinaryMessage {

					reader := bytes.NewReader(message)
					scanner := bufio.NewReader(reader)
					res, _ := http.ReadResponse(scanner, nil)

					if id := res.Header.Get(transport.InletsHeader); len(id) > 0 {
						bus.Send(id, res)
					}
				}
			}
		}()

		go func() {

			ticker := time.NewTicker(pingPeriod)
			// wg.Add(1)
			defer func() {

				// cLock.Lock()
				// _, ok := <-connectionDone
				// if ok {
				// 	cLock.Unlock()
				// } else {
				// 	close(connectionDone)
				// 	cLock.Unlock()
				// }
				wg.Done()
				ticker.Stop()
			}()

			for {
				util.Debugf("wait for request %+v", clientID)

				select {
				case outboundRequest := <-client.Data:

					if outboundRequest == nil {
						// util.Debug("ok", ok)
						util.Error("close")
						return
						// continue
					}

					util.Debugf("[%s] request written to websocket", outboundRequest.Header.Get(transport.InletsHeader))

					buf := new(bytes.Buffer)

					outboundRequest.Write(buf)

					ws.SetWriteDeadline(time.Now().Add(writeWait))
					err = ws.WriteMessage(websocket.BinaryMessage, buf.Bytes())
					if err != nil {
						util.Error("wrtte:", err)
						return
					}

				case <-ticker.C:

					// ws.SetWriteDeadline(time.Now().Add(writeWait))
					if err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(writeWait)); err != nil {
						util.Error("WriteControl:", err)
						return
					}

				}
			}

		}()

		// <-connectionDone
		wg.Wait()

		util.Debug("client closed,clientID:", clientID)
	}
}

func authorized(token string, r *http.Request) error {

	auth := r.Header.Get("Authorization")
	valid := false
	if len(token) == 0 {
		valid = true
	} else {
		prefix := "Bearer "
		if strings.HasPrefix(auth, prefix); len(auth) > len(prefix) && auth[len(prefix):] == token {
			valid = true
		}
	}

	if !valid {
		return fmt.Errorf("send token in header Authorization: Bearer <token>")
	}

	return nil
}
