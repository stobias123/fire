package coal

import (
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

type index struct {
	coll  string
	index mgo.Index
}

// An Indexer can be used to manage indexes for models.
type Indexer struct {
	indexes []index
}

// NewIndexer returns a new indexer.
func NewIndexer() *Indexer {
	return &Indexer{}
}

// Add will add an index to the internal index list. Fields that are prefixed
// with a dash will result in an descending index. See the MongoDB documentation
// for more details.
func (i *Indexer) Add(model Model, unique bool, expireAfter time.Duration, fields ...string) {
	// construct key from fields
	var key []string
	for _, f := range fields {
		key = append(key, F(model, f))
	}

	// add index
	i.AddRaw(C(model), mgo.Index{
		Key:         key,
		Unique:      unique,
		ExpireAfter: expireAfter,
		Background:  true,
	})
}

// AddPartial is similar to Add except that it adds a partial index.
func (i *Indexer) AddPartial(model Model, unique bool, expireAfter time.Duration, fields []string, filter bson.M) {
	// construct key from fields
	var key []string
	for _, f := range fields {
		key = append(key, F(model, f))
	}

	// add index
	i.AddRaw(C(model), mgo.Index{
		Key:           key,
		Unique:        unique,
		ExpireAfter:   expireAfter,
		PartialFilter: filter,
		Background:    true,
	})
}

// AddRaw will add a raw mgo.Index to the internal index list.
func (i *Indexer) AddRaw(coll string, idx mgo.Index) {
	i.indexes = append(i.indexes, index{
		coll:  coll,
		index: idx,
	})
}

// Ensure will ensure that the required indexes exist. It may fail early if some
// of the indexes are already existing and do not match the supplied index.
func (i *Indexer) Ensure(store *Store) error {
	// copy store
	s := store.Copy()
	defer s.Close()

	// go through all raw indexes
	for _, i := range i.indexes {
		// ensure single index
		err := s.DB().C(i.coll).EnsureIndex(i.index)
		if err != nil {
			return err
		}
	}

	return nil
}
