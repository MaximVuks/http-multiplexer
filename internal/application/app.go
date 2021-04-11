package application

import (
	"context"
	"flag"
	"github.com/maximvuks/http-multiplexer/internal/server"
	"github.com/maximvuks/http-multiplexer/internal/server/http_server"
	"github.com/maximvuks/http-multiplexer/internal/url_processor"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Application struct {
	wg         *sync.WaitGroup
	ctx        context.Context
	ctxCancel  context.CancelFunc
	logger     *log.Logger
	errorChan  chan error
	httpServer server.Server
}

func New() *Application {
	httpServerPort := flag.Int("port", 6000, "port for http server")
	httpMaxConnections := flag.Int("maxConnections", 100, "max connections for http server")
	maxConcurrentUrlProcess := flag.Int("maxConcurrentUrlProcess", 4, "max concurrent url process")
	urlConnectionTimeout := flag.Int("urlConnectionTimeout", 1, "url connection timeout in seconds")
	flag.Parse()

	wg := new(sync.WaitGroup)
	ctx, ctxCancel := context.WithCancel(context.Background())
	logger := log.New(os.Stdout, "", log.LstdFlags)
	errorChan := make(chan error, math.MaxUint8)
	client := &http.Client{}

	urlProcessor := url_processor.NewUrlProcessor(
		wg,
		logger,
		client,
		*maxConcurrentUrlProcess,
		time.Duration(*urlConnectionTimeout)*time.Second,
	)

	processUrlHandler := http_server.NewProcessUrlHandler(urlProcessor)

	httpServer := http_server.New(
		ctx,
		*httpServerPort,
		logger,
		errorChan,
		*httpMaxConnections,
		map[string]http.HandlerFunc{
			"/processUrl": processUrlHandler.ProcessUrl,
		},
	)

	return &Application{
		wg:         wg,
		ctx:        ctx,
		ctxCancel:  ctxCancel,
		logger:     logger,
		errorChan:  errorChan,
		httpServer: httpServer,
	}
}

func (app *Application) Run() {
	defer app.Stop()

	app.httpServer.Run()
	app.logger.Println("application started")

	select {
	case err := <-app.errorChan:
		app.logger.Panicf("service crashed: %s", err.Error())
	case <-app.ctx.Done():
		app.logger.Println("service stop by context")
	case sig := <-waitExitSignal():
		app.logger.Printf("signal for stop app: %s", sig.String())
	}
}

func (app *Application) Stop() {
	app.logger.Println("stopping service")

	app.ctxCancel()
	httpServerErr := app.httpServer.Stop()

	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Second*5)

	go func() {
		defer ctxCancel()
		app.wg.Wait()
	}()

	<-ctx.Done()

	if httpServerErr != nil {
		app.logger.Panic("http server stopped incorrectly")
	}

	if ctx.Err() != context.Canceled {
		app.logger.Panic("service stopped with timeout")
	}

	app.logger.Println("service successfully stopped")
}

func waitExitSignal() chan os.Signal {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	return sigs
}
