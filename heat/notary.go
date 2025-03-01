package heat

import (
	"context"
	"time"

	"github.com/256dpi/xo"

	"github.com/256dpi/fire/coal"
	"github.com/256dpi/fire/stick"
)

// Notary is used to issue and verify tokens from keys.
type Notary struct {
	issuer string
	secret []byte
}

// NewNotary creates a new notary with the specified name and secret. It will
// panic if the name is missing or the specified secret is less than 16 bytes.
func NewNotary(name string, secret []byte) *Notary {
	// check name
	if name == "" {
		panic("heat: missing name")
	}

	// check secret
	if len(secret) < minSecretLen {
		panic("heat: secret too small")
	}

	return &Notary{
		secret: secret,
		issuer: name,
	}
}

// Issue will generate a token from the specified key.
func (n *Notary) Issue(ctx context.Context, key Key) (string, error) {
	// trace
	_, span := xo.Trace(ctx, "heat/Notary.Issue")
	defer span.End()

	// get meta
	meta := GetMeta(key)

	// get base
	base := key.GetBase()

	// ensure id
	if base.ID == "" {
		base.ID = coal.New()
	}

	// ensure issued
	if base.Issued.IsZero() {
		base.Issued = time.Now()
	}

	// ensure expires
	if base.Expires.IsZero() {
		base.Expires = base.Issued.Add(meta.Expiry)
	}

	// validate key
	err := key.Validate()
	if err != nil {
		return "", err
	}

	// get data
	var data stick.Map
	err = data.Marshal(key, stick.JSON)
	if err != nil {
		return "", err
	}

	// issue token
	token, err := Issue(n.secret, n.issuer, meta.Name, RawKey{
		ID:      base.ID,
		Issued:  base.Issued,
		Expires: base.Expires,
		Data:    data,
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

// Verify will verify the specified token and fill the specified key.
func (n *Notary) Verify(ctx context.Context, key Key, token string) error {
	// trace
	_, span := xo.Trace(ctx, "heat/Notary.Verify")
	defer span.End()

	// get meta
	meta := GetMeta(key)

	// verify token
	rawKey, err := Verify(n.secret, n.issuer, meta.Name, token)
	if err != nil {
		return err
	}

	// check id
	kid, err := coal.FromHex(rawKey.ID)
	if err != nil {
		return xo.F("invalid token id")
	}

	// set base
	*key.GetBase() = Base{
		ID:      kid,
		Issued:  rawKey.Issued,
		Expires: rawKey.Expires,
	}

	// assign data
	err = rawKey.Data.Unmarshal(key, stick.JSON)
	if err != nil {
		return err
	}

	// validate key
	err = key.Validate()
	if err != nil {
		return err
	}

	return nil
}
