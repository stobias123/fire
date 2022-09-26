package coal

import (
	"time"

	"github.com/256dpi/xo"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ID is shorthand type for the object id.
type ID = string

// New will return a new object id, optionally using a custom timestamp.
func New(timestamp ...time.Time) ID {
	// check timestamp
	if len(timestamp) > 0 {
		return primitive.NewObjectIDFromTimestamp(timestamp[0]).Hex()
	}

	return primitive.NewObjectID().Hex()
}

// IsHex will assess whether the provided string is a valid hex encoded
// object id.
func IsHex(str string) bool {
	_, err := FromHex(str)
	return err == nil
}

// FromHex will convert the provided string to an object id.
func FromHex(str string) (ID, error) {
	id, err := primitive.ObjectIDFromHex(str)
	return id.Hex(), xo.W(err)
}

// MustFromHex is a vestage of when id was the ObjectID type.
// It remains for compat reasons.
// I'm not using it.
func MustFromHex(str string) string {
	return str
}
