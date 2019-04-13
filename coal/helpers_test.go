package coal

import (
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/stretchr/testify/assert"
)

func TestC(t *testing.T) {
	assert.Equal(t, "posts", C(&postModel{}))
}

func TestF(t *testing.T) {
	assert.Equal(t, "text_body", F(&postModel{}, "TextBody"))
	assert.Equal(t, "-text_body", F(&postModel{}, "-TextBody"))

	assert.PanicsWithValue(t, `coal: field "Foo" not found on "coal.postModel"`, func() {
		F(&postModel{}, "Foo")
	})
}

func TestA(t *testing.T) {
	assert.Equal(t, "text-body", A(&postModel{}, "TextBody"))

	assert.PanicsWithValue(t, `coal: field "Foo" not found on "coal.postModel"`, func() {
		A(&postModel{}, "Foo")
	})
}

func TestR(t *testing.T) {
	assert.Equal(t, "post", R(&commentModel{}, "Post"))

	assert.PanicsWithValue(t, `coal: field "Foo" not found on "coal.postModel"`, func() {
		R(&postModel{}, "Foo")
	})
}

func TestL(t *testing.T) {
	assert.Equal(t, "Title", L(&postModel{}, "foo", true))

	assert.PanicsWithValue(t, `coal: no or multiple fields flagged as "bar" found on "coal.postModel"`, func() {
		L(&postModel{}, "bar", true)
	})

	assert.PanicsWithValue(t, `coal: no or multiple fields flagged as "quz" found on "coal.postModel"`, func() {
		L(&postModel{}, "quz", true)
	})
}

func TestP(t *testing.T) {
	id := bson.NewObjectId()
	assert.Equal(t, &id, P(id))
}

func TestN(t *testing.T) {
	var id *bson.ObjectId
	assert.Equal(t, id, N())
	assert.NotEqual(t, nil, N())
}

func TestT(t *testing.T) {
	t1 := time.Now()
	t2 := T(t1)
	assert.Equal(t, t1, *t2)
}

func TestUnique(t *testing.T) {
	id1 := bson.NewObjectId()
	id2 := bson.NewObjectId()

	assert.Equal(t, []bson.ObjectId{id1}, Unique([]bson.ObjectId{id1}))
	assert.Equal(t, []bson.ObjectId{id1}, Unique([]bson.ObjectId{id1, id1}))
	assert.Equal(t, []bson.ObjectId{id1, id2}, Unique([]bson.ObjectId{id1, id2, id1}))
	assert.Equal(t, []bson.ObjectId{id1, id2}, Unique([]bson.ObjectId{id1, id2, id1, id2}))
}

func TestContains(t *testing.T) {
	a := bson.NewObjectId()
	b := bson.NewObjectId()
	c := bson.NewObjectId()
	d := bson.NewObjectId()

	assert.True(t, Contains([]bson.ObjectId{a, b, c}, a))
	assert.True(t, Contains([]bson.ObjectId{a, b, c}, b))
	assert.True(t, Contains([]bson.ObjectId{a, b, c}, c))
	assert.False(t, Contains([]bson.ObjectId{a, b, c}, d))
}

func TestIncludes(t *testing.T) {
	a := bson.NewObjectId()
	b := bson.NewObjectId()
	c := bson.NewObjectId()
	d := bson.NewObjectId()

	assert.True(t, Includes([]bson.ObjectId{a, b, c}, []bson.ObjectId{a}))
	assert.True(t, Includes([]bson.ObjectId{a, b, c}, []bson.ObjectId{a, b}))
	assert.True(t, Includes([]bson.ObjectId{a, b, c}, []bson.ObjectId{a, b, c}))
	assert.False(t, Includes([]bson.ObjectId{a, b, c}, []bson.ObjectId{a, b, c, d}))
	assert.False(t, Includes([]bson.ObjectId{a, b, c}, []bson.ObjectId{d}))
}

func TestRequire(t *testing.T) {
	assert.NotPanics(t, func() {
		Require(&postModel{}, "foo")
	})

	assert.PanicsWithValue(t, `coal: no or multiple fields flagged as "bar" found on "coal.postModel"`, func() {
		Require(&postModel{}, "bar")
	})

	assert.PanicsWithValue(t, `coal: no or multiple fields flagged as "quz" found on "coal.postModel"`, func() {
		Require(&postModel{}, "quz")
	})
}
