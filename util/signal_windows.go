package util

import (
	"time"
)

//重置ticker
func ResetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

const timeout = 10

func SignalHandle() {

	return
}

//SignalHandleWithOtherChan 收到信号的时候，往指定的通道里面发送消息
func SignalHandleWithOtherChan() {

	return
}
