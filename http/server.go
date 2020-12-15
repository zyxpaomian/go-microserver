package http

import (
	"github.com/gorilla/mux"
	log "microserver/common/formatlog"
	"net/http"
	"reflect"
	"runtime"
)

type WWWMux struct {
	r *mux.Router
}

func New() *WWWMux {
	return &WWWMux{r: mux.NewRouter()}
}

func (m *WWWMux) GetRouter() *mux.Router {
	return m.r
}

// 记录日志
func AccessLogHandler(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infof("[http] %s - %s", r.Method, r.RequestURI)
		h(w, r)
	}
}

// 注册URL映射
func (m *WWWMux) RegistURLMapping(path string, method string, handle func(http.ResponseWriter, *http.Request)) {
	log.Infof("[http] URL注册映射, path: %v, method: %v, handle: %v", path, method, runtime.FuncForPC(reflect.ValueOf(handle).Pointer()).Name())
	handle = AccessLogHandler(handle)
	m.r.HandleFunc(path, handle).Methods(method)
}
