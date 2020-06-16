package coal

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/256dpi/lungo"
	"github.com/256dpi/xo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// MustConnect will call Connect and panic on errors.
func MustConnect(uri string) *Store {
	// connect store
	store, err := Connect(uri)
	if err != nil {
		panic(err)
	}

	return store
}

// Connect will connect to the specified database and return a new store. The
// read and write concern is set to majority by default.
//
// In summary, queries may return data that has bas been committed but may not
// be the most recent committed data. Also, long running cursors on indexed
// fields may return duplicate or missing documents due to the documents moving
// within the index. For operations involving multiple documents a transaction
// must be used to ensure atomicity, consistency and isolation.
func Connect(uri string) (*Store, error) {
	// parse url
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, xo.W(err)
	}

	// get default db
	defaultDB := strings.Trim(parsedURL.Path, "/")

	// prepare options
	opts := options.Client().ApplyURI(uri)
	opts.SetReadConcern(readconcern.Linearizable())
	opts.SetWriteConcern(writeconcern.New(writeconcern.WMajority()))

	// create client
	client, err := lungo.Connect(nil, opts)
	if err != nil {
		return nil, xo.W(err)
	}

	// ping server
	err = client.Ping(nil, nil)
	if err != nil {
		return nil, xo.W(err)
	}

	return NewStore(client, defaultDB, nil), nil
}

// MustOpen will call Open and panic on errors.
func MustOpen(store lungo.Store, defaultDB string, reporter func(error)) *Store {
	// open store
	s, err := Open(store, defaultDB, reporter)
	if err != nil {
		panic(err)
	}

	return s
}

// Open will open the database using the provided lungo store. If the store is
// missing an in-memory store will be created.
func Open(store lungo.Store, defaultDB string, reporter func(error)) (*Store, error) {
	// set default memory store
	if store == nil {
		store = lungo.NewMemoryStore()
	}

	// create client
	client, engine, err := lungo.Open(nil, lungo.Options{
		Store:          store,
		ExpireInterval: time.Minute,
		ExpireErrors:   reporter,
	})
	if err != nil {
		return nil, xo.W(err)
	}

	return NewStore(client, defaultDB, engine), nil
}

// NewStore creates a store that uses the specified client, default database and
// engine. The engine may be nil if no lungo database is used.
func NewStore(client lungo.IClient, defaultDB string, engine *lungo.Engine) *Store {
	return &Store{
		client: client,
		defDB:  defaultDB,
		engine: engine,
	}
}

// A Store manages the usage of a database client.
type Store struct {
	client   lungo.IClient
	defDB    string
	engine   *lungo.Engine
	colls    sync.Map
	managers sync.Map
}

// Client returns the client used by this store.
func (s *Store) Client() lungo.IClient {
	return s.client
}

// DB returns the database used by this store.
func (s *Store) DB() lungo.IDatabase {
	return s.client.Database(s.defDB)
}

// C will return the collection for the specified model. The collection is just
// a thin wrapper around the driver collection API to integrate tracing. Since
// it does not perform any checks, it is recommended to use the manager to
// perform safe CRUD operations.
func (s *Store) C(model Model) *Collection {
	// get meta
	meta := GetMeta(model)

	// check cache
	val, ok := s.colls.Load(meta)
	if ok {
		return val.(*Collection)
	}

	// create collection
	coll := &Collection{
		coll: s.DB().Collection(meta.Collection),
	}

	// cache collection
	s.colls.Store(meta, coll)

	return coll
}

// M will return the manager for the specified model. The manager will translate
// query and update documents as well as perform extensive checks before running
// operations to ensure they are as safe as possible.
func (s *Store) M(model Model) *Manager {
	// get meta
	meta := GetMeta(model)

	// check cache
	val, ok := s.managers.Load(meta)
	if ok {
		return val.(*Manager)
	}

	// create manager
	manager := &Manager{
		meta:  meta,
		coll:  s.C(model),
		trans: NewTranslator(model),
	}

	// cache collection
	s.managers.Store(meta, manager)

	return manager
}

// T will create a transaction around the specified callback. If the callback
// returns no error the transaction will be committed. If T itself does not
// return an error the transaction has been committed. The created context must
// be used with all operations that should be included in the transaction.
//
// A transaction has the effect that the read concern is upgraded to "snapshot"
// which results in isolated and linearizable reads and writes of the data that
// has been committed prior to the start of the transaction:
//
// - Writes that conflict with other transactional writes will return an error.
//   Non-transactional writes will wait until the transaction has completed.
// - Reads are not guaranteed to be stable, another transaction may delete or
//   modify the document an also commit concurrently. Therefore, documents that
//   must "survive" the transaction and cause transactional writes to abort,
//   must be locked by incrementing or changing a field to a new value.
func (s *Store) T(ctx context.Context, fn func(context.Context) error) error {
	// set context background
	if ctx == nil {
		ctx = context.Background()
	}

	// check if transaction already exists
	if HasTransaction(ctx) {
		return fn(ctx)
	}

	// trace
	ctx, span := xo.Trace(ctx, "coal/Store.T")
	defer span.End()

	// prepare options
	opts := options.Session().
		SetCausalConsistency(true).
		SetDefaultReadConcern(readconcern.Snapshot())

	// start transaction
	return xo.W(s.client.UseSessionWithOptions(ctx, opts, func(sc lungo.ISessionContext) error {
		// start transaction
		err := sc.StartTransaction()
		if err != nil {
			return xo.W(err)
		}

		// call function
		err = fn(withKey(sc, hasTransaction))
		if err != nil {
			_ = sc.AbortTransaction(sc)
			return xo.W(err)
		}

		// commit transaction
		err = sc.CommitTransaction(sc)
		if err != nil {
			return xo.W(err)
		}

		return nil
	}))
}

// Close will close the store and its associated client.
func (s *Store) Close() error {
	// disconnect client
	err := s.client.Disconnect(nil)
	if err != nil {
		return xo.W(err)
	}

	// close engine
	if s.engine != nil {
		s.engine.Close()
	}

	return nil
}

type contextKey struct{}

var hasTransaction = contextKey{}

func withKey(ctx context.Context, key interface{}) context.Context {
	return context.WithValue(ctx, key, true)
}

func getKey(ctx context.Context, key interface{}) bool {
	if ctx != nil {
		ok, _ := ctx.Value(key).(bool)
		return ok
	}

	return false
}

// HasTransaction will return whether the context carries a transaction.
func HasTransaction(ctx context.Context) bool {
	return getKey(ctx, hasTransaction)
}
