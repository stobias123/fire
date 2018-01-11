package ash

import (
	"github.com/256dpi/fire"

	"gopkg.in/mgo.v2/bson"
)

// E is a short-hand function to create an enforcer.
func E(name string, m fire.Matcher, h fire.Handler) *Enforcer {
	return fire.C(name, m, h)
}

// An Enforcer is returned by an Authorizer to enforce the previously inspected
// Authorization.
//
// Enforcers should only return errors if the operation is clearly not allowed for
// the presented candidate and that this information is general knowledge (e.g.
// API documentation). In order to prevent the leakage of implementation details
// the enforcer should mutate the context's Query field to hide existing data
// from the candidate.
type Enforcer = fire.Callback

// GrantAccess will enforce the authorization without any changes to the
// context. It should be used if the presented candidate has full access to the
// data (.e.g a superuser).
func GrantAccess() *Enforcer {
	return E("ash/GrantAccess", nil, func(_ *fire.Context) error {
		return nil
	})
}

// DenyAccess will enforce the authorization by directly returning an access
// denied error. It should be used if the operation should not be authorized in
// any case (.e.g a candidate accessing a resource he has clearly no access to).
//
// Note: Usually access is denied by returning no enforcer. This enforcer should
// only be returned to immediately stop the authorization process and prevent
// other enforcers from authorizing the operation.
func DenyAccess() *Enforcer {
	return E("ash/DenyAccess", nil, func(_ *fire.Context) error {
		return fire.ErrAccessDenied
	})
}

// FilterQuery will enforce the authorization by adding the passed filters to the
// Filter query of the context. It should be used if the candidate is allowed to
// access the resource in general, but some records should be filtered out.
func FilterQuery(filters bson.M) *Enforcer {
	return E("ash/FilterQuery", nil, func(ctx *fire.Context) error {
		// assign specified filters
		for key, value := range filters {
			ctx.Filter[key] = value
		}

		return nil
	})
}
