package logx

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type Event struct {
	Ts    time.Time `json:"ts"`
	Level string    `json:"level"`
	Svc   string    `json:"service"`
	Msg   string    `json:"msg"`
	Err   string    `json:"err,omitempty"`
	Extra any       `json:"extra,omitempty"`
}

var (
	enc = json.NewEncoder(os.Stdout)
	mu  sync.Mutex
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
	}
	log(ev)
}
