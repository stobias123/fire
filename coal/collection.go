package coal

import (
	"context"
	"errors"

	"github.com/256dpi/lungo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/256dpi/fire/cinder"
)

// ErrBreak can be returned to break out from an iterator.
var ErrBreak = errors.New("break")

// IsMissing returns whether the provided error describes a missing document.
func IsMissing(err error) bool {
	return err == lungo.ErrNoDocuments || errors.Is(err, lungo.ErrNoDocuments)
}

// IsDuplicate returns whether the provided error describes a duplicate document.
func IsDuplicate(err error) bool {
	return lungo.IsUniquenessError(err)
}

// Collection mimics a collection and adds tracing.
type Collection struct {
	coll lungo.ICollection
}

// Native will return the underlying native collection.
func (c *Collection) Native() lungo.ICollection {
	return c.coll
}

// Aggregate wraps the native Aggregate collection method and yields the
// returned cursor.
func (c *Collection) Aggregate(ctx context.Context, pipeline interface{}, fn func(lungo.ICursor) error, opts ...*options.AggregateOptions) error {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.Aggregate")
	span.Log("pipeline", pipeline)
	defer span.Finish()

	// aggregate
	csr, err := c.coll.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		return err
	}

	// yield cursor
	err = fn(csr)
	if err != nil {
		_ = csr.Close(ctx)
		return err
	}

	// close cursor
	err = csr.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}

// AggregateAll wraps the native Aggregate collection method and decodes all
// documents to the provided slice.
func (c *Collection) AggregateAll(ctx context.Context, slicePtr interface{}, pipeline interface{}, opts ...*options.AggregateOptions) error {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.AggregateAll")
	span.Log("pipeline", pipeline)
	defer span.Finish()

	// aggregate
	csr, err := c.coll.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		return err
	}

	// decode all documents
	err = csr.All(ctx, slicePtr)
	if err != nil {
		_ = csr.Close(ctx)
		return err
	}

	return nil
}

// AggregateIter wraps the native Aggregate collection method and calls the
// provided callback with the decode method until ErrBreak is returned or the
// cursor has been exhausted.
func (c *Collection) AggregateIter(ctx context.Context, pipeline interface{}, fn func(decode func(interface{}) error) error, opts ...*options.AggregateOptions) error {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.AggregateIter")
	span.Log("pipeline", pipeline)
	defer span.Finish()

	// aggregate
	csr, err := c.coll.Aggregate(ctx, pipeline, opts...)
	if err != nil {
		return err
	}

	// iterate over all documents
	for csr.Next(ctx) {
		err = fn(csr.Decode)
		if err == ErrBreak {
			break
		} else if err != nil {
			_ = csr.Close(ctx)
			return err
		}
	}

	// close cursor
	err = csr.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}

// BulkWrite wraps the native BulkWrite collection method.
func (c *Collection) BulkWrite(ctx context.Context, models []mongo.WriteModel, opts ...*options.BulkWriteOptions) (*mongo.BulkWriteResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.BulkWrite")
	defer span.Finish()

	return c.coll.BulkWrite(ctx, models, opts...)
}

// CountDocuments wraps the native CountDocuments collection method.
func (c *Collection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.CountDocuments")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.CountDocuments(ctx, filter, opts...)
}

// DeleteMany wraps the native DeleteMany collection method.
func (c *Collection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.DeleteMany")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.DeleteMany(ctx, filter, opts...)
}

// DeleteOne wraps the native DeleteOne collection method.
func (c *Collection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.DeleteOne")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.DeleteOne(ctx, filter, opts...)
}

// Distinct wraps the native Distinct collection method.
func (c *Collection) Distinct(ctx context.Context, field string, filter interface{}, opts ...*options.DistinctOptions) ([]interface{}, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.Distinct")
	span.Log("field", field)
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.Distinct(ctx, field, filter, opts...)
}

// EstimatedDocumentCount wraps the native EstimatedDocumentCount collection method.
func (c *Collection) EstimatedDocumentCount(ctx context.Context, opts ...*options.EstimatedDocumentCountOptions) (int64, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.EstimatedDocumentCount")
	defer span.Finish()

	return c.coll.EstimatedDocumentCount(ctx, opts...)
}

