package util

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type InletError struct {
	Err error  `json:"-"`
	Msg string `json:"msg"`
	Ret int    `json:"status"`
}

func (err InletError) Error() string {
	// return err.error.Error() + err.Msg
	return fmt.Sprintf("err:%v,msg:%v,status:%v", err.Err, err.Msg, err.Ret)
}

func NewInletError(ret int, msg string, err error) InletError {
	return InletError{Ret: ret, Msg: msg, Err: err}
}

func HandleResponse(w http.ResponseWriter, errRes InletError, resInfo interface{}) {

	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	if errRes.Ret != 0 {
		json.NewEncoder(w).Encode(errRes)
	} else {

		json.NewEncoder(w).Encode(resInfo)
	}

}
