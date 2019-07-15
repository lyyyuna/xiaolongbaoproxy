package goproxy

import (
	"fmt"
	"net/http"
	"os"
	"github.com/google/logger"
)

func StartProxy(port string, logpath string) {

	lf, err := os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		fmt.Println("Fail to open the log file")
		return
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
	go server.ListenAndServe()
}

