package util

import (
	"os"
	"os/signal"
	"syscall"
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

	t1 := time.NewTimer(timeout * time.Hour)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR1, syscall.SIGUSR2)
	for {
		select {
		case sig := <-ch:
			switch sig {
			case syscall.SIGUSR1:
				SetLogLevel("DEBUG")
				ResetTimer(t1, timeout*time.Hour)
			case syscall.SIGUSR2:
				SetLogLevel("ERROR")
				ResetTimer(t1, timeout*time.Hour)
			default:
			}
		case <-t1.C:
			SetLogLevel("ERROR")
			ResetTimer(t1, timeout*time.Hour)
		}
	}
}

//SignalHandleWithOtherChan 收到信号的时候，往指定的通道里面发送消息
func SignalHandleWithOtherChan(useChan chan<- struct{}) {

	t1 := time.NewTimer(timeout * time.Hour)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGUSR1, syscall.SIGUSR2)
	for {
		select {
		case sig := <-ch:
			switch sig {
			case syscall.SIGUSR1:

				SetLogLevel("DEBUG")
				ResetTimer(t1, timeout*time.Hour)

				select {
				case useChan <- struct{}{}:
				default:
				}

			case syscall.SIGUSR2:

				SetLogLevel("ERROR")
				ResetTimer(t1, timeout*time.Hour)

				select {
				case useChan <- struct{}{}:
				default:
				}
				// t1.Stop()
			default:
			}
		case <-t1.C:
			SetLogLevel("ERROR")
			ResetTimer(t1, timeout*time.Hour)
		}
	}
}
