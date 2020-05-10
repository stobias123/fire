package blaze

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/256dpi/fire"
	"github.com/256dpi/fire/coal"
)

func TestAddFileIndexes(t *testing.T) {
	withTester(t, func(t *testing.T, tester *fire.Tester) {
		catalog := coal.NewCatalog()
		AddFileIndexes(catalog)
		assert.NoError(t, catalog.EnsureIndexes(tester.Store))
		assert.NoError(t, catalog.EnsureIndexes(tester.Store))
	})
}

func TestBlobValidate(t *testing.T) {
	blob := &Blob{}

	err := blob.Validate("foo")
	assert.Error(t, err)
	assert.Equal(t, "foo type invalid", err.Error())

	blob.Type = "foo/bar"

	err = blob.Validate("foo")
	assert.Error(t, err)
	assert.Equal(t, "foo bytes missing", err.Error())

	err = blob.Validate("foo", "bar/foo")
	assert.Error(t, err)
	assert.Equal(t, "foo type unallowed", err.Error())

	blob.Bytes = []byte("foo")

	err = blob.Validate("foo")
	assert.NoError(t, err)
}

func TestLinkValidate(t *testing.T) {
	link := &Link{}

	err := link.Validate("foo")
	assert.Error(t, err)
	assert.Equal(t, "foo type invalid", err.Error())

	link.Type = "foo/bar"

	err = link.Validate("foo")
	assert.Error(t, err)
	assert.Equal(t, "foo zero length", err.Error())

	err = link.Validate("foo", "bar/foo")
	assert.Error(t, err)
	assert.Equal(t, "foo type unallowed", err.Error())

	link.Length = 12

	err = link.Validate("foo")
	assert.Equal(t, "foo invalid file", err.Error())

	link.File = coal.P(coal.ID{})

	err = link.Validate("foo")
	assert.Equal(t, "foo invalid file", err.Error())

	link.File = coal.P(coal.New())

	err = link.Validate("foo")
	assert.NoError(t, err)
}
