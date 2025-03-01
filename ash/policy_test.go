package ash

import (
	"net/http"
	"testing"

	"github.com/256dpi/serve"
	"github.com/256dpi/xo"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/256dpi/fire"
	"github.com/256dpi/fire/coal"
	"github.com/256dpi/fire/flame"
	"github.com/256dpi/fire/heat"
	"github.com/256dpi/fire/roast"
	"github.com/256dpi/fire/stick"
)

type exampleModel struct {
	coal.Base          `json:"-" bson:",inline" coal:"examples"`
	Mode               string  `json:"mode"`
	User               coal.ID `json:"user" coal:"user:users"`
	stick.NoValidation `json:"-" bson:"-"`
}

func (m *exampleModel) Foo() string {
	return "foo"
}

func TestPolicy(t *testing.T) {
	store := coal.MustOpen(nil, "test", xo.Panic)

	notary := heat.NewNotary("test", heat.MustRand(16))

	policy := flame.DefaultPolicy(notary)
	policy.Grants = flame.StaticGrants(true, false, false, false, false)

	auth := flame.NewAuthenticator(store, policy, xo.Panic)

	var magicID coal.ID

	group := fire.NewGroup(xo.Panic)
	group.Add(&fire.Controller{
		Store: store,
		Model: &exampleModel{},
		Properties: map[string]string{
			"Foo": "foo",
		},
		Authorizers: fire.L{
			// basic
			flame.Callback(false),

			// identity
			IdentifyPublic(),
			IdentifyToken(nil, func(info *flame.AuthInfo) Identity {
				return info.ResourceOwner.(*flame.User)
			}),

			// select policy
			SelectPublic(func() *Policy {
				return &Policy{
					Access: None,
				}
			}),
			SelectMatch(&flame.User{}, func(identity Identity) *Policy {
				user := identity.(*flame.User)
				return &Policy{
					Access: Full,
					Actions: map[string]bool{
						"c1": true,
						"r1": true,
					},
					Fields: AccessTable{
						"User": Full,
						"Mode": Full,
					},
					GetFilter: func(ctx *fire.Context) bson.M {
						return bson.M{
							"Mode": bson.M{
								"$ne": "hidden",
							},
						}
					},
					VerifyID: func(ctx *fire.Context, id coal.ID) Access {
						if id == magicID {
							return Read
						}
						return Full
					},
					VerifyModel: func(ctx *fire.Context, model coal.Model) Access {
						example := model.(*exampleModel)
						if example.User == user.ID() {
							return Full
						}
						return Read
					},
					VerifyCreate: func(ctx *fire.Context, model coal.Model) bool {
						return model.(*exampleModel).User == user.ID()
					},
					VerifyUpdate: func(ctx *fire.Context, model coal.Model) bool {
						return model.(*exampleModel).Mode != "invalid"
					},
					GetFields: func(ctx *fire.Context, model coal.Model) AccessTable {
						example := model.(*exampleModel)
						if example.User == user.ID() {
							return AccessTable{
								"User": Full,
								"Mode": Full,
							}
						}

						return AccessTable{
							"User": Find,
						}
					},
					GetProperties: func(ctx *fire.Context, model coal.Model) AccessTable {
						return AccessTable{
							"Foo": List,
						}
					},
				}
			}),

			// execute policy
			Execute(),
		},
		CollectionActions: fire.M{
			"c1": fire.A("c1", []string{"POST"}, 128, func(ctx *fire.Context) error {
				return nil
			}),
			"c2": fire.A("c1", []string{"POST"}, 128, func(ctx *fire.Context) error {
				return nil
			}),
		},
		ResourceActions: fire.M{
			"r1": fire.A("c1", []string{"POST"}, 128, func(ctx *fire.Context) error {
				return nil
			}),
			"r2": fire.A("c1", []string{"POST"}, 128, func(ctx *fire.Context) error {
				return nil
			}),
		},
	})

	api := serve.Compose(
		auth.Authorizer(nil, false, true, true),
		group.Endpoint("/api/"),
	)

	handler := http.NewServeMux()
	handler.Handle("/api/", api)
	handler.Handle("/auth/", auth.Endpoint("/auth/"))

	tester := roast.NewTester(roast.Config{
		Store:         store,
		Models:        []coal.Model{&exampleModel{}},
		Handler:       handler,
		DataNamespace: "api",
		AuthNamespace: "auth",
		TokenEndpoint: "token",
	})

	user := tester.Insert(&flame.User{
		Name:     "Test",
		Email:    "test@example.org",
		Password: "1234",
	}).(*flame.User)

	app := tester.Insert(&flame.Application{
		Name: "Main",
		Key:  "main",
	}).(*flame.Application)

	tester.Insert(&exampleModel{
		Mode: "hidden",
		User: coal.New(),
	})

	example1 := tester.Insert(&exampleModel{
		Mode: "foo",
		User: coal.New(),
	}).(*exampleModel)

	example2 := tester.Insert(&exampleModel{
		Mode: "bar",
		User: user.ID(),
	}).(*exampleModel)

	// public access
	tester.ListError(t, &exampleModel{}, roast.AccessDenied)

	// authenticate
	tester.Authenticate(app.Key, user.Email, "1234")

	// private access
	tester.List(t, &exampleModel{}, []coal.Model{
		&exampleModel{
			Base: coal.B(example1.ID()),
		},
		&exampleModel{
			Base: coal.B(example2.ID()),
			Mode: "bar",
			User: example2.User,
		},
	})

	tester.Find(t, example1, &exampleModel{
		Base: coal.B(example1.ID()),
		User: example1.User,
	})
	tester.Find(t, example2, &exampleModel{
		Base: coal.B(example2.ID()),
		Mode: "bar",
		User: example2.User,
	})

	tester.CreateError(t, &exampleModel{
		User: coal.New(),
	}, roast.AccessDenied)
	tester.Create(t, &exampleModel{
		User: user.ID(),
	}, &exampleModel{
		User: user.ID(),
	}, nil)

	tester.UpdateError(t, example1, roast.AccessDenied)
	tester.Update(t, example2, example2, nil)

	example2.Mode = "invalid"
	tester.UpdateError(t, example2, roast.AccessDenied)
	example2.Mode = "bar"

	magicID = example2.ID()
	tester.DeleteError(t, example2, roast.AccessDenied)
	magicID = ""
	tester.Delete(t, example2, nil)

	code, _ := tester.Call(t, tester.URL("examples", "c1"), nil, nil)
	assert.Equal(t, http.StatusOK, code)

	code, _ = tester.Call(t, tester.URL("examples", "c2"), nil, nil)
	assert.Equal(t, http.StatusUnauthorized, code)

	code, _ = tester.Call(t, tester.URL("examples", example1.ID(), "r1"), nil, nil)
	assert.Equal(t, http.StatusOK, code)

	code, _ = tester.Call(t, tester.URL("examples", example1.ID(), "r2"), nil, nil)
	assert.Equal(t, http.StatusUnauthorized, code)
}
