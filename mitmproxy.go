package goproxy

import (
	"github.com/google/logger"
	"net/http"
	"os"
)

func StartProxy(port string, logpath string) *http.Server{

	lf, err := os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		panic("Fail to open the log file")
	}
	defer lf.Close()
	logg := logger.Init("LoggerExample", false, true, lf)
	defer logg.Close()

	handler := &WrappedHandler{logg}
	server := &http.Server{
		Addr: ":" + port,
		Handler: handler,
	}
	logger.Info("Proxy server starting..")
	// go server.ListenAndServe()
	return server
}

