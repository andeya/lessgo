package grace

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/lessgo/lessgo/logs"
)

// Server embedded fasthttp.Server
type Server struct {
	*fasthttp.Server
	logger           logs.Logger
	GraceListener    net.Listener
	SignalHooks      map[int]map[os.Signal][]func()
	tlsInnerListener *graceListener
	wg               sync.WaitGroup
	sigChan          chan os.Signal
	isChild          bool
	state            uint8
	Network          string
	Addr             string
}

// Serve accepts incoming connections on the Listener l,
// creating a new service goroutine for each.
// The service goroutines read requests and then call srv.Handler to reply to them.
func (srv *Server) Serve() (err error) {
	srv.state = StateRunning
	err = srv.Server.Serve(srv.GraceListener)
	srv.logger.Sys("%v Waiting for connections to finish...", syscall.Getpid())
	srv.wg.Wait()
	srv.state = StateTerminate
	return
}

// ListenAndServe listens on the TCP network address srv.Addr and then calls Serve
// to handle requests on incoming connections. If srv.Addr is blank, ":http" is
// used.
func (srv *Server) ListenAndServe() (err error) {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}

	go srv.handleSignals()

	l, err := srv.getListener(addr)
	if err != nil {
		srv.logger.Sys("%v", err)
		return err
	}

	srv.GraceListener = newGraceListener(l, srv)

	if srv.isChild {
		process, err := os.FindProcess(os.Getppid())
		if err != nil {
			srv.logger.Sys("%v", err)
			return err
		}
		err = process.Kill()
		if err != nil {
			return err
		}
	}

	srv.logger.Sys("%v %v", os.Getpid(), srv.Addr)
	return srv.Serve()
}

// ListenAndServeTLS listens on the TCP network address srv.Addr and then calls
// Serve to handle requests on incoming TLS connections.
//
// Filenames containing a certificate and matching private key for the server must
// be provided. If the certificate is signed by a certificate authority, the
// certFile should be the concatenation of the server's certificate followed by the
// CA's certificate.
//
// If srv.Addr is blank, ":https" is used.
func (srv *Server) ListenAndServeTLS(certFile, keyFile string) (err error) {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}

	go srv.handleSignals()

	l, err := srv.getListener(addr)
	if err != nil {
		srv.logger.Sys("%v", err)
		return err
	}

	srv.tlsInnerListener = newGraceListener(l, srv)
	srv.GraceListener = tls.NewListener(srv.tlsInnerListener, &tls.Config{})

	if srv.isChild {
		process, err := os.FindProcess(os.Getppid())
		if err != nil {
			srv.logger.Sys("%v", err)
			return err
		}
		err = process.Kill()
		if err != nil {
			return err
		}
	}
	srv.logger.Sys("%v %v", os.Getpid(), srv.Addr)
	return srv.ServeTLS(l, certFile, keyFile)
}

// getListener either opens a new socket to listen on, or takes the acceptor socket
// it got passed when restarted.
func (srv *Server) getListener(laddr string) (l net.Listener, err error) {
	if srv.isChild {
		var ptrOffset uint
		if len(socketPtrOffsetMap) > 0 {
			ptrOffset = socketPtrOffsetMap[laddr]
			srv.logger.Sys("laddr %v ptr offset %v", laddr, socketPtrOffsetMap[laddr])
		}

		f := os.NewFile(uintptr(3+ptrOffset), "")
		l, err = net.FileListener(f)
		if err != nil {
			err = fmt.Errorf("net.FileListener error: %v", err)
			return
		}
	} else {
		l, err = net.Listen(srv.Network, laddr)
		if err != nil {
			err = fmt.Errorf("net.Listen error: %v", err)
			return
		}
	}
	return
}

// handleSignals listens for os Signals and calls any hooked in function that the
// user had registered with the signal.
func (srv *Server) handleSignals() {
	var sig os.Signal

	signal.Notify(
		srv.sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	pid := syscall.Getpid()
	for {
		sig = <-srv.sigChan
		srv.signalHooks(PreSignal, sig)
		switch sig {
		case syscall.SIGHUP:
			srv.logger.Sys("%v Received SIGHUP. forking.", pid)
			err := srv.fork()
			if err != nil {
				srv.logger.Sys("Fork err: %v", err)
			}
		case syscall.SIGINT:
			srv.logger.Sys("%v Received SIGINT.", pid)
			srv.shutdown()
		case syscall.SIGTERM:
			srv.logger.Sys("%v Received SIGTERM.", pid)
			srv.shutdown()
		default:
			srv.logger.Sys("Received %v: nothing i care about...", sig)
		}
		srv.signalHooks(PostSignal, sig)
	}
}

func (srv *Server) signalHooks(ppFlag int, sig os.Signal) {
	if _, notSet := srv.SignalHooks[ppFlag][sig]; !notSet {
		return
	}
	for _, f := range srv.SignalHooks[ppFlag][sig] {
		f()
	}
	return
}

// shutdown closes the listener so that no new connections are accepted. it also
// starts a goroutine that will serverTimeout (stop all running requests) the server
// after DefaultTimeout.
func (srv *Server) shutdown() {
	if srv.state != StateRunning {
		return
	}

	srv.state = StateShuttingDown
	if DefaultTimeout >= 0 {
		go srv.serverTimeout(DefaultTimeout)
	}
	err := srv.GraceListener.Close()
	if err != nil {
		srv.logger.Sys("%v Listener.Close() error: %v", syscall.Getpid(), err)
	} else {
		srv.logger.Sys("%v %v Listener closed.", syscall.Getpid(), srv.GraceListener.Addr())
	}
}

// serverTimeout forces the server to shutdown in a given timeout - whether it
// finished outstanding requests or not. if Read/WriteTimeout are not set or the
// max header size is very big a connection could hang
func (srv *Server) serverTimeout(d time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			srv.logger.Sys("WaitGroup at 0 %v", r)
		}
	}()
	if srv.state != StateShuttingDown {
		return
	}
	time.Sleep(d)
	srv.logger.Sys("[STOP - Hammer Time] Forcefully shutting down parent")
	for {
		if srv.state == StateTerminate {
			break
		}
		srv.wg.Done()
	}
}

func (srv *Server) fork() (err error) {
	regLock.Lock()
	defer regLock.Unlock()
	if runningServersForked {
		return
	}
	runningServersForked = true

	var files = make([]*os.File, len(runningServers))
	var orderArgs = make([]string, len(runningServers))
	for _, srvPtr := range runningServers {
		switch srvPtr.GraceListener.(type) {
		case *graceListener:
			files[socketPtrOffsetMap[srvPtr.Addr]] = srvPtr.GraceListener.(*graceListener).File()
		default:
			files[socketPtrOffsetMap[srvPtr.Addr]] = srvPtr.tlsInnerListener.File()
		}
		orderArgs[socketPtrOffsetMap[srvPtr.Addr]] = srvPtr.Addr
	}

	srv.logger.Sys("%v", files)
	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-graceful" {
				break
			}
			args = append(args, arg)
		}
	}
	args = append(args, "-graceful")
	if len(runningServers) > 1 {
		args = append(args, fmt.Sprintf(`-socketorder=%s`, strings.Join(orderArgs, ",")))
		srv.logger.Sys("%v", args)
	}
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = files
	err = cmd.Start()
	if err != nil {
		srv.logger.Fatal("Restart: Failed to launch, error: %v", err)
	}

	return
}
