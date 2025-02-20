package coal

import (
	"context"
	"reflect"

	"github.com/256dpi/lungo/bsonkit"
	"github.com/256dpi/xo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: Validate updates before writing.

// Flags can be used to change the behaviour of operations.
type Flags int

const (
	// NoTransaction will allow running operations without a transaction that by
	// default require a transaction.
	NoTransaction Flags = 1 << iota

	// NoValidation will allow storing and retrieving invalid models.
	NoValidation

	// TextScoreSort will prepend the sort with a sort based on the text score
	// of documents. The Base.Score attribute is set to the respective score.
	TextScoreSort
)

// Has returns whether the receiver has set all provided flags.
func (f Flags) Has(flags Flags) bool {
	return f&flags == flags
}

// Merge will combine the provided flags.
func Merge(flags []Flags) Flags {
	var f Flags
	for _, l := range flags {
		f |= l
	}
	return f
}

// ErrTransactionRequired is returned if an operation would be unsafe to perform
// without a transaction.
var ErrTransactionRequired = xo.BF("operation requires a transaction")

// ErrMetaMismatch is returned if the provided model does not match the model
// that is manged by the manager.
var ErrMetaMismatch = xo.BF("provided model does not match managed model")

var incrementLock = bson.M{
	"$inc": bson.M{
		"_lk": 1,
	},
}

var metaTextScore = bson.M{
	"$meta": "textScore",
}

var returnAfterUpdate = options.FindOneAndUpdate().SetReturnDocument(options.After)

// Manager manages operations on collection of documents. It will validate
// operations and ensure that they are safe under the MongoDB guarantees.
type Manager struct {
	meta  *Meta
	coll  *Collection
	trans *Translator
}

// C is a shorthand to access the underlying collection.
func (m *Manager) C() *Collection {
	return m.coll
}

// T is a shorthand to access the underlying translator.
func (m *Manager) T() *Translator {
	return m.trans
}

// Find will find the document with the specified id. It will return whether
// a document has been found. Lock can be set to true to force a write lock on
// the document and prevent a stale read during a transaction.
//
// A transaction is required for locking.
func (m *Manager) Find(ctx context.Context, model Model, id ID, lock bool, flags ...Flags) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Find")
	span.Tag("id", id)
	defer span.End()

	// check lock
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// ensure model
	if model == nil {
		model = m.meta.Make()
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// prepare filter
	filter := bson.M{
		"_id": id,
	}

	// find document
	var err error
	if lock {
		err = m.coll.FindOneAndUpdate(ctx, filter, incrementLock, returnAfterUpdate).Decode(model)
	} else {
		err = m.coll.FindOne(ctx, filter).Decode(model)
	}
	if IsMissing(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	// validate model
	if !Merge(flags).Has(NoValidation) {
		err = model.Validate()
		if err != nil {
			return false, xo.W(err)
		}
	}

	return true, nil
}

// FindFirst will find the first document that matches the specified filter. It
// will return whether a document has been found. Lock can be set to true to
// force a write lock on the document and prevent a stale read during a
// transaction.
//
// A transaction is required for locking.
//
// Warning: If the operation depends on interleaving writes to not include or
// exclude documents from the filter it should be run during a transaction.
func (m *Manager) FindFirst(ctx context.Context, model Model, filter bson.M, sort []string, skip int64, lock bool, flags ...Flags) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.FindFirst")
	defer span.End()

	// check lock
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// check lock
	if lock && (skip > 0) {
		return false, xo.F("cannot lock with skip")
	}

	// check model
	if model == nil {
		model = m.meta.Make()
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return false, err
	}

	// translate sort
	var sortDoc bson.D
	if len(sort) > 0 {
		sortDoc, err = m.trans.Sort(sort)
		if err != nil {
			return false, err
		}
	}

	// find document
	if lock {
		// prepare options
		opts := options.FindOneAndUpdate()
		if sortDoc != nil {
			opts.SetSort(sortDoc)
		}

		// find and update
		err = m.coll.FindOneAndUpdate(ctx, filterDoc, incrementLock, returnAfterUpdate, opts).Decode(model)
	} else {
		// prepare options
		opts := options.FindOne()
		if sortDoc != nil {
			opts.SetSort(sortDoc)
		}
		if skip > 0 {
			opts.SetSkip(skip)
		}

		// find
		err = m.coll.FindOne(ctx, filterDoc, opts).Decode(model)
	}
	if IsMissing(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	// validate model
	if !Merge(flags).Has(NoValidation) {
		err = model.Validate()
		if err != nil {
			return false, xo.W(err)
		}
	}

	return true, nil
}

// FindAll will find all documents that match the specified filter. Lock can be
// set to true to force a write lock on the documents and prevent a stale read
// during a transaction.
//
// A transaction is required to ensure isolation.
//
// NoTransaction: The result may miss documents or include them multiple times
// if interleaving operations move the documents in the used index.
func (m *Manager) FindAll(ctx context.Context, list interface{}, filter bson.M, sort []string, skip, limit int64, lock bool, flags ...Flags) error {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.FindAll")
	defer span.End()

	// check list
	if list == nil {
		return xo.F("missing list")
	}
	lt := reflect.TypeOf(list)
	if lt.Kind() != reflect.Ptr || lt.Elem().Kind() != reflect.Slice {
		return xo.F("expected slice pointer")
	}
	et := lt.Elem().Elem()
	if (et.Kind() == reflect.Struct && et != m.meta.Type) || (et.Kind() == reflect.Ptr && et.Elem() != m.meta.Type) {
		return xo.F("expected slice of matching models")
	}

	// require transaction if locked or not unsafe
	if (lock || !Merge(flags).Has(NoTransaction)) && !HasTransaction(ctx) {
		return ErrTransactionRequired.Wrap()
	}

	// check lock
	if lock && (skip > 0 || limit > 0) {
		return xo.F("cannot lock with skip and limit")
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return err
	}

	// prepare options
	opts := options.Find()

	// set sort
	if len(sort) > 0 {
		opts.Sort, err = m.trans.Sort(sort)
		if err != nil {
			return err
		}
	}

	// set skip
	if skip > 0 {
		opts.SetSkip(skip)
	}

	// set limit
	if limit > 0 {
		opts.SetLimit(limit)
	}

	// handle text score sort
	if Merge(flags).Has(TextScoreSort) {
		// set projection
		opts.SetProjection(bson.M{
			"_sc": metaTextScore,
		})

		// prepend score sort
		rawSort, _ := opts.Sort.(bson.D)
		rawSort = append(bson.D{
			{Key: "_sc", Value: metaTextScore},
		}, rawSort...)
		opts.SetSort(rawSort)
	}

	// lock documents
	if lock {
		_, err = m.coll.UpdateMany(ctx, filterDoc, incrementLock)
		if err != nil {
			return err
		}
	}

	// find documents
	iter, err := m.coll.Find(ctx, filterDoc, opts)
	if err != nil {
		return err
	}

	// decode all
	err = iter.All(list)
	if err != nil {
		return err
	}

	// validate models
	if !Merge(flags).Has(NoValidation) {
		for _, model := range Slice(list) {
			err = model.Validate()
			if err != nil {
				return xo.W(err)
			}
		}
	}

	return nil
}

// FindEach will find all documents that match the specified filter. Lock can be
// set to true to force a write lock on the documents and prevent a stale read
// during a transaction.
//
// A transaction is required to ensure isolation.
//
// NoTransaction: The result may miss documents or include them multiple times
// if interleaving operations move the documents in the used index.
func (m *Manager) FindEach(ctx context.Context, filter bson.M, sort []string, skip, limit int64, lock bool, flags ...Flags) (*ManagedIterator, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.FindEach")

	// finish span on error
	var iter *Iterator
	defer func() {
		if iter == nil {
			span.End()
		}
	}()

	// require transaction if locked or not unsafe
	if (lock || !Merge(flags).Has(NoTransaction)) && !HasTransaction(ctx) {
		return nil, ErrTransactionRequired.Wrap()
	}

	// check lock
	if lock && (skip > 0 || limit > 0) {
		return nil, xo.F("cannot lock with skip and limit")
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return nil, err
	}

	// prepare options
	opts := options.Find()

	// set sort
	if len(sort) > 0 {
		opts.Sort, err = m.trans.Sort(sort)
		if err != nil {
			return nil, err
		}
	}

	// set skip
	if skip > 0 {
		opts.SetSkip(skip)
	}

	// set limit
	if limit > 0 {
		opts.SetLimit(limit)
	}

	// lock documents
	if lock {
		_, err = m.coll.UpdateMany(ctx, filterDoc, incrementLock)
		if err != nil {
			return nil, err
		}
	}

	// find documents
	iter, err = m.coll.Find(ctx, filterDoc, opts)
	if err != nil {
		return nil, err
	}

	// attach span
	iter.spans = append(iter.spans, span)

	// determine validation
	validate := !Merge(flags).Has(NoValidation)

	return &ManagedIterator{
		meta:     m.meta,
		iterator: iter,
		validate: validate,
	}, nil
}

// Project will return the field of the specified document. It will also return
// whether a document has been found at all.
//
// A transaction is required for locking.
func (m *Manager) Project(ctx context.Context, id ID, field string, lock bool) (interface{}, bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Project")
	defer span.End()

	// project
	var res interface{}
	var found bool
	err := m.project(ctx, bson.M{
		"_id": id,
	}, field, nil, 0, 1, lock, func(id ID, val interface{}) bool {
		res = val
		found = true
		return false
	}, NoTransaction)
	if err != nil {
		return nil, false, err
	}

	return res, found, nil
}

// ProjectFirst will return the field of the first matching document. It will
// also return whether a document has been found at all.
//
// A transaction is required for locking.
//
// Warning: If the operation depends on interleaving writes to not include or
// exclude documents from the filter it should be run during a transaction.
func (m *Manager) ProjectFirst(ctx context.Context, filter bson.M, field string, sort []string, skip int64, lock bool) (interface{}, bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.ProjectFirst")
	defer span.End()

	// project
	var res interface{}
	var found bool
	err := m.project(ctx, filter, field, sort, skip, 1, lock, func(id ID, val interface{}) bool {
		res = val
		found = true
		return false
	}, NoTransaction)
	if err != nil {
		return nil, false, err
	}

	return res, found, nil
}

// ProjectAll will look up the specified field for all matching documents and
// return a map with their ids and field values.
//
// A transaction is required to ensure isolation.
//
// NoTransaction: The result may miss documents or include them multiple times
// if interleaving operations move the documents in the used index.
func (m *Manager) ProjectAll(ctx context.Context, filter bson.M, field string, sort []string, skip, limit int64, lock bool, flags ...Flags) (map[ID]interface{}, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.ProjectAll")
	defer span.End()

	// project
	res := make(map[ID]interface{})
	err := m.project(ctx, filter, field, sort, skip, limit, lock, func(id ID, val interface{}) bool {
		res[id] = val
		return true
	}, flags...)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ProjectEach will look up the specified field for all documents matching the
// specified filter and yield them to the provided function until all have been
// found or false has been returned.
//
// A transaction is required to ensure isolation.
//
// NoTransaction: The result may miss documents or include them multiple times
// if interleaving operations move the documents in the used index.
func (m *Manager) ProjectEach(ctx context.Context, filter bson.M, field string, sort []string, skip, limit int64, lock bool, fn func(id ID, val interface{}) bool, flags ...Flags) error {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.ProjectEach")
	defer span.End()

	return m.project(ctx, filter, field, sort, skip, limit, lock, fn, flags...)
}

func (m *Manager) project(ctx context.Context, filter bson.M, field string, sort []string, skip, limit int64, lock bool, fn func(id ID, val interface{}) bool, flags ...Flags) error {
	// require transaction if locked or not unsafe
	if (lock || !Merge(flags).Has(NoTransaction)) && !HasTransaction(ctx) {
		return ErrTransactionRequired.Wrap()
	}

	// check lock
	if lock && (skip > 0 || limit > 0) {
		return xo.F("cannot lock with skip and limit")
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return err
	}

	// translate field
	field, err = m.trans.Field(field)
	if err != nil {
		return err
	}

	// prepare options
	opts := options.Find()

	// set sort
	if len(sort) > 0 {
		opts.Sort, err = m.trans.Sort(sort)
		if err != nil {
			return err
		}
	}

	// set skip
	if skip > 0 {
		opts.SetSkip(skip)
	}

	// set limit
	if limit > 0 {
		opts.SetLimit(limit)
	}

	// set projection
	opts.SetProjection(bson.M{
		field: 1,
	})

	// lock documents
	if lock {
		_, err = m.coll.UpdateMany(ctx, filterDoc, incrementLock)
		if err != nil {
			return err
		}
	}

	// find documents
	iter, err := m.coll.Find(ctx, filterDoc, opts)
	if err != nil {
		return err
	}

	// ensure close
	defer iter.Close()

	// iterate
	for iter.Next() {
		// decode item
		item := make(map[string]interface{})
		err = iter.Decode(&item)
		if err != nil {
			return err
		}

		// yield pair
		if !fn(item["_id"].(ID), item[field]) {
			break
		}
	}

	// check error
	err = iter.Error()
	if err != nil {
		return err
	}

	return nil
}

// Count will count the documents that match the specified filter. Lock can be
// set to true to force a write lock on the documents and prevent a stale read
// during a transaction.
//
// A transaction is required to ensure isolation.
//
// NoTransaction: The result may miss documents or include them multiple times
// if interleaving operations move the documents in the used index.
func (m *Manager) Count(ctx context.Context, filter bson.M, skip, limit int64, lock bool, flags ...Flags) (int64, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Count")
	defer span.End()

	// require transaction if locked or not unsafe
	if (lock || !Merge(flags).Has(NoTransaction)) && !HasTransaction(ctx) {
		return 0, ErrTransactionRequired.Wrap()
	}

	// check lock
	if lock && (skip > 0 || limit > 0) {
		return 0, xo.F("cannot lock with skip and limit")
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return 0, err
	}

	// prepare options
	opts := options.Count()

	// set skip
	if skip > 0 {
		opts.SetSkip(skip)
	}

	// set limit
	if limit > 0 {
		opts.SetLimit(limit)
	}

	// update if locked
	if lock {
		res, err := m.coll.UpdateMany(ctx, filterDoc, incrementLock)
		if err != nil {
			return 0, err
		}

		return res.ModifiedCount, nil
	}

	// count documents
	count, err := m.coll.CountDocuments(ctx, filterDoc, opts)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Distinct will find all documents that match the specified filter and collect
// the specified field. Lock can be set to true to force a write lock on the
// documents and prevent a stale read during a transaction.
//
// A transaction is required to ensure isolation.
//
// NoTransaction: The result may miss documents or include them multiple times
// if interleaving operations move the documents in the used index.
func (m *Manager) Distinct(ctx context.Context, field string, filter bson.M, lock bool, flags ...Flags) ([]interface{}, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Distinct")
	defer span.End()

	// require transaction if locked or not unsafe
	if (lock || !Merge(flags).Has(NoTransaction)) && !HasTransaction(ctx) {
		return nil, ErrTransactionRequired.Wrap()
	}

	// translate field
	field, err := m.trans.Field(field)
	if err != nil {
		return nil, err
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return nil, err
	}

	// lock documents
	if lock {
		_, err = m.coll.UpdateMany(ctx, filterDoc, incrementLock)
		if err != nil {
			return nil, err
		}
	}

	// distinct
	result, err := m.coll.Distinct(ctx, field, filterDoc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Insert will insert the provided document. If the document has a zero id a new
// id will be generated and assigned.
func (m *Manager) Insert(ctx context.Context, models Model, flags ...Flags) error {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Insert")
	defer span.End()

	return m.insert(ctx, []Model{models}, flags...)
}

// InsertAll will insert the provided documents. If a document has a zero id a
// new id will be generated and assigned. The documents are inserted in order
// until an error is encountered.
func (m *Manager) InsertAll(ctx context.Context, models []Model, flags ...Flags) error {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.InsertAll")
	defer span.End()

	return m.insert(ctx, models, flags...)
}

func (m *Manager) insert(ctx context.Context, models []Model, flags ...Flags) error {
	// check length
	if len(models) == 0 {
		return nil
	}

	// check models and ensure ids
	for _, model := range models {
		// check model
		if GetMeta(model) != m.meta {
			return ErrMetaMismatch.Wrap()
		}

		// ensure id
		if model.ID() == "" {
			model.GetBase().DocID = New()
		}
	}

	// validate models
	if !Merge(flags).Has(NoValidation) {
		for _, model := range models {
			err := model.Validate()
			if err != nil {
				return xo.W(err)
			}
		}
	}

	// get documents
	docs := make([]interface{}, 0, len(models))
	for _, model := range models {
		docs = append(docs, model)
	}

	// insert documents or document
	if len(docs) > 1 {
		_, err := m.coll.InsertMany(ctx, docs, options.InsertMany().SetOrdered(true))
		if err != nil {
			return err
		}
	} else {
		_, err := m.coll.InsertOne(ctx, docs[0])
		if err != nil {
			return err
		}
	}

	return nil
}

// InsertIfMissing will insert the provided document if no document matched the
// provided filter. If the document has a zero id a new id will be generated and
// assigned. It will return whether a document has been inserted. The underlying
// upsert operation will Merge the filter with the model fields. Lock can be set
// to true to force a write lock on the existing document and prevent a stale
// read during a transaction.
//
// A transaction is required for locking.
//
// Warning: Even with transactions there is a risk for duplicate inserts when
// the filter is not covered by a unique index.
func (m *Manager) InsertIfMissing(ctx context.Context, filter bson.M, model Model, lock bool, flags ...Flags) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.InsertIfMissing")
	defer span.End()

	// require transaction
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return false, err
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// ensure id
	if model.ID() == "" {
		model.GetBase().DocID = New()
	}

	// validate model
	if !Merge(flags).Has(NoValidation) {
		err = model.Validate()
		if err != nil {
			return false, xo.W(err)
		}
	}

	// prepare options
	opts := options.Update().SetUpsert(true)

	// prepare update
	update := bson.M{
		"$setOnInsert": model,
	}

	// increment lock
	if lock {
		update["$inc"] = bson.M{
			"_lk": 1,
		}
	}

	// upsert document
	res, err := m.coll.UpdateOne(ctx, filterDoc, update, opts)
	if err != nil {
		return false, err
	}

	return res.UpsertedCount == 1, nil
}

// Replace will replace the existing document with the provided one. It will
// return whether a document has been found. Lock can be set to true to force a
// write lock on the document and prevent a stale read during a transaction in
// case the replace did not change the document.
//
// A transaction is required for locking.
func (m *Manager) Replace(ctx context.Context, model Model, lock bool, flags ...Flags) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Replace")
	defer span.End()

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// check id
	if model.ID() == "" {
		return false, xo.F("model has a zero id")
	}

	// require transaction
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// validate model
	if !Merge(flags).Has(NoValidation) {
		err := model.Validate()
		if err != nil {
			return false, xo.W(err)
		}
	}

	// increment lock manually
	if lock {
		model.GetBase().Lock += 1000
	}

	// replace document
	res, err := m.coll.ReplaceOne(ctx, bson.M{
		"_id": model.ID(),
	}, model)
	if err != nil {
		return false, err
	}

	return res.MatchedCount == 1, nil
}

// ReplaceFirst will replace the first document that matches the specified filter.
// It will return whether a document has been found. Lock can be set to true to
// force a write lock on the document and prevent a stale read during a
// transaction if the replace did not cause an update.
//
// A transaction is required for locking.
//
// Warning: If the operation depends on interleaving writes to not include or
// exclude documents from the filter it should be run as part of a transaction.
func (m *Manager) ReplaceFirst(ctx context.Context, filter bson.M, model Model, lock bool, flags ...Flags) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.ReplaceFirst")
	defer span.End()

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// require transaction
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// validate model
	if !Merge(flags).Has(NoValidation) {
		err := model.Validate()
		if err != nil {
			return false, xo.W(err)
		}
	}

	// increment lock manually
	if lock {
		model.GetBase().Lock += 1000
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return false, err
	}

	// replace document
	res, err := m.coll.ReplaceOne(ctx, filterDoc, model)
	if err != nil {
		return false, err
	}

	return res.MatchedCount == 1, nil
}

// Update will update the document with the specified id. It will return whether
// a document has been found. Lock can be set to true to force a write lock on
// the document and prevent a stale read during a transaction in case the
// update did not change the document.
//
// A transaction is required for locking.
func (m *Manager) Update(ctx context.Context, model Model, id ID, update bson.M, lock bool) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Update")
	defer span.End()

	// require transaction
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// check model
	if model == nil {
		model = m.meta.Make()
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// translate update
	updateDoc, err := m.trans.Document(update)
	if err != nil {
		return false, err
	}

	// increment lock
	if lock {
		_, err := bsonkit.Put(&updateDoc, "$inc._lk", 1, false)
		if err != nil {
			return false, xo.WF(err, "unable to add lock")
		}
	}

	// find and update document
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err = m.coll.FindOneAndUpdate(ctx, bson.M{
		"_id": id,
	}, updateDoc, opts).Decode(model)
	if IsMissing(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// UpdateFirst will update the first document that matches the specified filter.
// It will return whether a document has been found. Lock can be set to true to
// force a write lock on the document and prevent a stale read during a
// transaction in case the update did not change the document.
//
// A transaction is required for locking.
//
// Warning: If the operation depends on interleaving writes to not include or
// exclude documents from the filter it should be run as part of a transaction.
func (m *Manager) UpdateFirst(ctx context.Context, model Model, filter, update bson.M, sort []string, lock bool) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.UpdateFirst")
	defer span.End()

	// require transaction
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// check model
	if model == nil {
		model = m.meta.Make()
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return false, err
	}

	// translate update
	updateDoc, err := m.trans.Document(update)
	if err != nil {
		return false, err
	}

	// prepare options
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	// set sort
	if len(sort) > 0 {
		opts.Sort, err = m.trans.Sort(sort)
		if err != nil {
			return false, err
		}
	}

	// increment lock
	if lock {
		_, err := bsonkit.Put(&updateDoc, "$inc._lk", 1, false)
		if err != nil {
			return false, xo.WF(err, "unable to add lock")
		}
	}

	// find and update document
	err = m.coll.FindOneAndUpdate(ctx, filterDoc, updateDoc, opts).Decode(model)
	if IsMissing(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// UpdateAll will update the documents that match the specified filter. It will
// return the number of matched documents. Lock can be set to true to force a
// write lock on the documents and prevent a stale read during a transaction in
// case the operation did not change all documents.
//
// A transaction is required for locking.
//
// Warning: If the operation depends on interleaving writes to not include or
// exclude documents from the filter it should be run as part of a transaction.
func (m *Manager) UpdateAll(ctx context.Context, filter, update bson.M, lock bool) (int64, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.UpdateAll")
	defer span.End()

	// require transaction
	if lock && !HasTransaction(ctx) {
		return 0, ErrTransactionRequired.Wrap()
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return 0, err
	}

	// translate update
	updateDoc, err := m.trans.Document(update)
	if err != nil {
		return 0, err
	}

	// increment lock
	if lock {
		_, err := bsonkit.Put(&updateDoc, "$inc._lk", 1, false)
		if err != nil {
			return 0, xo.WF(err, "unable to add lock")
		}
	}

	// update documents
	res, err := m.coll.UpdateMany(ctx, filterDoc, updateDoc)
	if err != nil {
		return 0, err
	}

	return res.MatchedCount, nil
}

// Upsert will update the first document that matches the specified filter. If
// no document has been found, the update document is applied to the filter and
// inserted. It will return whether a document has been inserted. Lock can be
// set to true to force a write lock on the existing document and prevent a
// stale read during a transaction.
//
// A transaction is required for locking.
//
// Warning: Even with transactions there is a risk for duplicate inserts when
// the filter is not covered by a unique index.
func (m *Manager) Upsert(ctx context.Context, model Model, filter, update bson.M, sort []string, lock bool) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Upsert")
	defer span.End()

	// require transaction
	if lock && !HasTransaction(ctx) {
		return false, ErrTransactionRequired.Wrap()
	}

	// check model
	if model == nil {
		model = m.meta.Make()
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return false, err
	}

	// translate update
	updateDoc, err := m.trans.Document(update)
	if err != nil {
		return false, err
	}

	// increment lock
	if lock {
		_, err := bsonkit.Put(&updateDoc, "$inc._lk", 1, false)
		if err != nil {
			return false, xo.WF(err, "unable to add lock")
		}
	}

	// prepare options
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	// set sort
	if len(sort) > 0 {
		opts.Sort, err = m.trans.Sort(sort)
		if err != nil {
			return false, err
		}
	}

	// set token (to determine insert vs. update)
	token := New()
	_, err = bsonkit.Put(&updateDoc, "$setOnInsert._tk", token, false)
	if err != nil {
		return false, xo.WF(err, "unable to set token")
	}

	// find and update document
	err = m.coll.FindOneAndUpdate(ctx, filterDoc, updateDoc, opts).Decode(model)
	if IsMissing(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return model.GetBase().Token == token, nil
}

// Delete will remove the document with the specified id. It will return
// whether a document has been found and deleted.
func (m *Manager) Delete(ctx context.Context, model Model, id ID) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.Delete")
	defer span.End()

	// delete document
	if model == nil {
		res, err := m.coll.DeleteOne(ctx, bson.M{
			"_id": id,
		})
		if err != nil {
			return false, err
		}

		return res.DeletedCount == 1, nil
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// find and delete document
	err := m.coll.FindOneAndDelete(ctx, bson.M{
		"_id": id,
	}).Decode(model)
	if IsMissing(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// DeleteAll will delete the documents that match the specified filter. It will
// return the number of deleted documents.
//
// Warning: If the operation depends on interleaving writes to not include or
// exclude documents from the filter it should be run as part of a transaction.
func (m *Manager) DeleteAll(ctx context.Context, filter bson.M) (int64, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.DeleteAll")
	defer span.End()

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return 0, err
	}

	// update documents
	res, err := m.coll.DeleteMany(ctx, filterDoc)
	if err != nil {
		return 0, err
	}

	return res.DeletedCount, nil
}

// DeleteFirst will delete the first document that matches the specified filter.
// It will return whether a document has been found and deleted.
//
// Warning: If the operation depends on interleaving writes to not include or
// exclude documents from the filter it should be run as part of a transaction.
func (m *Manager) DeleteFirst(ctx context.Context, model Model, filter bson.M, sort []string) (bool, error) {
	// trace
	ctx, span := xo.Trace(ctx, "coal/Manager.DeleteFirst")
	defer span.End()

	// translate filter
	filterDoc, err := m.trans.Document(filter)
	if err != nil {
		return false, err
	}

	// check model
	if model == nil {
		model = m.meta.Make()
	}

	// check model
	if GetMeta(model) != m.meta {
		return false, ErrMetaMismatch.Wrap()
	}

	// prepare options
	opts := options.FindOneAndDelete()

	// set sort
	if len(sort) > 0 {
		opts.Sort, err = m.trans.Sort(sort)
		if err != nil {
			return false, err
		}
	}

	// find and delete document
	err = m.coll.FindOneAndDelete(ctx, filterDoc, opts).Decode(model)
	if IsMissing(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// ManagedIterator wraps an iterator to enforce decoding to a model.
type ManagedIterator struct {
	meta     *Meta
	iterator *Iterator
	validate bool
}

// Next will load the next document from the cursor and if available return true.
// If it returns false the iteration must be stopped due to the cursor being
// exhausted or an error.
func (i *ManagedIterator) Next() bool {
	return i.iterator.Next()
}

// Decode will decode the loaded document to the specified model.
func (i *ManagedIterator) Decode(model Model) error {
	// check model
	if GetMeta(model) != i.meta {
		return ErrMetaMismatch.Wrap()
	}

	// decode
	err := i.iterator.Decode(model)
	if err != nil {
		return err
	}

	// validate if requested
	if i.validate {
		err = model.Validate()
		if err != nil {
			return xo.W(err)
		}
	}

	return nil
}

// Error returns the first error encountered during iteration. It should always
// be checked when done to ensure there have been no errors.
func (i *ManagedIterator) Error() error {
	return i.iterator.Error()
}

// Close will close the underlying cursor. A call to it should be deferred right
// after obtaining an iterator. Close should be called also if the iterator is
// still valid but no longer used by the application.
func (i *ManagedIterator) Close() {
	i.iterator.Close()
}