// Find wraps the native Find collection method and yields the returned cursor.
func (c *Collection) Find(ctx context.Context, filter interface{}, fn func(csr lungo.ICursor) error, opts ...*options.FindOptions) error {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.Find")
	span.Log("filter", filter)
	defer span.Finish()

	// find
	csr, err := c.coll.Find(ctx, filter, opts...)
	if err != nil {
		return err
	}

	// yield cursor
	err = fn(csr)
	if err != nil {
		_ = csr.Close(ctx)
		return err
	}

	// close cursor
	err = csr.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}

// FindAll wraps the native Find collection method and decodes all documents to
// the provided slice.
func (c *Collection) FindAll(ctx context.Context, slicePtr interface{}, filter interface{}, opts ...*options.FindOptions) error {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.FindAll")
	span.Log("filter", filter)
	defer span.Finish()

	// find
	csr, err := c.coll.Find(ctx, filter, opts...)
	if err != nil {
		return err
	}

	// decode all documents
	err = csr.All(ctx, slicePtr)
	if err != nil {
		_ = csr.Close(ctx)
		return err
	}

	return nil
}

// FindIter wraps the native Find collection method and calls the provided
// callback with the decode method until ErrBreak or an error is returned or the
// cursor has been exhausted.
func (c *Collection) FindIter(ctx context.Context, filter interface{}, fn func(decode func(interface{}) error) error, opts ...*options.FindOptions) error {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.FindIter")
	span.Log("filter", filter)
	defer span.Finish()

	// find
	csr, err := c.coll.Find(ctx, filter, opts...)
	if err != nil {
		return err
	}

	// iterate over all documents
	for csr.Next(ctx) {
		err = fn(csr.Decode)
		if err == ErrBreak {
			break
		} else if err != nil {
			_ = csr.Close(ctx)
			return err
		}
	}

	// close cursor
	err = csr.Close(ctx)
	if err != nil {
		return err
	}

	return nil
}

// FindOne wraps the native FindOne collection method.
func (c *Collection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) lungo.ISingleResult {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.FindOne")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.FindOne(ctx, filter, opts...)
}

// FindOneAndDelete wraps the native FindOneAndDelete collection method.
func (c *Collection) FindOneAndDelete(ctx context.Context, filter interface{}, opts ...*options.FindOneAndDeleteOptions) lungo.ISingleResult {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.FindOneAndDelete")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.FindOneAndDelete(ctx, filter, opts...)
}

// FindOneAndReplace wraps the native FindOneAndReplace collection method.
func (c *Collection) FindOneAndReplace(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.FindOneAndReplaceOptions) lungo.ISingleResult {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.FindOneAndReplace")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.FindOneAndReplace(ctx, filter, replacement, opts...)
}

// FindOneAndUpdate wraps the native FindOneAndUpdate collection method.
func (c *Collection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) lungo.ISingleResult {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.FindOneAndUpdate")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.FindOneAndUpdate(ctx, filter, update, opts...)
}

// InsertMany wraps the native InsertMany collection method.
func (c *Collection) InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.InsertMany")
	span.Log("count", len(documents))
	defer span.Finish()

	return c.coll.InsertMany(ctx, documents, opts...)
}

// InsertOne wraps the native InsertOne collection method.
func (c *Collection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.InsertOne")
	defer span.Finish()

	return c.coll.InsertOne(ctx, document, opts...)
}

// ReplaceOne wraps the native ReplaceOne collection method.
func (c *Collection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.ReplaceOne")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.ReplaceOne(ctx, filter, replacement, opts...)
}

// UpdateMany wraps the native UpdateMany collection method.
func (c *Collection) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.UpdateMany")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.UpdateMany(ctx, filter, update, opts...)
}

// UpdateOne wraps the native UpdateOne collection method.
func (c *Collection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	// track
	ctx, span := cinder.Track(ctx, "coal/Collection.UpdateOne")
	span.Log("filter", filter)
	defer span.Finish()

	return c.coll.UpdateOne(ctx, filter, update, opts...)
}
