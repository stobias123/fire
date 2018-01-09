package fire

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/256dpi/fire/coal"

	"github.com/256dpi/jsonapi"
	"gopkg.in/mgo.v2/bson"
)

// A Tester provides facilities to the test a fire API.
type Tester struct {
	// The store to use for cleaning the database.
	Store *coal.Store

	// The registered models.
	Models []coal.Model

	// The handler to be tested.
	Handler http.Handler

	// A path prefix e.g. 'api'.
	Prefix string

	// The header to be set on all requests and contexts.
	Header map[string]string

	// Context to be set on fake requests.
	Context context.Context
}

// NewTester returns a new tester.
func NewTester(store *coal.Store, models ...coal.Model) *Tester {
	return &Tester{
		Store:   store,
		Models:  models,
		Header:  make(map[string]string),
		Context: context.Background(),
	}
}

// Assign will create a controller group with the specified controllers and
// assign in to the Handler attribute of the tester.
func (t *Tester) Assign(prefix string, controllers ...*Controller) {
	group := NewGroup()
	group.Add(controllers...)
	group.Reporter = func(err error) {
		panic(err)
	}

	t.Handler = Compose(RootTracer(), group.Endpoint(prefix))
}

// Clean will remove the collections of models that have been registered and
// reset the header map.
func (t *Tester) Clean() {
	store := t.Store.Copy()
	defer store.Close()

	for _, model := range t.Models {
		// remove all is faster than dropping the collection
		_, err := store.C(model).RemoveAll(nil)
		if err != nil {
			panic(err)
		}
	}

	// reset header
	t.Header = make(map[string]string)

	// reset context
	t.Context = context.Background()
}

// Save will save the specified model.
func (t *Tester) Save(model coal.Model) coal.Model {
	store := t.Store.Copy()
	defer store.Close()

	// initialize model
	model = coal.Init(model)

	// insert to collection
	err := store.C(model).Insert(model)
	if err != nil {
		panic(err)
	}

	return model
}

// FindLast will return the last saved model.
func (t *Tester) FindLast(model coal.Model) coal.Model {
	store := t.Store.Copy()
	defer store.Close()

	err := store.C(model).Find(nil).Sort("-_id").One(model)
	if err != nil {
		panic(err)
	}

	return coal.Init(model)
}

// Path returns a root prefixed path for the supplied path.
func (t *Tester) Path(path string) string {
	// add root slash
	path = "/" + strings.Trim(path, "/")

	// add prefix if available
	if t.Prefix != "" {
		path = "/" + t.Prefix + path
	}

	return path
}

// RunAuthorizer is a helper to test validators. The caller should assert the
// returned error of the validator, the state of the supplied model and maybe
// other objects in the database.
//
// Note: Only the Operation, Query, Model and Store are set since these are the
// only attributes an authorizer should rely on.
//
// Note: A fake http request is set to allow access to request headers.
func (t *Tester) RunAuthorizer(op Operation, selector, filter bson.M, model coal.Model, authorizer *Callback) error {
	// call validator with context
	return t.RunHandler(op, selector, filter, model, nil, nil, authorizer.Handler)
}

// RunValidator is a helper to test validators. The caller should assert the
// returned error of the validator, the state of the supplied model and maybe
// other objects in the database.
//
// Note: Only the Operation, Model and Store attribute of the context are set since
// these are the only attributes a validator should rely on.
//
// Note: A fake http request is set to allow access to request headers.
func (t *Tester) RunValidator(op Operation, model coal.Model, validator *Callback) error {
	// check operation
	if !op.Write() {
		panic("fire: validators should only be run on create, update or delete")
	}

	// call validator with context
	return t.RunHandler(op, nil, nil, model, nil, nil, validator.Handler)
}

// RunNotifier is a helper to test notifiers. The caller should assert the
// returned error of the notifier and the state of the response or other triggered
// actions.
//
// Note: Only the Response attribute of the context is set since this is the
// only attribute a notifier should rely on.
//
// Note: A fake http request is set to allow access to request headers.
func (t *Tester) RunNotifier(op Operation, response *jsonapi.Document, notifier *Callback) error {
	// call notifier with context
	return t.RunHandler(op, nil, nil, nil, nil, response, notifier.Handler)
}

// WithContext runs the given function with a prepared context.
func (t *Tester) WithContext(op Operation, selector, filter bson.M, model coal.Model, fn func(*Context)) {
	t.RunHandler(op, selector, filter, model, nil, nil, func(ctx *Context) error {
		fn(ctx)
		return nil
	})
}

// RunHandler builds a context and runs the passed handler with it.
func (t *Tester) RunHandler(op Operation, selector, filter bson.M, model coal.Model, sorting []string, response *jsonapi.Document, h Handler) error {
	// get store
	store := t.Store.Copy()
	defer store.Close()

	// init model if present
	if model != nil {
		coal.Init(model)
	}

	// create request
	req, err := http.NewRequest("GET", "", nil)
	if err != nil {
		panic(err)
	}

	// set headers
	for key, value := range t.Header {
		req.Header.Set(key, value)
	}

	// set context
	req = req.WithContext(t.Context)

	// init queries
	if selector == nil {
		selector = bson.M{}
	}
	if filter == nil {
		filter = bson.M{}
	}

	// create tracer
	tracer := NewTracerWithRoot("fire/Tester.RunHandler")
	defer tracer.Finish(true)

	// create context
	ctx := &Context{
		Operation:      op,
		Selector:       selector,
		Filter:         filter,
		Model:          model,
		Sorting:        sorting,
		Response:       response,
		Store:          store,
		HTTPRequest:    req,
		ResponseWriter: httptest.NewRecorder(),
		Tracer:         tracer,
	}

	// call handler
	return h(ctx)
}

// Request will run the specified request against the registered handler. This
// function can be used to create custom testing facilities.
func (t *Tester) Request(method, path string, payload string, callback func(*httptest.ResponseRecorder, *http.Request)) {
	// create request
	request, err := http.NewRequest(method, t.Path(path), strings.NewReader(payload))
	if err != nil {
		panic(err)
	}

	// prepare recorder
	recorder := httptest.NewRecorder()

	// preset jsonapi accept header
	request.Header.Set("Accept", jsonapi.MediaType)

	// add content type if required
	if method == "POST" || method == "PATCH" || method == "DELETE" {
		request.Header.Set("Content-Type", jsonapi.MediaType)
	}

	// set custom headers
	for k, v := range t.Header {
		request.Header.Set(k, v)
	}

	// server request
	t.Handler.ServeHTTP(recorder, request)

	// run callback
	callback(recorder, request)
}

// DebugRequest returns a string of information to debug requests.
func (t *Tester) DebugRequest(r *http.Request, rr *httptest.ResponseRecorder) string {
	return fmt.Sprintf(`
	URL:    %s
	Header: %s
	Status: %d
	Header: %v
	Body:   %v`, r.URL.String(), r.Header, rr.Code, rr.HeaderMap, rr.Body.String())
}
