package fire

import (
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/labstack/echo"
)

// The InspectableComponent interface can be implement by a component in order to
// be inspectable by a inspector.
type InspectableComponent interface {
	// Inspect will be called by the application to print a list of used
	// components and their configuration.
	Inspect() string
}

// An Inspector can be used during development to print the applications
// component stack, the route table and log requests to writer.
type Inspector struct {
	Writer      io.Writer
	Application *Application
}

// DefaultInspector creates and returns a new inspector that writes to stdout.
func DefaultInspector(app *Application) *Inspector {
	return NewInspector(app, os.Stdout)
}

// NewInspector creates and returns a new inspector.
func NewInspector(app *Application, writer io.Writer) *Inspector {
	return &Inspector{
		Application: app,
		Writer:      writer,
	}
}

// Register implements the Component interface.
func (i *Inspector) Register(router *echo.Echo) {
	router.Use(i.requestLogger)
	router.SetHTTPErrorHandler(i.errorHandler)
}

// Setup implements the BootableComponent interface.
func (i *Inspector) Setup() error {
	// print header
	fmt.Fprintln(i.Writer, "==> Fire application starting...")

	// print component info
	fmt.Fprintln(i.Writer, "==> Mounted components:")
	i.inspectComponents()

	// print routing table
	fmt.Fprintln(i.Writer, "==> Registered routes:")
	i.inspectRoutingTable()

	// print footer
	fmt.Fprintln(i.Writer, "==> Ready to go!")

	return nil
}

// Teardown implements the BootableComponent interface.
func (i *Inspector) Teardown() error {
	// print footer
	fmt.Fprintln(i.Writer, "==> Fire application is stopping...")

	return nil
}

func (i *Inspector) inspectComponents() {
	// inspect all components
	for _, component := range i.Application.components {
		if inspectable, ok := component.(InspectableComponent); ok {
			fmt.Fprintf(i.Writer, inspectable.Inspect())
		}
	}
}

func (i *Inspector) inspectRoutingTable() {
	// prepare routes
	var routes []string

	// add all routes as string
	for _, route := range i.Application.router.Routes() {
		routes = append(routes, fmt.Sprintf("%6s  %-30s", route.Method, route.Path))
	}

	// sort routes
	sort.Strings(routes)

	// print routes
	for _, route := range routes {
		fmt.Fprintln(i.Writer, route)
	}
}

func (i *Inspector) requestLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()
		res := c.Response()

		// save start
		start := time.Now()

		// call next handler
		if err := next(c); err != nil {
			c.Error(err)
		}

		// get request duration
		duration := time.Since(start).String()

		// log request
		fmt.Fprintf(i.Writer, "%6s  %-30s  %d  %s\n", req.Method(), req.URL().Path(), res.Status(), duration)

		return nil
	}
}

func (i *Inspector) errorHandler(err error, ctx echo.Context) {
	fmt.Fprintf(i.Writer, "   ERR  \"%s\"\n", err)
}
