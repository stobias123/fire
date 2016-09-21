// Package fire implements a small and opinionated framework for Go providing
// Ember compatible JSON APIs.
package fire

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/labstack/echo"
	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/engine/standard"
	"gopkg.in/tomb.v2"
)

// Map is a general purpose map used for configuration.
type Map map[string]interface{}

// A Component that can be mounted in an application.
type Component interface {
	// Describe must return a ComponentInfo struct that describes the component.
	Describe() ComponentInfo
}

// A RoutableComponent is a component that accepts requests from a router for
// routes that haven been registered using Register().
type RoutableComponent interface {
	Component

	// Register will be called by the application with a new echo router.
	Register(router *echo.Echo)
}

// A BootableComponent is an extended component with additional methods for
// setup and teardown.
type BootableComponent interface {
	Component

	// Setup will be called before the applications starts and allows further
	// initialization.
	Setup() error

	// Teardown will be called after applications has stopped and allows proper
	// cleanup.
	Teardown() error
}

// An InspectorComponent is an extended component that is able to inspect the
// boot process of an application and inspect all used components and the router
// instance.
type InspectorComponent interface {
	Component

	BeforeRegister([]Component)
	BeforeSetup([]BootableComponent)
	BeforeRun(*echo.Echo)
	AfterRun()
	AfterTeardown()
}

// A ReporterComponent is an extended component that is responsible for
// reporting errors.
type ReporterComponent interface {
	Component

	Report(err error) error
}

// An Application provides a simple way to combine multiple components.
type Application struct {
	components []Component
	routables  []RoutableComponent
	bootables  []BootableComponent
	inspectors []InspectorComponent
	reporters  []ReporterComponent

	mutex  sync.Mutex
	server engine.Server
	tomb   tomb.Tomb
}

// New creates and returns a new Application.
func New() *Application {
	return &Application{}
}

// Mount will mount the passed Component in the application.
//
// Note: Each component should only be mounted once before calling Run or Start.
func (a *Application) Mount(component Component) {
	// synchronize access
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// check status
	if a.server != nil {
		panic("Application has already been started")
	}

	// check component
	if component == nil {
		panic("Mount must be called with a component")
	}

	// add routable component
	if c, ok := component.(RoutableComponent); ok {
		a.routables = append(a.routables, c)
	}

	// add bootable component
	if c, ok := component.(BootableComponent); ok {
		a.bootables = append(a.bootables, c)
	}

	// add inspector
	if c, ok := component.(InspectorComponent); ok {
		a.inspectors = append(a.inspectors, c)
	}

	// add reporter
	if c, ok := component.(ReporterComponent); ok {
		a.reporters = append(a.reporters, c)
	}

	a.components = append(a.components, component)
}

// Start will start the application using a new server listening on the
// specified address.
//
// See StartWith.
func (a *Application) Start(addr string) {
	a.StartWith(standard.New(addr))
}

// StartSecure will start the application with a new server listening on the
// specified address using the provided TLS certificate.
//
// See StartWith.
func (a *Application) StartSecure(addr, certFile, keyFile string) {
	a.StartWith(standard.WithTLS(addr, certFile, keyFile))
}

// StartWith will start the application using the specified server.
//
// Note: Any errors that occur during the boot process of the application and
// later during request processing are reported using the registered reporters.
// If there are no reporters or one of the reporter fails to report the error,
// the calling goroutine will panic and print the error (see Exec).
func (a *Application) StartWith(server engine.Server) {
	// synchronize access
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// check status
	if a.server != nil {
		panic("Application has already been started")
	}

	// check server
	if server == nil {
		panic("StartWith must be called with a server")
	}

	// set server
	a.server = server

	// run app
	a.tomb.Go(a.runner)
}

// Exec will execute the passed function in the context of the application
// and call all reporters if an error occurs.
//
// Note: If a reporter fails to report an occurring error, the current goroutine
// will panic and print the original error and the reporter's error.
func (a *Application) Exec(fn func() error) {
	err := fn()
	if err != nil {
		a.report(err)
	}
}

// Stop will stop a running application and wait until it has been properly stopped.
func (a *Application) Stop() {
	// synchronize access
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// kill controlling tomb
	a.tomb.Kill(nil)

	// stop app by stopping the server
	a.server.Stop()

	// wait until goroutine finishes
	a.tomb.Wait()
}

// Yield will block the calling goroutine until the the application has been
// stopped. It will automatically stop the application if the process receives
// the SIGINT signal.
func (a *Application) Yield() {
	// prepare signal pipeline
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	select {
	// wait for interrupt and stop app
	case <-interrupt:
		a.Stop()
	// wait for app to close and return
	case <-a.tomb.Dead():
		return
	}
}

func (a *Application) runner() error {
	a.Exec(a.boot)
	return nil
}

func (a *Application) boot() error {
	// create new router
	router := echo.New()

	// set error handler
	router.SetHTTPErrorHandler(a.errorHandler)

	// signal before register event
	for _, i := range a.inspectors {
		i.BeforeRegister(a.components)
	}

	// TODO: Create group and pass group?

	// register routable components
	for _, c := range a.routables {
		c.Register(router)
	}

	// signal before setup event
	for _, i := range a.inspectors {
		i.BeforeSetup(a.bootables)
	}

	// setup bootable components
	for _, c := range a.bootables {
		err := c.Setup()
		if err != nil {
			return err
		}
	}

	// signal before run event
	for _, i := range a.inspectors {
		i.BeforeRun(router)
	}

	// run router
	err := router.Run(a.server)
	if err != nil {
		select {
		case <-a.tomb.Dying():
			// Stop() has been called and therefore the error returned by run
			// can be ignored as it is always the underlying listener failing
		default:
			return err
		}
	}

	// signal after run event
	for _, i := range a.inspectors {
		i.AfterRun()
	}

	// teardown bootable components
	for _, c := range a.bootables {
		err := c.Teardown()
		if err != nil {
			return err
		}
	}

	// signal after teardown event
	for _, i := range a.inspectors {
		i.AfterTeardown()
	}

	return nil
}

func (a *Application) errorHandler(err error, ctx echo.Context) {
	// treat echo.HTTPError instances as already treated errors
	if he, ok := err.(*echo.HTTPError); ok && http.StatusText(he.Code) != "" {
		// write response if not yet committed
		if !ctx.Response().Committed() {
			ctx.NoContent(he.Code)
		}

		return
	}

	// report error
	a.report(err)

	// write response if not yet committed
	if !ctx.Response().Committed() {
		ctx.NoContent(http.StatusInternalServerError)
	}
}

func (a *Application) report(err error) {
	// prepare variable that tracks if the error has at least been reported once
	var reportedOnce bool

	// iterate over all reporters
	for _, r := range a.reporters {
		// attempt to report error
		rErr := r.Report(err)
		if rErr != nil {
			name := r.Describe().Name
			panic(fmt.Sprintf("%s returned '%s' while reporting '%s'", name, rErr, err))
		}

		// mark report
		reportedOnce = true
	}

	// check tracker
	if !reportedOnce {
		panic(fmt.Sprintf("No reporter found to report '%s'", err))
	}
}
