package logx

import (
	"encoding/json"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Event struct {
	Ts    time.Time `json:"ts"`
	Level string    `json:"level"`
	Svc   string    `json:"service"`
	Msg   string    `json:"msg"`
	Err   string    `json:"err,omitempty"`
	Stack string    `json:"stack,omitempty"`
	Extra any       `json:"extra,omitempty"`
}

var (
	enc = json.NewEncoder(os.Stdout)
	mu  sync.Mutex
	dbg = strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug")
)

func log(ev Event) {
	mu.Lock()
	defer mu.Unlock()
	_ = enc.Encode(ev)
}

func Info(service, msg string, extra any) {
	log(Event{Ts: time.Now().UTC(), Level: "info", Svc: service, Msg: msg, Extra: extra})
}

func Error(service, msg string, err error, extra any) {
	ev := Event{Ts: time.Now().UTC(), Level: "error", Svc: service, Msg: msg, Extra: extra}
	if err != nil {
		ev.Err = err.Error()
		if dbg {
			ev.Stack = string(debug.Stack())
		}
	}
	log(ev)
}
