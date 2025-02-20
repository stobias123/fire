package fire

import (
	"strings"
	"testing"
	"time"

	"github.com/256dpi/xo"

	"github.com/256dpi/fire/coal"
	"github.com/256dpi/fire/stick"
)

type postModel struct {
	coal.Base  `json:"-" bson:",inline" coal:"posts"`
	Title      string       `json:"title" bson:"title"`
	Published  bool         `json:"published"`
	TextBody   string       `json:"text-body" bson:"text_body"`
	Deleted    *time.Time   `json:"-" bson:"deleted_at" coal:"fire-soft-delete"`
	Comments   coal.HasMany `json:"-" bson:"-" coal:"comments:comments:post"`
	Selections coal.HasMany `json:"-" bson:"-" coal:"selections:selections:posts"`
	Note       coal.HasOne  `json:"-" bson:"-" coal:"note:notes:post"`
}

func (p *postModel) Validate() error {
	if p.Title == "error" {
		return xo.SF("validation error")
	}

	return nil
}

func (p *postModel) Virtual() int64 {
	return 42
}

func (p *postModel) VirtualError() (string, error) {
	if p.Title == "virtual-error" {
		return "", xo.SF("virtual error")
	}

	return p.Title, nil
}

func (p *postModel) SetTitle(title string) {
	p.Title = title
}

func (p *postModel) Strings() (string, string) {
	return p.Title, p.TextBody
}

type commentModel struct {
	coal.Base          `json:"-" bson:",inline" coal:"comments"`
	Message            string     `json:"message"`
	Deleted            *time.Time `json:"-" bson:"deleted_at" coal:"fire-soft-delete"`
	Parent             *coal.ID   `json:"-" bson:"parent_id" coal:"parent:comments"`
	Post               coal.ID    `json:"-" bson:"post_id" coal:"post:posts"`
	stick.NoValidation `json:"-" bson:"-"`
}

type selectionModel struct {
	coal.Base          `json:"-" bson:",inline" coal:"selections:selections"`
	Name               string    `json:"name"`
	CreateToken        string    `json:"create-token,omitempty" bson:"create_token" coal:"fire-idempotent-create"`
	UpdateToken        string    `json:"update-token,omitempty" bson:"update_token" coal:"fire-consistent-update"`
	Posts              []coal.ID `json:"-" bson:"post_ids" coal:"posts:posts"`
	stick.NoValidation `json:"-" bson:"-"`
}

type noteModel struct {
	coal.Base          `json:"-" bson:",inline" coal:"notes"`
	Title              string  `json:"title" bson:"title"`
	Post               coal.ID `json:"-" bson:"post_id" coal:"post:posts"`
	stick.NoValidation `json:"-" bson:"-"`
}

type fooModel struct {
	coal.Base          `json:"-" bson:",inline" coal:"foos"`
	String             string    `json:"string"`
	Foo                coal.ID   `json:"-" bson:"foo_id" coal:"foo:foos"`
	OptFoo             *coal.ID  `json:"-" bson:"opt_foo_id" coal:"opt-foo:foos"`
	Foos               []coal.ID `json:"-" bson:"foo_ids" coal:"foos:foos"`
	Bar                coal.ID   `json:"-" bson:"bar_id" coal:"bar:bars"`
	OptBar             *coal.ID  `json:"-" bson:"opt_bar_id" coal:"opt-bar:bars"`
	Bars               []coal.ID `json:"-" bson:"bar_ids" coal:"bars:bars"`
	stick.NoValidation `json:"-" bson:"-"`
}

type barModel struct {
	coal.Base          `json:"-" bson:",inline" coal:"bars"`
	Foo                coal.ID `json:"-" bson:"foo_id" coal:"foo:foos"`
	stick.NoValidation `json:"-" bson:"-"`
}

//var mongoStore = coal.MustConnect("mongodb://0.0.0.0/test-fire", xo.Panic)
var mongoStore = coal.MustConnect("mongodb+srv://skidrow:ZpJfzt1zEGLBQQdH@skidrow.rc4a3ei.mongodb.net/test?retryWrites=true&w=majority", xo.Panic)
var lungoStore = coal.MustOpen(nil, "test-fire", xo.Panic)

var modelList = []coal.Model{&postModel{}, &commentModel{}, &selectionModel{}, &noteModel{}, &fooModel{}, &barModel{}}

func withTester(t *testing.T, fn func(*testing.T, *Tester)) {
	t.Run("Mongo", func(t *testing.T) {
		tester := NewTester(mongoStore, modelList...)
		tester.Clean()
		fn(t, tester)
	})

	t.Run("Lungo", func(t *testing.T) {
		tester := NewTester(lungoStore, modelList...)
		tester.Clean()
		fn(t, tester)
	})
}

func numID(n uint8) coal.ID {
	return ""
}

func linkUnescape(str string) string {
	str = strings.ReplaceAll(str, "%5B", "[")
	str = strings.ReplaceAll(str, "%5D", "]")
	str = strings.ReplaceAll(str, "%2A", "*")
	return strings.ReplaceAll(str, "%2C", ",")
}
