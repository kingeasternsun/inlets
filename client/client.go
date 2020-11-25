package client

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kingeastern/inlets/util"

	// log "github.com/sirupsen/logrus"

	"github.com/gorilla/websocket"
	"github.com/kingeastern/inlets/transport"
	"github.com/pkg/errors"
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

var httpClient *http.Client

// Client for inlets
type Client struct {
	// Remote site for websocket address
	Remote string

	// Map of upstream servers dns.entry=http://ip:port
	UpstreamMap map[string]string

	// Token for authentication
	Token string

	// PingWaitDuration duration to wait between pings
	// PingWaitDuration time.Duration
}

// Connect connect and serve traffic through websocket
func (c *Client) Connect() error {
	// log.SetReportCaller(true)

	httpClient = http.DefaultClient
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	remote := c.Remote
	if !strings.HasPrefix(remote, "ws") {
		remote = "ws://" + remote
	}

	remoteURL, urlErr := url.Parse(remote)
	if urlErr != nil {
		return errors.Wrap(urlErr, "bad remote URL")
	}

	u := url.URL{Scheme: remoteURL.Scheme, Host: remoteURL.Host, Path: remoteURL.Path}

	util.Debugf("connecting to %s", u.String())

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), http.Header{
		"Authorization": []string{"Bearer " + c.Token},
	})

	if err != nil {
		return err
	}

	util.Debugf("Connected to websocket: %s", ws.LocalAddr())

	defer ws.Close()

	/*
		http://www.gorillatoolkit.org/pkg/websocket
		Connections support one concurrent reader and one concurrent writer.

		Applications are responsible for ensuring that no more than one goroutine calls the write methods (NextWriter, SetWriteDeadline, WriteMessage, WriteJSON, EnableWriteCompression, SetCompressionLevel) concurrently and that no more than one goroutine calls the read methods (NextReader, SetReadDeadline, ReadMessage, ReadJSON, SetPongHandler, SetPingHandler) concurrently.

		The Close and WriteControl methods can be called concurrently with all other methods.

	*/

	// Send pings
	// tickerDone := make(chan bool)
	// go func() {
	// 	log.Printf("Writing pings")

	// 	ticker := time.NewTicker((c.PingWaitDuration * 9) / 10) // send on a period which is around 9/10ths of original value
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			if err := ws.Ping(); err != nil {
	// 				close(tickerDone)
	// 			}
	// 			break
	// 		case <-tickerDone:
	// 			log.Printf("tickerDone, no more pings will be sent from client\n")
	// 			return
	// 		}
	// 	}
	// }()

	// pongs := make(chan string, 1)
	// ws.SetReadDeadline(time.Now().Add(pongWait))
	// // ws.SetPongHandler(func(m string) error {
	// ws.SetPingHandler(func(m string) error {

	// 	util.Debug("receive ping")
	// 	select {
	// 	case pongs <- m:
	// 	default:
	// 	}

	// 	return nil
	// })

	//add by wdy, add label to controller
	label, terr := util.GetHostLabel()
	if terr != nil {
		util.Error(err)
	} else {
		util.Debug(label)
		ws.WriteMessage(websocket.TextMessage, []byte(label))
	}

	// Work with websocket
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {

			messageType, message, err := ws.ReadMessage()
			if err != nil {
				util.Error("read:", err)
				return
			}

			//不需要处理ping ，因为默认的handler就会回复pong
			switch messageType {
			case websocket.TextMessage:
				util.Debugf("TextMessage: %s\n", message)

			case websocket.BinaryMessage:
				// proxyToUpstream

				buf := bytes.NewBuffer(message)
				bufReader := bufio.NewReader(buf)
				req, readReqErr := http.ReadRequest(bufReader)
				if readReqErr != nil {
					util.Debug(readReqErr)
					return
				}

				inletsID := req.Header.Get(transport.InletsHeader)
				// util.Debugf("[%s] recv: %d", requestID, len(message))

				util.Debugf("[%s] %s", inletsID, req.RequestURI)

				body, _ := ioutil.ReadAll(req.Body)

				proxyHost := ""
				if val, ok := c.UpstreamMap[req.Host]; ok {
					proxyHost = val
				} else if val, ok := c.UpstreamMap[""]; ok {
					proxyHost = val
				}

				requestURI := fmt.Sprintf("%s%s", proxyHost, req.URL.String())
				if len(req.URL.RawQuery) > 0 {
					requestURI = requestURI + "?" + req.URL.RawQuery
				}

				util.Debugf("[%s] proxy => %s", inletsID, requestURI)

				newReq, newReqErr := http.NewRequest(req.Method, requestURI, bytes.NewReader(body))
				if newReqErr != nil {
					util.Debugf("[%s] newReqErr: %s", inletsID, newReqErr.Error())
					return
				}

				transport.CopyHeaders(newReq.Header, &req.Header)

				res, resErr := httpClient.Do(newReq)

				if resErr != nil {
					util.Debugf("[%s] Upstream tunnel err: %s", inletsID, resErr.Error())

					errRes := http.Response{
						StatusCode: http.StatusBadGateway,
						Body:       ioutil.NopCloser(strings.NewReader(resErr.Error())),
						Header:     http.Header{},
					}

					errRes.Header.Set(transport.InletsHeader, inletsID)
					buf2 := new(bytes.Buffer)
					errRes.Write(buf2)
					if errRes.Body != nil {
						errRes.Body.Close()
					}

					ws.SetWriteDeadline(time.Now().Add(writeWait))
					err = ws.WriteMessage(websocket.BinaryMessage, buf2.Bytes())
					if err != nil {
						util.Error("write:", err)
						return
					}

				} else {
					util.Debugf("[%s] tunnel res.Status => %s", inletsID, res.Status)

					buf2 := new(bytes.Buffer)
					res.Header.Set(transport.InletsHeader, inletsID)

					res.Write(buf2)
					if res.Body != nil {
						res.Body.Close()
					}

					util.Debugf("[%s] %s bytes", inletsID, buf2)

					ws.SetWriteDeadline(time.Now().Add(writeWait))
					err = ws.WriteMessage(websocket.BinaryMessage, buf2.Bytes())
					if err != nil {
						util.Error("write:", err)
						return
					}
				}
			}

		}
	}()

	<-done

	return errors.New("done")
}
