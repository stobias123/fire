package fire

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/256dpi/jsonapi/v2"
	"github.com/256dpi/serve"
	"github.com/256dpi/xo"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/256dpi/fire/coal"
	"github.com/256dpi/fire/stick"
)

func TestBasicOperations(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// get empty list of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data":[],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		var id string

		// attempt to create post with missing document
		tester.Request("POST", "posts", `{}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "missing document"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create post with invalid type
		tester.Request("POST", "posts", `{
			"data": {
				"type": "foo"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create post with invalid id
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"id": "`+coal.New()+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "unnecessary resource id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create post with invalid attribute
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"foo": "bar"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "invalid attribute",
					"source": {
						"pointer": "/data/attributes/foo"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create post
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+id+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/comments",
								"related": "/posts/`+id+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/selections",
								"related": "/posts/`+id+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+id+`/relationships/note",
								"related": "/posts/`+id+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get list of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+id+`",
						"attributes": {
							"title": "Post 1",
							"published": false,
							"text-body": ""
						},
						"relationships": {
							"comments": {
								"data": [],
								"links": {
									"self": "/posts/`+id+`/relationships/comments",
									"related": "/posts/`+id+`/comments"
								}
							},
							"selections": {
								"data": [],
								"links": {
									"self": "/posts/`+id+`/relationships/selections",
									"related": "/posts/`+id+`/selections"
								}
							},
							"note": {
								"data": null,
								"links": {
									"self": "/posts/`+id+`/relationships/note",
									"related": "/posts/`+id+`/note"
								}
							}
						}
					}
				],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to update post with missing document
		tester.Request("PATCH", "posts/"+id, `{}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "missing document"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to update post with invalid type
		tester.Request("PATCH", "posts/"+id, `{
			"data": {
				"type": "foo",
				"id": "`+id+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to update post with invalid id
		tester.Request("PATCH", "posts/"+id, `{
			"data": {
				"type": "posts",
				"id": "`+coal.New()+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "resource id mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to update post with invalid id
		tester.Request("PATCH", "posts/foo", `{
			"data": {
				"type": "posts",
				"id": "`+id+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "invalid resource id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to update post with invalid attribute
		tester.Request("PATCH", "posts/"+id, `{
			"data": {
				"type": "posts",
				"id": "`+id+`",
				"attributes": {
					"foo": "bar"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "invalid attribute",
					"source": {
						"pointer": "/data/attributes/foo"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update post
		tester.Request("PATCH", "posts/"+id, `{
			"data": {
				"type": "posts",
				"id": "`+id+`",
				"attributes": {
					"text-body": "Post 1 Text"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+id+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": "Post 1 Text"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/comments",
								"related": "/posts/`+id+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/selections",
								"related": "/posts/`+id+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+id+`/relationships/note",
								"related": "/posts/`+id+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to get single post with invalid id
		tester.Request("GET", "posts/foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "invalid resource id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to get not existing post
		tester.Request("GET", "posts/"+coal.New(), "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNotFound, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "404",
					"title": "not found",
					"detail": "resource not found"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get single post
		tester.Request("GET", "posts/"+id, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+id+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": "Post 1 Text"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/comments",
								"related": "/posts/`+id+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/selections",
								"related": "/posts/`+id+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+id+`/relationships/note",
								"related": "/posts/`+id+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to delete post with invalid id
		tester.Request("DELETE", "posts/foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "invalid resource id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete post
		tester.Request("DELETE", "posts/"+id, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNoContent, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, "", r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty list of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data":[],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestHasOneRelationships(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create new post
		post := tester.Insert(&postModel{
			Title: "Post 2",
		}).ID()

		// get single post
		tester.Request("GET", "posts/"+post, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post+`",
					"attributes": {
						"title": "Post 2",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post+`/relationships/comments",
								"related": "/posts/`+post+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post+`/relationships/selections",
								"related": "/posts/`+post+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post+`/relationships/note",
								"related": "/posts/`+post+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get invalid relation
		tester.Request("GET", "posts/"+post+"/foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get invalid relationship
		tester.Request("GET", "posts/"+post+"/relationships/foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty unset related note
		tester.Request("GET", "posts/"+post+"/note", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": null,
				"links": {
					"self": "/posts/`+post+`/note"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create related note with invalid relationship
		tester.Request("POST", "notes", `{
			"data": {
				"type": "notes",
				"attributes": {
					"title": "Note 2"
				},
				"relationships": {
					"foo": {
						"data": {
							"type": "foo",
							"id": "`+post+`"
						}
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship",
					"source": {
						"pointer": "/data/relationships/foo"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create related note with invalid type
		tester.Request("POST", "notes", `{
			"data": {
				"type": "notes",
				"attributes": {
					"title": "Note 2"
				},
				"relationships": {
					"post": {
						"data": {
							"type": "foo",
							"id": "`+post+`"
						}
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		var note string

		// create related note
		tester.Request("POST", "notes", `{
			"data": {
				"type": "notes",
				"attributes": {
					"title": "Note 2"
				},
				"relationships": {
					"post": {
						"data": {
							"type": "posts",
							"id": "`+post+`"
						}
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			note = tester.FindLast(&noteModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "notes",
					"id": "`+note+`",
					"attributes": {
						"title": "Note 2"
					},
					"relationships": {
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/notes/`+note+`/relationships/post",
								"related": "/notes/`+note+`/post"
							}
						}
					}
				},
				"links": {
					"self": "/notes/`+note+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get related note
		tester.Request("GET", "posts/"+post+"/note", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "notes",
					"id": "`+note+`",
					"attributes": {
						"title": "Note 2"
					},
					"relationships": {
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/notes/`+note+`/relationships/post",
								"related": "/notes/`+note+`/post"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post+`/note"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get note relationship
		tester.Request("GET", "posts/"+post+"/relationships/note", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "notes",
					"id": "`+note+`"
				},
				"links": {
					"self": "/posts/`+post+`/relationships/note",
					"related": "/posts/`+post+`/note"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestHasManyRelationships(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create existing post & comment
		existingPost := tester.Insert(&postModel{
			Title: "Post 1",
		})
		tester.Insert(&commentModel{
			Message: "Comment 1",
			Post:    existingPost.ID(),
		})

		// create new post
		post := tester.Insert(&postModel{
			Title: "Post 2",
		}).ID()

		// get single post
		tester.Request("GET", "posts/"+post, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post+`",
					"attributes": {
						"title": "Post 2",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post+`/relationships/comments",
								"related": "/posts/`+post+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post+`/relationships/selections",
								"related": "/posts/`+post+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post+`/relationships/note",
								"related": "/posts/`+post+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty list of related comments
		tester.Request("GET", "posts/"+post+"/comments", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data":[],
				"links": {
					"self": "/posts/`+post+`/comments"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create related comment with invalid type
		tester.Request("POST", "comments", `{
			"data": {
				"type": "comments",
				"attributes": {
					"message": "Comment 2"
				},
				"relationships": {
					"post": {
						"data": {
							"type": "foo",
							"id": "`+post+`"
						}
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		var comment string

		// create related comment
		tester.Request("POST", "comments", `{
			"data": {
				"type": "comments",
				"attributes": {
					"message": "Comment 2"
				},
				"relationships": {
					"post": {
						"data": {
							"type": "posts",
							"id": "`+post+`"
						}
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			comment = tester.FindLast(&commentModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "comments",
					"id": "`+comment+`",
					"attributes": {
						"message": "Comment 2"
					},
					"relationships": {
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/comments/`+comment+`/relationships/post",
								"related": "/comments/`+comment+`/post"
							}
						},
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment+`/relationships/parent",
								"related": "/comments/`+comment+`/parent"
							}
						}
					}
				},
				"links": {
					"self": "/comments/`+comment+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get list of related comments
		tester.Request("GET", "posts/"+post+"/comments", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "comments",
						"id": "`+comment+`",
						"attributes": {
							"message": "Comment 2"
						},
						"relationships": {
							"post": {
								"data": {
									"type": "posts",
									"id": "`+post+`"
								},
								"links": {
									"self": "/comments/`+comment+`/relationships/post",
									"related": "/comments/`+comment+`/post"
								}
							},
							"parent": {
								"data": null,
								"links": {
									"self": "/comments/`+comment+`/relationships/parent",
									"related": "/comments/`+comment+`/parent"
								}
							}
						}
					}
				],
				"links": {
					"self": "/posts/`+post+`/comments"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get comments relationship
		tester.Request("GET", "posts/"+post+"/relationships/comments", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "comments",
						"id": "`+comment+`"
					}
				],
				"links": {
					"self": "/posts/`+post+`/relationships/comments",
					"related": "/posts/`+post+`/comments"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestToOneRelationships(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create posts
		post1 := tester.Insert(&postModel{
			Title: "Post 1",
		}).ID()
		post2 := tester.Insert(&postModel{
			Title: "Post 2",
		}).ID()

		// create comment
		comment1 := tester.Insert(&commentModel{
			Message: "Comment 1",
			Post:    coal.MustFromHex(post1),
		}).ID()

		var comment2 string

		// create relating comment
		tester.Request("POST", "comments", `{
			"data": {
				"type": "comments",
				"attributes": {
					"message": "Comment 2"
				},
				"relationships": {
					"post": {
						"data": {
							"type": "posts",
							"id": "`+post1+`"
						}
					},
					"parent": {
						"data": {
							"type": "comments",
							"id": "`+comment1+`"
						}
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			comment2 = tester.FindLast(&commentModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "comments",
					"id": "`+comment2+`",
					"attributes": {
						"message": "Comment 2"
					},
					"relationships": {
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post1+`"
							},
							"links": {
								"self": "/comments/`+comment2+`/relationships/post",
								"related": "/comments/`+comment2+`/post"
							}
						},
						"parent": {
							"data": {
								"type": "comments",
								"id": "`+comment1+`"
							},
							"links": {
								"self": "/comments/`+comment2+`/relationships/parent",
								"related": "/comments/`+comment2+`/parent"
							}
						}
					}
				},
				"links": {
					"self": "/comments/`+comment2+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get related post
		tester.Request("GET", "comments/"+comment2+"/post", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [
								{
									"type": "comments",
									"id": "`+comment1+`"
								},
								{
									"type": "comments",
									"id": "`+comment2+`"
								}
							],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/comments/`+comment2+`/post"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get post relationship
		tester.Request("GET", "comments/"+comment2+"/relationships/post", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`"
				},
				"links": {
					"self": "/comments/`+comment2+`/relationships/post",
					"related": "/comments/`+comment2+`/post"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to replace invalid relationship
		tester.Request("PATCH", "comments/"+comment2+"/relationships/foo", `{
			"data": {
				"type": "posts",
				"id": "`+post2+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to replace post relationship with invalid type
		tester.Request("PATCH", "comments/"+comment2+"/relationships/post", `{
			"data": {
				"type": "foo",
				"id": "`+post2+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to replace post relationship with invalid id
		tester.Request("PATCH", "comments/"+comment2+"/relationships/post", `{
			"data": {
				"type": "posts",
				"id": "foo"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// replace post relationship
		tester.Request("PATCH", "comments/"+comment2+"/relationships/post", `{
			"data": {
				"type": "posts",
				"id": "`+post2+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post2+`"
				},
				"links": {
					"self": "/comments/`+comment2+`/relationships/post",
					"related": "/comments/`+comment2+`/post"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get replaced post relationship
		tester.Request("GET", "comments/"+comment2+"/relationships/post", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post2+`"
				},
				"links": {
					"self": "/comments/`+comment2+`/relationships/post",
					"related": "/comments/`+comment2+`/post"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get existing parent relationship
		tester.Request("GET", "comments/"+comment2+"/relationships/parent", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "comments",
					"id": "`+comment1+`"
				},
				"links": {
					"self": "/comments/`+comment2+`/relationships/parent",
					"related": "/comments/`+comment2+`/parent"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get existing related resource
		tester.Request("GET", "comments/"+comment2+"/parent", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "comments",
					"id": "`+comment1+`",
					"attributes": {
						"message": "Comment 1"
					},
					"relationships": {
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment1+`/relationships/parent",
								"related": "/comments/`+comment1+`/parent"
							}
						},
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post1+`"
							},
							"links": {
								"self": "/comments/`+comment1+`/relationships/post",
								"related": "/comments/`+comment1+`/post"
							}
						}
					}
				},
				"links": {
					"self": "/comments/`+comment2+`/parent"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// unset parent relationship
		tester.Request("PATCH", "comments/"+comment2+"/relationships/parent", `{
			"data": null
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": null,
				"links": {
					"self": "/comments/`+comment2+`/relationships/parent",
					"related": "/comments/`+comment2+`/parent"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// fetch unset parent relationship
		tester.Request("GET", "comments/"+comment2+"/relationships/parent", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": null,
				"links": {
					"self": "/comments/`+comment2+`/relationships/parent",
					"related": "/comments/`+comment2+`/parent"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// fetch unset related resource
		tester.Request("GET", "comments/"+comment2+"/parent", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": null,
				"links": {
					"self": "/comments/`+comment2+`/parent"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestToManyRelationships(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create posts
		post1 := tester.Insert(&postModel{
			Title: "Post 1",
		}).ID()
		post2 := tester.Insert(&postModel{
			Title: "Post 2",
		}).ID()
		post3 := tester.Insert(&postModel{
			Title: "Post 3",
		}).ID()

		var selection string

		// create selection
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {
					"name": "Selection 1"
				},
				"relationships": {
					"posts": {
						"data": [
							{
								"type": "posts",
								"id": "`+post1+`"
							},
							{
								"type": "posts",
								"id": "`+post2+`"
							}
						]
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			selection = tester.FindLast(&selectionModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "selections",
					"id": "`+selection+`",
					"attributes": {
						"name": "Selection 1"
					},
					"relationships": {
						"posts": {
							"data": [
								{
									"type": "posts",
									"id": "`+post1+`"
								},
								{
									"type": "posts",
									"id": "`+post2+`"
								}
							],
							"links": {
								"self": "/selections/`+selection+`/relationships/posts",
								"related": "/selections/`+selection+`/posts"
							}
						}
					}
				},
				"links": {
					"self": "/selections/`+selection+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get related post
		tester.Request("GET", "selections/"+selection+"/posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post1+`",
						"attributes": {
							"title": "Post 1",
							"published": false,
							"text-body": ""
						},
						"relationships": {
							"comments": {
								"data": [],
								"links": {
									"self": "/posts/`+post1+`/relationships/comments",
									"related": "/posts/`+post1+`/comments"
								}
							},
							"selections": {
								"data": [
									{
										"type": "selections",
										"id": "`+selection+`"
									}
								],
								"links": {
									"self": "/posts/`+post1+`/relationships/selections",
									"related": "/posts/`+post1+`/selections"
								}
							},
							"note": {
								"data": null,
								"links": {
									"self": "/posts/`+post1+`/relationships/note",
									"related": "/posts/`+post1+`/note"
								}
							}
						}
					},
					{
						"type": "posts",
						"id": "`+post2+`",
						"attributes": {
							"title": "Post 2",
							"published": false,
							"text-body": ""
						},
						"relationships": {
							"comments": {
								"data": [],
								"links": {
									"self": "/posts/`+post2+`/relationships/comments",
									"related": "/posts/`+post2+`/comments"
								}
							},
							"selections": {
								"data": [
									{
										"type": "selections",
										"id": "`+selection+`"
									}
								],
								"links": {
									"self": "/posts/`+post2+`/relationships/selections",
									"related": "/posts/`+post2+`/selections"
								}
							},
							"note": {
								"data": null,
								"links": {
									"self": "/posts/`+post2+`/relationships/note",
									"related": "/posts/`+post2+`/note"
								}
							}
						}
					}
				],
				"links": {
					"self": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get posts relationship
		tester.Request("GET", "selections/"+selection+"/relationships/posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post1+`"
					},
					{
						"type": "posts",
						"id": "`+post2+`"
					}
				],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to replace posts relationship with invalid type
		tester.Request("PATCH", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "foo",
					"id": "`+post3+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to replace posts relationship with invalid id
		tester.Request("PATCH", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "foo"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// unset posts relationship
		tester.Request("PATCH", "selections/"+selection+"/relationships/posts", `{
			"data": null
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// replace posts relationship
		tester.Request("PATCH", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "`+post3+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post3+`"
					}
				],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get updated posts relationship
		tester.Request("GET", "selections/"+selection+"/relationships/posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post3+`"
					}
				],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to add to invalid relationship
		tester.Request("POST", "selections/"+selection+"/relationships/foo", `{
			"data": [
				{
					"type": "posts",
					"id": "`+post1+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to add posts relationship with invalid type
		tester.Request("POST", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "foo",
					"id": "`+post1+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to add posts relationship with invalid id
		tester.Request("POST", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "foo"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// add posts relationship
		tester.Request("POST", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "`+post1+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post3+`"
					},
					{
						"type": "posts",
						"id": "`+post1+`"
					}
				],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// add existing id to posts relationship
		tester.Request("POST", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "`+post1+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post3+`"
					},
					{
						"type": "posts",
						"id": "`+post1+`"
					}
				],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get posts relationship
		tester.Request("GET", "selections/"+selection+"/relationships/posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post3+`"
					},
					{
						"type": "posts",
						"id": "`+post1+`"
					}
				],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to remove from invalid relationship
		tester.Request("DELETE", "selections/"+selection+"/relationships/foo", `{
			"data": [
				{
					"type": "posts",
					"id": "`+post3+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to remove from posts relationships with invalid type
		tester.Request("DELETE", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "foo",
					"id": "`+post3+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "resource type mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to remove from posts relationships with invalid id
		tester.Request("DELETE", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "foo"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid relationship id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// remove from posts relationships
		tester.Request("DELETE", "selections/"+selection+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "`+post3+`"
				},
				{
					"type": "posts",
					"id": "`+post1+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty posts relationship
		tester.Request("GET", "selections/"+selection+"/relationships/posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [],
				"links": {
					"self": "/selections/`+selection+`/relationships/posts",
					"related": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty list of related posts
		tester.Request("GET", "selections/"+selection+"/posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data":[],
				"links": {
					"self": "/selections/`+selection+`/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty list of related selections
		tester.Request("GET", "posts/"+post1+"/selections", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data":[],
				"links": {
					"self": "/posts/`+post1+`/selections"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestModelValidation(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create post
		post := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
		})

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "error"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "validation error"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post, `{
			"data": {
				"type": "posts",
				"id": "`+post+`",
				"attributes": {
					"title": "error"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "validation error"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete
		tester.Request("DELETE", "/posts/"+post, "{}", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNoContent, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, "", r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestSupported(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model:     &commentModel{},
			Supported: Except(List),
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// attempt list comments
		tester.Request("GET", "comments", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusMethodNotAllowed, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "405",
					"title": "method not allowed",
					"detail": "unsupported operation"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestFiltering(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model:   &postModel{},
			Filters: []string{"Title", "Published"},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model:   &selectionModel{},
			Filters: []string{"Posts"},
		}, &Controller{
			Model:   &noteModel{},
			Filters: []string{"Post"},
		})

		// create posts
		post1 := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()
		post2 := tester.Insert(&postModel{
			Title:     "post-2",
			Published: false,
		}).ID()
		post3 := tester.Insert(&postModel{
			Title:     "post-3",
			Published: true,
		}).ID()

		// create selections
		selection := tester.Insert(&selectionModel{
			Name: "selection-1",
			Posts: []coal.ID{
				coal.MustFromHex(post1),
				coal.MustFromHex(post2),
				coal.MustFromHex(post3),
			},
		}).ID()
		tester.Insert(&selectionModel{
			Name: "selection-2",
		})

		// create notes
		note := tester.Insert(&noteModel{
			Title: "note-1",
			Post:  coal.MustFromHex(post1),
		}).ID()
		tester.Insert(&noteModel{
			Title: "note-2",
			Post:  coal.New(),
		})

		// test invalid filter
		tester.Request("GET", "posts?filter[foo]=bar", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid filter \"foo\""
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// test not supported filter
		tester.Request("GET", "posts?filter[text-body]=foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid filter \"text-body\""
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get posts with single value filter
		tester.Request("GET", "posts?filter[title]=post-1", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [
								{
									"type": "selections",
									"id": "`+selection+`"
								}
							],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": {
								"type": "notes",
								"id": "`+note+`"
							},
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[title]=post-1"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// get posts with multi value filter
		tester.Request("GET", "posts?filter[title]=post-2,post-3", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [
								{
									"type": "selections",
									"id": "`+selection+`"
								}
							],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post3+`",
					"attributes": {
						"title": "post-3",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/comments",
								"related": "/posts/`+post3+`/comments"
							}
						},
						"selections": {
							"data": [
								{
									"type": "selections",
									"id": "`+selection+`"
								}
							],
							"links": {
								"self": "/posts/`+post3+`/relationships/selections",
								"related": "/posts/`+post3+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post3+`/relationships/note",
								"related": "/posts/`+post3+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[title]=post-2,post-3"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// get posts with positive boolean
		tester.Request("GET", "posts?filter[published]=true", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [
								{
									"type": "selections",
									"id": "`+selection+`"
								}
							],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": {
								"type": "notes",
								"id": "`+note+`"
							},
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post3+`",
					"attributes": {
						"title": "post-3",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/comments",
								"related": "/posts/`+post3+`/comments"
							}
						},
						"selections": {
							"data": [
								{
									"type": "selections",
									"id": "`+selection+`"
								}
							],
							"links": {
								"self": "/posts/`+post3+`/relationships/selections",
								"related": "/posts/`+post3+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post3+`/relationships/note",
								"related": "/posts/`+post3+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[published]=true"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// get posts with negative boolean
		tester.Request("GET", "posts?filter[published]=false", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [
								{
									"type": "selections",
									"id": "`+selection+`"
								}
							],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[published]=false"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// test not supported relationship filter
		tester.Request("GET", "comments?filter[post]=foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid filter \"post\""
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get to-many posts with negative boolean
		tester.Request("GET", "selections/"+selection+"/posts?filter[published]=false", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [
								{
									"type": "selections",
									"id": "`+selection+`"
								}
							],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/selections/`+selection+`/posts?filter[published]=false"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// test invalid relationship filter id
		tester.Request("GET", "notes?filter[post]=foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "relationship filter value is not an object id"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// filter notes with to-one relationship filter
		tester.Request("GET", "notes?filter[post]="+post1, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "notes",
					"id": "`+note+`",
					"attributes": {
						"title": "note-1"
					},
					"relationships": {
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post1+`"
							},
							"links": {
								"self": "/notes/`+note+`/relationships/post",
								"related": "/notes/`+note+`/post"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/notes?filter[post]=`+post1+`"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// filter selections with to-many relationship filter
		tester.Request("GET", "selections?filter[posts]="+post1, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "selections",
					"id": "`+selection+`",
					"attributes": {
						"name": "selection-1"
					},
					"relationships": {
						"posts": {
							"data": [
								{
									"type": "posts",
									"id": "`+post1+`"
								},
								{
									"type": "posts",
									"id": "`+post2+`"
								},
								{
									"type": "posts",
									"id": "`+post3+`"
								}
							],
							"links": {
								"self": "/selections/`+selection+`/relationships/posts",
								"related": "/selections/`+selection+`/posts"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/selections?filter[posts]=`+post1+`"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// filter selections with multiple to-many relationship filters
		tester.Request("GET", "selections?filter[posts]="+post1+","+post2, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "selections",
					"id": "`+selection+`",
					"attributes": {
						"name": "selection-1"
					},
					"relationships": {
						"posts": {
							"data": [
								{
									"type": "posts",
									"id": "`+post1+`"
								},
								{
									"type": "posts",
									"id": "`+post2+`"
								},
								{
									"type": "posts",
									"id": "`+post3+`"
								}
							],
							"links": {
								"self": "/selections/`+selection+`/relationships/posts",
								"related": "/selections/`+selection+`/posts"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/selections?filter[posts]=`+post1+`,`+post2+`"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})
	})
}

func TestFilterHandlers(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		assert.PanicsWithValue(t, `fire: filter handler for missing filter "Title"`, func() {
			tester.Assign("", &Controller{
				Model: &postModel{},
				FilterHandlers: map[string]FilterHandler{
					"Title": nil,
				},
			})
		})

		tester.Assign("", &Controller{
			Model:   &postModel{},
			Filters: []string{"Title"},
			FilterHandlers: map[string]FilterHandler{
				"Title": func(ctx *Context, values []string) (bson.M, error) {
					if len(values) == 1 && values[0] == "error" {
						return nil, xo.SF("invalid title filter")
					} else if len(values) == 1 && values[0] == "true" {
						return bson.M{
							"Title": "bar",
						}, nil
					}
					return nil, nil
				},
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model:   &selectionModel{},
			Filters: []string{"Posts"},
		}, &Controller{
			Model:   &noteModel{},
			Filters: []string{"Post"},
		})

		// create posts
		post1 := tester.Insert(&postModel{
			Title: "foo",
		}).ID()
		post2 := tester.Insert(&postModel{
			Title: "bar",
		}).ID()
		tester.Insert(&postModel{
			Title: "baz",
		})

		// test filter handler error
		tester.Request("GET", "posts?filter[title]=error", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid title filter"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// test filter handler
		tester.Request("GET", "posts?filter[title]=true", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			ids := gjson.Get(r.Body.String(), "data.#.id").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				"`+post2+`"
			]`, ids, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[title]=true"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// test nil expression filter handler
		tester.Request("GET", "posts?filter[title]=foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			ids := gjson.Get(r.Body.String(), "data.#.id").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				"`+post1+`"
			]`, ids, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[title]=foo"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})
	})
}

func TestSorting(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model:   &postModel{},
			Sorters: []string{"Title", "TextBody"},
		}, &Controller{
			Model:   &commentModel{},
			Sorters: []string{"Message"},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create posts in random order
		post2 := tester.Insert(&postModel{
			Title:    "post-2",
			TextBody: "body-2",
		}).ID()
		post1 := tester.Insert(&postModel{
			Title:    "post-1",
			TextBody: "body-1",
		}).ID()
		post3 := tester.Insert(&postModel{
			Title:    "post-3",
			TextBody: "body-3",
		}).ID()

		// test invalid sorter
		tester.Request("GET", "posts?sort=foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "invalid sorter \"foo\""
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// test invalid sorter
		tester.Request("GET", "posts?sort=published", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "unsupported sorter \"published\""
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get posts in ascending order
		tester.Request("GET", "posts?sort=title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": false,
						"text-body": "body-1"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": false,
						"text-body": "body-2"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post3+`",
					"attributes": {
						"title": "post-3",
						"published": false,
						"text-body": "body-3"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/comments",
								"related": "/posts/`+post3+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/selections",
								"related": "/posts/`+post3+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post3+`/relationships/note",
								"related": "/posts/`+post3+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?sort=title"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// get posts in descending order
		tester.Request("GET", "posts?sort=-title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post3+`",
					"attributes": {
						"title": "post-3",
						"published": false,
						"text-body": "body-3"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/comments",
								"related": "/posts/`+post3+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/selections",
								"related": "/posts/`+post3+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post3+`/relationships/note",
								"related": "/posts/`+post3+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": false,
						"text-body": "body-2"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": false,
						"text-body": "body-1"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?sort=-title"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// get posts in ascending order
		tester.Request("GET", "posts?sort=text-body", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": false,
						"text-body": "body-1"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": false,
						"text-body": "body-2"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post3+`",
					"attributes": {
						"title": "post-3",
						"published": false,
						"text-body": "body-3"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/comments",
								"related": "/posts/`+post3+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/selections",
								"related": "/posts/`+post3+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post3+`/relationships/note",
								"related": "/posts/`+post3+`/note"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?sort=text-body"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// create post
		post := tester.Insert(&postModel{
			Title: "Post",
		}).ID()

		// create some comments
		comment1 := tester.Insert(&commentModel{
			Message: "Comment 1",
			Post:    post,
		}).ID()
		comment2 := tester.Insert(&commentModel{
			Message: "Comment 2",
			Post:    post,
		}).ID()
		comment3 := tester.Insert(&commentModel{
			Message: "Comment 3",
			Post:    post,
		}).ID()

		// get first page of comments
		tester.Request("GET", "posts/"+post+"/comments?sort=message", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "comments",
					"id": "`+comment1+`",
					"attributes": {
						"message": "Comment 1"
					},
					"relationships": {
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment1+`/relationships/parent",
								"related": "/comments/`+comment1+`/parent"
							}
						},
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/comments/`+comment1+`/relationships/post",
								"related": "/comments/`+comment1+`/post"
							}
						}
					}
				},
				{
					"type": "comments",
					"id": "`+comment2+`",
					"attributes": {
						"message": "Comment 2"
					},
					"relationships": {
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment2+`/relationships/parent",
								"related": "/comments/`+comment2+`/parent"
							}
						},
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/comments/`+comment2+`/relationships/post",
								"related": "/comments/`+comment2+`/post"
							}
						}
					}
				},
				{
					"type": "comments",
					"id": "`+comment3+`",
					"attributes": {
						"message": "Comment 3"
					},
					"relationships": {
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment3+`/relationships/parent",
								"related": "/comments/`+comment3+`/parent"
							}
						},
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/comments/`+comment3+`/relationships/post",
								"related": "/comments/`+comment3+`/post"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts/`+post+`/comments?sort=message"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// get second page of comments
		tester.Request("GET", "posts/"+post+"/comments?sort=-message", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				{
					"type": "comments",
					"id": "`+comment3+`",
					"attributes": {
						"message": "Comment 3"
					},
					"relationships": {
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment3+`/relationships/parent",
								"related": "/comments/`+comment3+`/parent"
							}
						},
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/comments/`+comment3+`/relationships/post",
								"related": "/comments/`+comment3+`/post"
							}
						}
					}
				},
				{
					"type": "comments",
					"id": "`+comment2+`",
					"attributes": {
						"message": "Comment 2"
					},
					"relationships": {
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment2+`/relationships/parent",
								"related": "/comments/`+comment2+`/parent"
							}
						},
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/comments/`+comment2+`/relationships/post",
								"related": "/comments/`+comment2+`/post"
							}
						}
					}
				},
				{
					"type": "comments",
					"id": "`+comment1+`",
					"attributes": {
						"message": "Comment 1"
					},
					"relationships": {
						"parent": {
							"data": null,
							"links": {
								"self": "/comments/`+comment1+`/relationships/parent",
								"related": "/comments/`+comment1+`/parent"
							}
						},
						"post": {
							"data": {
								"type": "posts",
								"id": "`+post+`"
							},
							"links": {
								"self": "/comments/`+comment1+`/relationships/post",
								"related": "/comments/`+comment1+`/post"
							}
						}
					}
				}
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts/`+post+`/comments?sort=-message"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})
	})
}

func TestSearching(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		if tester.Store.Lungo() {
			return
		}

		tester.Assign("", &Controller{
			Model:  &postModel{},
			Search: true,
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		name, err := tester.Store.C(&postModel{}).Native().Indexes().CreateOne(nil, mongo.IndexModel{
			Keys: bson.M{
				"$**": "text",
			},
		})
		assert.NoError(t, err)

		// create posts in random order
		post1 := tester.Insert(&postModel{
			Title:    "post-2",
			TextBody: "bar quz",
		}).ID()
		tester.Insert(&postModel{
			Title:    "post-1",
			TextBody: "bar baz",
		})
		post3 := tester.Insert(&postModel{
			Title:    "post-3",
			TextBody: "foo bar",
		}).ID()

		// attempt to search comments
		tester.Request("GET", "comments?search=foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "search not supported"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt search and sort
		tester.Request("GET", "posts?search=foo&sort=title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors":[{
					"status": "400",
					"title": "bad request",
					"detail": "cannot sort search"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// search posts
		tester.Request("GET", "posts?search=foo+Bar+-baz", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data.#.id").Raw
			score := gjson.Get(r.Body.String(), "data.#.meta.score").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				"`+post3+`",
				"`+post1+`"
			]`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `[
				1.5,
				0.75
			]`, score, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?search=foo+Bar+-baz"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		_, err = tester.Store.C(&postModel{}).Native().Indexes().DropOne(nil, name)
		assert.NoError(t, err)
	})
}

func TestProperties(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		// unknown property
		assert.PanicsWithValue(t, `fire: missing property method "Foo" for model "fire.postModel"`, func() {
			tester.Assign("", &Controller{
				Model: &postModel{},
				Properties: map[string]string{
					"Foo": "foo",
				},
			})
		})

		// invalid shape
		assert.PanicsWithValue(t, `fire: expected property method "SetTitle" for model "fire.postModel" to have no parameters and one or two return values`, func() {
			tester.Assign("", &Controller{
				Model: &postModel{},
				Properties: map[string]string{
					"SetTitle": "set-title",
				},
			})
		})

		// invalid second return value
		assert.PanicsWithValue(t, `fire: expected second return value of property method "Strings" for model "fire.postModel" to be of type error`, func() {
			tester.Assign("", &Controller{
				Model: &postModel{},
				Properties: map[string]string{
					"Strings": "strings",
				},
			})
		})

		group := tester.Assign("", &Controller{
			Model: &postModel{},
			Properties: map[string]string{
				"Virtual":      "virtual",
				"VirtualError": "virtual-error",
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// catch errors
		var errs []string
		group.reporter = func(err error) {
			errs = append(errs, err.Error())
		}

		// create post
		post1 := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": "",
						"virtual": 42,
						"virtual-error": "post-1"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				}],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post1, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": "",
						"virtual": 42,
						"virtual-error": "post-1"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create error post
		errorPost := tester.Insert(&postModel{
			Title: "virtual-error",
		}).ID()

		// error
		tester.Request("GET", "/posts/"+errorPost, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusInternalServerError, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "500",
					"title": "internal server error"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
		assert.Equal(t, "virtual error", errs[0])

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "post-2",
					"published": true,
					"virtual": 42
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post2 := tester.FindLast(&postModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": true,
						"text-body": "",
						"virtual": 42,
						"virtual-error": "post-2"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post2+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post1, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "Post 1",
						"published": true,
						"text-body": "",
						"virtual": 42,
						"virtual-error": "Post 1"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestCallbacks(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		var stages []Stage

		cb := C("Test", 0, All(), func(ctx *Context) error {
			stages = append(stages, ctx.Stage)
			return nil
		})

		action := A("Test", []string{"GET"}, 0, func(ctx *Context) error {
			return nil
		})

		tester.Assign("", &Controller{
			Model:       &fooModel{},
			Authorizers: L{cb},
			Verifiers:   L{cb},
			Modifiers:   L{cb},
			Validators:  L{cb},
			Decorators:  L{cb},
			Notifiers:   L{cb},
			ResourceActions: M{
				"test": action,
			},
			CollectionActions: M{
				"test": action,
			},
		})

		id := tester.Insert(&fooModel{
			String: "Hello World!",
		}).ID()

		id2 := tester.Insert(&fooModel{
			String: "Hello Cool!",
		}).ID()

		// list
		tester.Request("GET", "foos", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// find
		tester.Request("GET", "foos/"+id, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// create
		tester.Request("POST", "foos", `{
			"data": {
				"type": "foos",
				"attributes": {
					"string": "Hello you!"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Modifier,
				Validator,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// update
		tester.Request("PATCH", "foos/"+id, `{
			"data": {
				"type": "foos",
				"id": "`+id+`",
				"attributes": {
					"string": "Awesome!"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Modifier,
				Validator,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// set relationship
		tester.Request("PATCH", "foos/"+id+"/relationships/foo", `{
			"data": {
				"type": "foos",
				"id": "`+id2+`"
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Modifier,
				Validator,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// get related
		tester.Request("GET", "foos/"+id+"/foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Authorizer,
				Verifier,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// get relationship
		tester.Request("GET", "foos/"+id+"/relationships/foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// add relationship
		tester.Request("POST", "foos/"+id+"/relationships/foos", `{
			"data": [
				{
					"type": "foos",
					"id": "`+id2+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Modifier,
				Validator,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// remove relationship
		tester.Request("DELETE", "foos/"+id+"/relationships/foos", `{
			"data": [
				{
					"type": "foos",
					"id": "`+id2+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Modifier,
				Validator,
				Decorator,
				Notifier,
			}, stages)
			stages = nil
		})

		// resource action
		tester.Request("GET", "foos/"+id+"/test", ``, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
			}, stages)
			stages = nil
		})

		// delete
		tester.Request("DELETE", "foos/"+id, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNoContent, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
				Verifier,
				Modifier,
				Validator,
				Notifier,
			}, stages)
			stages = nil
		})

		// collection action
		tester.Request("GET", "foos/test", ``, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, []Stage{
				Authorizer,
			}, stages)
			stages = nil
		})
	})
}

func TestAuthorizers(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Authorizers: L{
				C("TestAuthorizer", Authorizer, All(), func(ctx *Context) error {
					return xo.SF("not authorized")
				}),
			},
		})

		// create post
		post := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "post-2"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post, `{
			"data": {
				"type": "posts",
				"id": "`+post+`",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete
		tester.Request("DELETE", "/posts/"+post, "{}", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestVerifiers(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Verifiers: L{
				C("TestVerifier", Verifier, All(), func(ctx *Context) error {
					return xo.SF("not authorized")
				}),
			},
		})

		// create post
		post := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "post-2"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post, `{
			"data": {
				"type": "posts",
				"id": "`+post+`",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete
		tester.Request("DELETE", "/posts/"+post, "{}", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusUnauthorized, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "401",
					"title": "unauthorized",
					"detail": "not authorized"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestModifiers(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		var calls []Operation

		tester.Assign("", &Controller{
			Model: &postModel{},
			Modifiers: L{
				C("TestModifier", Modifier, All(), func(ctx *Context) error {
					ctx.Model.(*postModel).TextBody += "!!!"
					calls = append(calls, ctx.Operation)
					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create post
		post1 := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
			TextBody:  "Hello",
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": "Hello"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				}],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post1, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": "Hello"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "post-2",
					"published": true,
					"text-body": "Hello"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post2 := tester.FindLast(&postModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": true,
						"text-body": "Hello!!!"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post2+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post1, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "Post 1",
						"published": true,
						"text-body": "Hello!!!"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete
		tester.Request("DELETE", "/posts/"+post1, "{}", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNoContent, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Empty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})

		assert.Equal(t, []Operation{Create, Update, Delete}, calls)
	})
}

func TestValidators(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Validators: L{
				C("TestValidators", Validator, All(), func(ctx *Context) error {
					return xo.SF("not valid")
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create post
		post := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
		})

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "post-2"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "not valid"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post, `{
			"data": {
				"type": "posts",
				"id": "`+post+`",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "not valid"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete
		tester.Request("DELETE", "/posts/"+post, "{}", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "not valid"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestDecorators(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Decorators: L{
				C("TestDecorator", Decorator, All(), func(ctx *Context) error {
					if ctx.Model != nil {
						ctx.Model.(*postModel).TextBody = "Hello World!"
					}

					for _, model := range ctx.Models {
						model.(*postModel).TextBody = "Hello World!"
					}

					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create post
		post1 := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": "Hello World!"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				}],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post1, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": "Hello World!"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "post-2",
					"published": true
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post2 := tester.FindLast(&postModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": true,
						"text-body": "Hello World!"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post2+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post1, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "Post 1",
						"published": true,
						"text-body": "Hello World!"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete
		tester.Request("DELETE", "/posts/"+post1, "{}", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNoContent, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Empty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestNotifiers(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Notifiers: L{
				C("TestNotifier", Notifier, All(), func(ctx *Context) error {
					if ctx.Response != nil {
						ctx.Response.Meta = jsonapi.Map{
							"Hello": "World!",
						}
					} else {
						ctx.Response = &jsonapi.Document{
							Meta: jsonapi.Map{
								"Hello": "World!",
							},
						}
					}

					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create post
		post1 := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// list
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				}],
				"links": {
					"self": "/posts"
				},
				"meta": {
					"Hello": "World!"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// find
		tester.Request("GET", "/posts/"+post1, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				},
				"meta": {
					"Hello": "World!"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create
		tester.Request("POST", "/posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "post-2",
					"published": true
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post2 := tester.FindLast(&postModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post2+`",
					"attributes": {
						"title": "post-2",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/comments",
								"related": "/posts/`+post2+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post2+`/relationships/selections",
								"related": "/posts/`+post2+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post2+`/relationships/note",
								"related": "/posts/`+post2+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post2+`"
				},
				"meta": {
					"Hello": "World!"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update
		tester.Request("PATCH", "/posts/"+post1, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "Post 1",
						"published": true,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/comments",
								"related": "/posts/`+post1+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post1+`/relationships/note",
								"related": "/posts/`+post1+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post1+`"
				},
				"meta": {
					"Hello": "World!"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete
		tester.Request("DELETE", "/posts/"+post1, "{}", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNoContent, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"meta": {
					"Hello": "World!"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestSparseFields(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Properties: map[string]string{
				"Virtual": "virtual",
			},
		}, &Controller{
			Model: &noteModel{},
		})

		// create post
		post := tester.Insert(&postModel{
			Title: "Post 1",
		}).ID()

		// get posts
		tester.Request("GET", "posts/"+post+"?fields[posts]=title,virtual,note", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"type": "posts",
				"id": "`+post+`",
				"attributes": {
					"title": "Post 1",
					"virtual": 42
				},
				"relationships": {
					"note": {
						"data": null,
						"links": {
							"self": "/posts/`+post+`/relationships/note",
							"related": "/posts/`+post+`/note"
						}
					}
				}
			}`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts/`+post+`?fields[posts]=title,virtual,note"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})

		// create note
		note := tester.Insert(&noteModel{
			Title: "Note 1",
			Post:  post,
		}).ID()

		// get related note
		tester.Request("GET", "/posts/"+post+"/note?fields[notes]=post", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			data := gjson.Get(r.Body.String(), "data").Raw
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"type": "notes",
				"id": "`+note+`",
				"relationships": {
					"post": {
						"data": {
							"type": "posts",
							"id": "`+post+`"
						},
						"links": {
							"self": "/notes/`+note+`/relationships/post",
							"related": "/notes/`+note+`/post"
						}
					}
				}
			}`, data, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts/`+post+`/note?fields[notes]=post"
			}`, linkUnescape(links), tester.DebugRequest(rq, r))
		})
	})
}

func TestReadableFields(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model:   &postModel{},
			Filters: []string{"Title"},
			Sorters: []string{"Title"},
			Authorizers: L{
				C("TestReadableFields", Authorizer, All(), func(ctx *Context) error {
					assert.Equal(t, []string{"Comments", "Note", "Published", "Selections", "TextBody", "Title"}, ctx.ReadableFields)
					assert.Equal(t, []string{"Published", "TextBody", "Title"}, ctx.WritableFields)
					ctx.ReadableFields = []string{"Published"}
					return nil
				}),
			},
		}, &Controller{
			Model:   &noteModel{},
			Filters: []string{"Post"},
			Authorizers: L{
				C("TestReadableFields", Authorizer, All(), func(ctx *Context) error {
					assert.Equal(t, []string{"Post", "Title"}, ctx.ReadableFields)
					assert.Equal(t, []string{"Post", "Title"}, ctx.WritableFields)
					ctx.ReadableFields = []string{}
					return nil
				}),
			},
		})

		// create post
		post := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// get posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post+`",
						"attributes": {
							"published": true
						}
					}
				],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to get relationship
		tester.Request("GET", "/posts/"+post+"/relationships/note", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "relationship is not readable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to get related note
		tester.Request("GET", "/posts/"+post+"/note", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "relationship is not readable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// filter posts
		tester.Request("GET", "posts?filter[title]=foo", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "filter field is not readable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// filter notes
		tester.Request("GET", "notes?filter[post]="+post, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "filter field is not readable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// sort posts
		tester.Request("GET", "posts?sort=title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "sort field is not readable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestReadableFieldsGetter(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Authorizers: L{
				C("TestReadableFieldsGetter", Authorizer, All(), func(ctx *Context) error {
					ctx.GetReadableFields = func(model coal.Model) []string {
						if model != nil {
							post := model.(*postModel)
							if post.Title == "post1" {
								return []string{"Title", "Published"}
							}
						}
						return []string{"Title"}
					}
					return nil
				}),
			},
		})

		// create posts
		post1 := tester.Insert(&postModel{
			Title:     "post1",
			Published: true,
		}).ID()
		post2 := tester.Insert(&postModel{
			Title:     "post2",
			Published: true,
		}).ID()

		// get posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post1+`",
						"attributes": {
							"title": "post1",
							"published": true
						}
					},
					{
						"type": "posts",
						"id": "`+post2+`",
						"attributes": {
							"title": "post2"
						}
					}
				],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestWritableFields(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Authorizers: L{
				C("TestWritableFields", Authorizer, All(), func(ctx *Context) error {
					assert.Equal(t, []string{"Comments", "Note", "Published", "Selections", "TextBody", "Title"}, ctx.ReadableFields)
					assert.Equal(t, []string{"Published", "TextBody", "Title"}, ctx.WritableFields)
					ctx.WritableFields = []string{"Title"}
					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
			Authorizers: L{
				C("TestWritableFields", Authorizer, All(), func(ctx *Context) error {
					assert.Equal(t, []string{"CreateToken", "Name", "Posts", "UpdateToken"}, ctx.ReadableFields)
					assert.Equal(t, []string{"CreateToken", "Name", "Posts", "UpdateToken"}, ctx.WritableFields)
					ctx.WritableFields = []string{}
					return nil
				}),
			},
		}, &Controller{
			Model: &noteModel{},
		})

		// attempt to create post with protected field
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "Post 1",
					"published": true
				},
				"relationships": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "field is not writable",
					"source": {
						"pointer": "Published"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create post with protected field zero value
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "Post 1",
					"published": false
				},
				"relationships": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.NotEmpty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})

		post1 := tester.FindLast(&postModel{}).ID()

		// attempt to update post with protected field
		tester.Request("PATCH", "posts/"+post1, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`",
				"attributes": {
					"published": true
				},
				"relationships": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "field is not writable",
					"source": {
						"pointer": "Published"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update post with protected field zero value
		tester.Request("PATCH", "posts/"+post1, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`",
				"attributes": {
					"published": false
				},
				"relationships": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.NotEmpty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create selection with protected relationship
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {},
				"relationships": {
					"posts": {
						"data": [{
							"type": "posts",
							"id": "`+coal.New()+`"
						}]
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "field is not writable",
					"source": {
						"pointer": "Posts"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create selection with protected relationship zero value
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {},
				"relationships": {
					"posts": {
						"data": null
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.NotEmpty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})

		selection1 := tester.FindLast(&selectionModel{}).ID()

		// attempt to update selection with protected relationship
		tester.Request("PATCH", "selections/"+selection1, `{
			"data": {
				"type": "selections",
				"id": "`+selection1+`",
				"relationships": {
					"posts": {
						"data": [{
							"type": "posts",
							"id": "`+coal.New()+`"
						}]
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "field is not writable",
					"source": {
						"pointer": "Posts"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update selection with protected relationship zero value
		tester.Request("PATCH", "selections/"+selection1, `{
			"data": {
				"type": "selections",
				"id": "`+selection1+`",
				"relationships": {
					"posts": {
						"data": null
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.NotEmpty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})

		post2 := coal.New()

		// attempt to update posts relationship
		tester.Request("PATCH", "selections/"+selection1+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "`+coal.New()+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "relationship is not writable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to add posts relationship
		tester.Request("POST", "selections/"+selection1+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "`+coal.New()+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "relationship is not writable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to remove from posts relationship
		tester.Request("DELETE", "selections/"+selection1+"/relationships/posts", `{
			"data": [
				{
					"type": "posts",
					"id": "`+post2+`"
				}
			]
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "relationship is not writable"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestWritableFieldsGetter(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Authorizers: L{
				C("TestWritableFieldsGetter", Authorizer, All(), func(ctx *Context) error {
					ctx.ReadableFields = []string{"Title"}
					ctx.GetWritableFields = func(model coal.Model) []string {
						if model != nil {
							post := model.(*postModel)
							if post.Title == "post2" {
								return []string{"Title", "Published"}
							}
						}
						return []string{"Title"}
					}
					return nil
				}),
			},
		})

		// create posts
		post1 := tester.Insert(&postModel{
			Title: "post1",
		}).ID()
		post2 := tester.Insert(&postModel{
			Title: "post2",
		}).ID()

		// attempt to update post
		tester.Request("PATCH", "posts/"+post1, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`",
				"attributes": {
					"published": true
				},
				"relationships": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "field is not writable",
					"source": {
						"pointer": "Published"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create post
		tester.Request("PATCH", "posts/"+post2, `{
			"data": {
				"type": "posts",
				"id": "`+post2+`",
				"attributes": {
					"published": true
				},
				"relationships": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.NotEmpty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestReadableProperties(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Properties: map[string]string{
				"Virtual":      "virtual",
				"VirtualError": "virtual-error",
			},
			Authorizers: L{
				C("TestReadableProperties", Authorizer, All(), func(ctx *Context) error {
					assert.Equal(t, []string{"Virtual", "VirtualError"}, ctx.ReadableProperties)
					ctx.ReadableProperties = []string{"Virtual"}
					return nil
				}),
			},
		}, &Controller{
			Model: &noteModel{},
			Authorizers: L{
				C("TestReadableProperties", Authorizer, All(), func(ctx *Context) error {
					assert.Equal(t, []string{}, ctx.ReadableProperties)
					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		})

		// create post
		post := tester.Insert(&postModel{
			Title:     "post-1",
			Published: true,
		}).ID()

		// get posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post+`",
						"attributes": {
							"title": "post-1",
							"published": true,
							"text-body": "",
							"virtual": 42
						},
						"relationships": {
							"comments": {
								"data": [],
								"links": {
									"self": "/posts/`+post+`/relationships/comments",
									"related": "/posts/`+post+`/comments"
								}
							},
							"selections": {
								"data": [],
								"links": {
									"self": "/posts/`+post+`/relationships/selections",
									"related": "/posts/`+post+`/selections"
								}
							},
							"note": {
								"data": null,
								"links": {
									"self": "/posts/`+post+`/relationships/note",
									"related": "/posts/`+post+`/note"
								}
							}
						}
					}
				],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestReadablePropertiesGetter(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Properties: map[string]string{
				"Virtual":      "virtual",
				"VirtualError": "virtual-error",
			},
			Authorizers: L{
				C("TestReadablePropertiesGetter", Authorizer, All(), func(ctx *Context) error {
					ctx.ReadableFields = []string{"Title"}
					ctx.GetReadableProperties = func(model coal.Model) []string {
						if model != nil {
							if model.(*postModel).Title == "post1" {
								return []string{"Virtual", "VirtualError"}
							}
						}
						return []string{"Virtual"}
					}
					return nil
				}),
			},
		})

		// create posts
		post1 := tester.Insert(&postModel{
			Title: "post1",
		}).ID()
		post2 := tester.Insert(&postModel{
			Title: "post2",
		}).ID()

		// get posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post1+`",
						"attributes": {
							"title": "post1",
							"virtual": 42,
							"virtual-error": "post1"
						}
					},
					{
						"type": "posts",
						"id": "`+post2+`",
						"attributes": {
							"title": "post2",
							"virtual": 42
						}
					}
				],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestRelationshipFilters(t *testing.T) {
	// TODO: Support to one relationships?
	// TODO: Support to many relationships?

	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Authorizers: L{
				C("TestRelationshipFilters", Authorizer, All(), func(ctx *Context) error {
					ctx.RelationshipFilters = map[string][]bson.M{
						"Comments": {
							{
								"Message": "bar",
							},
						},
						"Selections": {
							{
								"Name": "bar",
							},
						},
						"Note": {
							{
								"Title": "bar",
							},
						},
					}
					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
			Authorizers: L{
				C("TestRelationshipFilters", Authorizer, All(), func(ctx *Context) error {
					ctx.RelationshipFilters = map[string][]bson.M{
						"Posts": {
							{
								"Title": "bar",
							},
						},
					}
					return nil
				}),
			},
		}, &Controller{
			Model: &noteModel{},
			Authorizers: L{
				C("TestRelationshipFilters", Authorizer, All(), func(ctx *Context) error {
					ctx.RelationshipFilters = map[string][]bson.M{
						"Post": {
							{
								"Title": "x",
							},
						},
					}
					return nil
				}),
			},
		})

		// create post
		post := tester.Insert(&postModel{
			Title: "post",
		}).ID()

		// create comment
		comment1 := coal.New()
		tester.Insert(&commentModel{
			Base:    coal.B(coal.MustFromHex(comment1)),
			Message: "foo",
			Parent:  stick.P(coal.MustFromHex(comment1)),
			Post:    coal.MustFromHex(post),
		})
		comment2 := coal.New()
		tester.Insert(&commentModel{
			Base:    coal.B(coal.MustFromHex(comment2)),
			Message: "bar",
			Parent:  stick.P(coal.MustFromHex(comment2)),
			Post:    coal.MustFromHex(post),
		})

		// create selection
		tester.Insert(&selectionModel{
			Name: "foo",
			Posts: []coal.ID{
				coal.MustFromHex(post),
			},
		})
		selection2 := tester.Insert(&selectionModel{
			Name: "bar",
			Posts: []coal.ID{
				coal.MustFromHex(post),
			},
		}).ID()

		// create notes
		tester.Insert(&noteModel{
			Title: "foo",
			Post:  coal.MustFromHex(post),
		})
		note2 := tester.Insert(&noteModel{
			Title: "bar",
			Post:  coal.MustFromHex(post),
		}).ID()

		// get posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+post+`",
						"attributes": {
							"title": "post",
							"published": false,
							"text-body": ""
						},
						"relationships": {
							"comments": {
								"data": [
									{
										"id": "`+comment2+`",
										"type": "comments"
									}
								],
								"links": {
									"self": "/posts/`+post+`/relationships/comments",
									"related": "/posts/`+post+`/comments"
								}
							},
							"selections": {
								"data": [
									{
										"id": "`+selection2+`",
										"type": "selections"
									}
								],
								"links": {
									"self": "/posts/`+post+`/relationships/selections",
									"related": "/posts/`+post+`/selections"
								}
							},
							"note": {
								"data": {
									"id": "`+note2+`",
									"type": "notes"
								},
								"links": {
									"self": "/posts/`+post+`/relationships/note",
									"related": "/posts/`+post+`/note"
								}
							}
						}
					}
				],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// // get comments
		// tester.Request("GET", "comments", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
		// 	assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
		// 	assert.JSONEq(t, `{
		// 		"data": [
		// 			{
		// 				"type": "comments",
		// 				"id": "`+comment+`",
		// 				"attributes": {
		// 					"message": "comment"
		// 				},
		// 				"relationships": {
		// 					"parent": {
		// 						"data": [],
		// 						"links": {
		// 							"self": "/comments/`+comment+`/relationships/parent",
		// 							"related": "/comments/`+comment+`/parent"
		// 						}
		// 					},
		// 					"post": {
		// 						"data": null,
		// 						"links": {
		// 							"self": "/comments/`+comment+`/relationships/post",
		// 							"related": "/comments/`+comment+`/post"
		// 						}
		// 					}
		// 				}
		// 			}
		// 		],
		// 		"links": {
		// 			"self": "/comments"
		// 		}
		// 	}`, r.Body.String(), tester.DebugRequest(rq, r))
		// })

		// // get selections
		// tester.Request("GET", "selections", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
		// 	assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
		// 	assert.JSONEq(t, `{
		// 		"data": [
		// 			{
		// 				"type": "selections",
		// 				"id": "`+selection+`",
		// 				"attributes": {
		// 					"message": "comment"
		// 				},
		// 				"relationships": {
		// 					"posts": {
		// 						"data": [],
		// 						"links": {
		// 							"self": "/selections/`+selection+`/relationships/posts",
		// 							"related": "/selections/`+selection+`/posts"
		// 						}
		// 					}
		// 				}
		// 			}
		// 		],
		// 		"links": {
		// 			"self": "/selections"
		// 		}
		// 	}`, r.Body.String(), tester.DebugRequest(rq, r))
		// })

		// // get notes
		// tester.Request("GET", "notes", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
		// 	assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
		// 	assert.JSONEq(t, `{
		// 		"data": [
		// 			{
		// 				"type": "notes",
		// 				"id": "`+note+`",
		// 				"attributes": {
		// 					"message": "note"
		// 				},
		// 				"relationships": {
		// 					"post": {
		// 						"data": null,
		// 						"links": {
		// 							"self": "/notes/`+note+`/relationships/post",
		// 							"related": "/notes/`+note+`/post"
		// 						}
		// 					}
		// 				}
		// 			}
		// 		],
		// 		"links": {
		// 			"self": "/notes"
		// 		}
		// 	}`, r.Body.String(), tester.DebugRequest(rq, r))
		// })
	})
}

func TestDeferredCallbacks(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Authorizers: L{
				C("Test", Authorizer, All(), func(ctx *Context) error {
					ctx.Defer(C("Test", Validator, Only(Create), func(ctx *Context) error {
						return io.EOF
					}))
					return nil
				}),
			},
		})

		// missing resource
		tester.Request("POST", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "400",
						"title": "bad request",
						"detail": "EOF"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestDatabaseErrors(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// missing resource
		tester.Request("GET", "posts/"+coal.New(), "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNotFound, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "404",
						"title": "not found",
						"detail": "resource not found"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// add unique index
		index, err := tester.Store.C(&postModel{}).Native().Indexes().CreateOne(tester.Context, mongo.IndexModel{
			Keys: bson.D{
				{Key: "title", Value: int32(1)},
			},
			Options: options.Index().SetUnique(true),
		})
		assert.NoError(t, err)

		// first post
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
		})

		// second post
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "400",
						"title": "bad request",
						"detail": "document is not unique"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// remove index
		_, err = tester.Store.C(&postModel{}).Native().Indexes().DropOne(tester.Context, index)
		assert.NoError(t, err)
	})
}

func TestTolerateViolations(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model: &postModel{},
			Authorizers: L{
				C("TestWritableFields", Authorizer, All(), func(ctx *Context) error {
					ctx.WritableFields = []string{"Title"}
					return nil
				}),
			},
			TolerateViolations: []string{"Published"},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
			Authorizers: L{
				C("TestWritableFields", Authorizer, All(), func(ctx *Context) error {
					ctx.WritableFields = []string{}
					return nil
				}),
			},
			TolerateViolations: []string{"Posts"},
		}, &Controller{
			Model: &noteModel{},
		})

		// attempt to create post with protected field
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "Post 1",
					"published": true
				},
				"relationships": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+post+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+post+`/relationships/comments",
								"related": "/posts/`+post+`/comments"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+post+`/relationships/note",
								"related": "/posts/`+post+`/note"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+post+`/relationships/selections",
								"related": "/posts/`+post+`/selections"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+post+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create selection with protected relationship
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {},
				"relationships": {
					"posts": {
						"data": [{
							"type": "posts",
							"id": "`+coal.New()+`"
						}]
					}
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			selection := tester.FindLast(&selectionModel{}).ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "selections",
					"id": "`+selection+`",
					"attributes": {
						"name": ""
					},
					"relationships": {
						"posts": {
							"data": [],
							"links": {
								"self": "/selections/`+selection+`/relationships/posts",
								"related": "/selections/`+selection+`/posts"
							}
						}
					}
				},
				"links": {
					"self": "/selections/`+selection+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestOffsetPagination(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model:     &postModel{},
			ListLimit: 7,
		}, &Controller{
			Model:     &commentModel{},
			ListLimit: 7,
		}, &Controller{
			Model:     &selectionModel{},
			ListLimit: 7,
		}, &Controller{
			Model:     &noteModel{},
			ListLimit: 7,
		})

		// prepare ids
		var ids []coal.ID

		// create some posts
		for i := 0; i < 10; i++ {
			ids = append(ids, tester.Insert(&postModel{
				Title: fmt.Sprintf("Post %d", i+1),
			}).ID())
		}

		// get first page of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 7, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 1", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[number]=1&page[size]=7",
				"first": "/posts?page[number]=1&page[size]=7",
				"last": "/posts?page[number]=2&page[size]=7",
				"next": "/posts?page[number]=2&page[size]=7"
			}`, linkUnescape(links))
		})

		// get first page of posts
		tester.Request("GET", "posts?page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 1", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[number]=1&page[size]=5",
				"first": "/posts?page[number]=1&page[size]=5",
				"last": "/posts?page[number]=2&page[size]=5",
				"next": "/posts?page[number]=2&page[size]=5"
			}`, linkUnescape(links))
		})

		// get first page of posts
		tester.Request("GET", "posts?page[number]=1&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 1", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[number]=1&page[size]=5",
				"first": "/posts?page[number]=1&page[size]=5",
				"last": "/posts?page[number]=2&page[size]=5",
				"next": "/posts?page[number]=2&page[size]=5"
			}`, linkUnescape(links))
		})

		// get second page of posts
		tester.Request("GET", "posts?page[number]=2&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 6", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[number]=2&page[size]=5",
				"first": "/posts?page[number]=1&page[size]=5",
				"last": "/posts?page[number]=2&page[size]=5",
				"prev": "/posts?page[number]=1&page[size]=5"
			}`, linkUnescape(links))
		})

		// create selection
		selection := tester.Insert(&selectionModel{
			Posts: ids,
		}).ID()

		// get first page of posts
		tester.Request("GET", "selections/"+selection+"/posts?page[number]=1&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 1", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/selections/`+selection+`/posts?page[number]=1&page[size]=5",
				"first": "/selections/`+selection+`/posts?page[number]=1&page[size]=5",
				"last": "/selections/`+selection+`/posts?page[number]=2&page[size]=5",
				"next": "/selections/`+selection+`/posts?page[number]=2&page[size]=5"
			}`, linkUnescape(links))
		})

		// get second page of posts
		tester.Request("GET", "selections/"+selection+"/posts?page[number]=2&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 6", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/selections/`+selection+`/posts?page[number]=2&page[size]=5",
				"first": "/selections/`+selection+`/posts?page[number]=1&page[size]=5",
				"last": "/selections/`+selection+`/posts?page[number]=2&page[size]=5",
				"prev": "/selections/`+selection+`/posts?page[number]=1&page[size]=5"
			}`, linkUnescape(links))
		})

		// create post
		post := tester.Insert(&postModel{
			Title: "Post",
		}).ID()

		// create some comments
		for i := 0; i < 10; i++ {
			tester.Insert(&commentModel{
				Message: fmt.Sprintf("Comment %d", i+1),
				Post:    post,
			})
		}

		// get first page of comments
		tester.Request("GET", "posts/"+post+"/comments?page[number]=1&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Comment 1", list[0].Get("attributes.message").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts/`+post+`/comments?page[number]=1&page[size]=5",
				"first": "/posts/`+post+`/comments?page[number]=1&page[size]=5",
				"last": "/posts/`+post+`/comments?page[number]=2&page[size]=5",
				"next": "/posts/`+post+`/comments?page[number]=2&page[size]=5"
			}`, linkUnescape(links))
		})

		// get second page of comments
		tester.Request("GET", "posts/"+post+"/comments?page[number]=2&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Comment 6", list[0].Get("attributes.message").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts/`+post+`/comments?page[number]=2&page[size]=5",
				"first": "/posts/`+post+`/comments?page[number]=1&page[size]=5",
				"last": "/posts/`+post+`/comments?page[number]=2&page[size]=5",
				"prev": "/posts/`+post+`/comments?page[number]=1&page[size]=5"
			}`, linkUnescape(links))
		})
	})
}

func TestCursorPagination(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model:            &postModel{},
			Filters:          []string{"Published"},
			Sorters:          []string{"Title", "TextBody", "Published"},
			ListLimit:        7,
			CursorPagination: true,
		}, &Controller{
			Model:            &commentModel{},
			ListLimit:        7,
			CursorPagination: true,
		}, &Controller{
			Model:            &selectionModel{},
			ListLimit:        7,
			CursorPagination: true,
		}, &Controller{
			Model:            &noteModel{},
			ListLimit:        7,
			CursorPagination: true,
		})

		// create some posts
		for i := 0; i < 10; i++ {
			tester.Insert(&postModel{
				Base:      coal.B(numID(uint8(i) + 1)),
				Title:     fmt.Sprintf("Post %02d", i+1),
				TextBody:  fmt.Sprintf("Body %02d", 10-i),
				Published: i >= 5,
			})
		}

		/* ascending */

		// get first page of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 7, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 01", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 07", list[6].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[after]=*&page[size]=7",
				"first": "/posts?page[after]=*&page[size]=7",
				"prev": null,
				"next": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABwA&page[size]=7",
				"last": "/posts?page[before]=*&page[size]=7"
			}`, linkUnescape(links))
		})

		// get first page of posts with size
		tester.Request("GET", "posts?page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 01", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 05", list[4].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[after]=*&page[size]=5",
				"first": "/posts?page[after]=*&page[size]=5",
				"prev": null,
				"next": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=5",
				"last": "/posts?page[before]=*&page[size]=5"
			}`, linkUnescape(links))
		})

		// get last page of posts
		tester.Request("GET", "posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 06", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 10", list[4].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=5",
				"first": "/posts?page[after]=*&page[size]=5",
				"prev": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=5",
				"next": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAACgA&page[size]=5",
				"last": "/posts?page[before]=*&page[size]=5"
			}`, linkUnescape(links))
		})

		// get empty last page of posts
		tester.Request("GET", "posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAACgA&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 0, len(list), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAACgA&page[size]=5",
				"first": "/posts?page[after]=*&page[size]=5",
				"prev": "/posts?page[before]=*&page[size]=5",
				"next": null,
				"last": "/posts?page[before]=*&page[size]=5"
			}`, linkUnescape(links))
		})

		// get capped page of posts
		tester.Request("GET", "posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=6", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 06", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 10", list[4].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=6",
				"first": "/posts?page[after]=*&page[size]=6",
				"prev": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=6",
				"next": null,
				"last": "/posts?page[before]=*&page[size]=6"
			}`, linkUnescape(links))
		})

		/* descending */

		// get last page of posts (reverse)
		tester.Request("GET", "posts?page[before]=*&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 06", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 10", list[4].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[before]=*&page[size]=5",
				"first": "/posts?page[after]=*&page[size]=5",
				"prev": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=5",
				"next": null,
				"last": "/posts?page[before]=*&page[size]=5"
			}`, linkUnescape(links))
		})

		// get middle page of posts (reverse)
		tester.Request("GET", "posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=3", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 3, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 03", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 05", list[2].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=3",
				"first": "/posts?page[after]=*&page[size]=3",
				"prev": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAAAwA&page[size]=3",
				"next": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=3",
				"last": "/posts?page[before]=*&page[size]=3"
			}`, linkUnescape(links))
		})

		// get first page of posts (reverse)
		tester.Request("GET", "posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 01", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 05", list[4].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=5",
				"first": "/posts?page[after]=*&page[size]=5",
				"prev": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAAAQA&page[size]=5",
				"next": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=5",
				"last": "/posts?page[before]=*&page[size]=5"
			}`, linkUnescape(links))
		})

		// get empty first page of posts (reverse)
		tester.Request("GET", "posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAAAQA&page[size]=5", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 0, len(list), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAAAQA&page[size]=5",
				"first": "/posts?page[after]=*&page[size]=5",
				"prev": null,
				"next": "/posts?page[after]=*&page[size]=5",
				"last": "/posts?page[before]=*&page[size]=5"
			}`, linkUnescape(links))
		})

		// get capped first page of posts (reverse)
		tester.Request("GET", "posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=6", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 01", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 05", list[4].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=6",
				"first": "/posts?page[after]=*&page[size]=6",
				"prev": null,
				"next": "/posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=6",
				"last": "/posts?page[before]=*&page[size]=6"
			}`, linkUnescape(links))
		})

		/* sorted */

		// get first page of sorted posts
		tester.Request("GET", "posts?sort=-title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 7, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 10", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 04", list[6].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[after]=*&page[size]=7&sort=-title",
				"first": "/posts?page[after]=*&page[size]=7&sort=-title",
				"prev": null,
				"next": "/posts?page[after]=IwAAAAIwAAgAAABQb3N0IDA0AAcxAAAAAAAAAAAAAAAABAA&page[size]=7&sort=-title",
				"last": "/posts?page[before]=*&page[size]=7&sort=-title"
			}`, linkUnescape(links))
		})

		// get last page of sorted posts
		tester.Request("GET", "posts?page[after]=IwAAAAIwAAgAAABQb3N0IDA0AAcxAAAAAAAAAAAAAAAABAA&page[size]=7&sort=-title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 3, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 03", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 01", list[2].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[after]=IwAAAAIwAAgAAABQb3N0IDA0AAcxAAAAAAAAAAAAAAAABAA&page[size]=7&sort=-title",
				"first": "/posts?page[after]=*&page[size]=7&sort=-title",
				"prev": "/posts?page[before]=IwAAAAIwAAgAAABQb3N0IDAzAAcxAAAAAAAAAAAAAAAAAwA&page[size]=7&sort=-title",
				"next": null,
				"last": "/posts?page[before]=*&page[size]=7&sort=-title"
			}`, linkUnescape(links))
		})

		// get first page of sorted posts (reverse)
		tester.Request("GET", "posts?page[before]=IwAAAAIwAAgAAABQb3N0IDAzAAcxAAAAAAAAAAAAAAAAAwA&page[size]=7&sort=-title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 7, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 10", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 04", list[6].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[before]=IwAAAAIwAAgAAABQb3N0IDAzAAcxAAAAAAAAAAAAAAAAAwA&page[size]=7&sort=-title",
				"first": "/posts?page[after]=*&page[size]=7&sort=-title",
				"prev": "/posts?page[before]=IwAAAAIwAAgAAABQb3N0IDEwAAcxAAAAAAAAAAAAAAAACgA&page[size]=7&sort=-title",
				"next": "/posts?page[after]=IwAAAAIwAAgAAABQb3N0IDA0AAcxAAAAAAAAAAAAAAAABAA&page[size]=7&sort=-title",
				"last": "/posts?page[before]=*&page[size]=7&sort=-title"
			}`, linkUnescape(links))
		})

		/* filtered */

		// get first page of filtered posts
		tester.Request("GET", "posts?filter[published]=true&page[size]=3", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 3, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 06", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 08", list[2].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[published]=true&page[after]=*&page[size]=3",
				"first": "/posts?filter[published]=true&page[after]=*&page[size]=3",
				"prev": null,
				"next": "/posts?filter[published]=true&page[after]=FAAAAAcwAAAAAAAAAAAAAAAACAA&page[size]=3",
				"last": "/posts?filter[published]=true&page[before]=*&page[size]=3"
			}`, linkUnescape(links))
		})

		// get last page of filtered posts
		tester.Request("GET", "posts?filter[published]=true&page[after]=FAAAAAcwAAAAAAAAAAAAAAAACAA&page[size]=3", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 2, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 09", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 10", list[1].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[published]=true&page[after]=FAAAAAcwAAAAAAAAAAAAAAAACAA&page[size]=3",
				"first": "/posts?filter[published]=true&page[after]=*&page[size]=3",
				"prev": "/posts?filter[published]=true&page[before]=FAAAAAcwAAAAAAAAAAAAAAAACQA&page[size]=3",
				"next": null,
				"last": "/posts?filter[published]=true&page[before]=*&page[size]=3"
			}`, linkUnescape(links))
		})

		// get first page of filtered posts (reverse)
		tester.Request("GET", "posts?filter[published]=true&page[before]=FAAAAAcwAAAAAAAAAAAAAAAACQA&page[size]=3", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 3, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 06", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 08", list[2].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?filter[published]=true&page[before]=FAAAAAcwAAAAAAAAAAAAAAAACQA&page[size]=3",
				"first": "/posts?filter[published]=true&page[after]=*&page[size]=3",
				"prev": "/posts?filter[published]=true&page[before]=FAAAAAcwAAAAAAAAAAAAAAAABgA&page[size]=3",
				"next": "/posts?filter[published]=true&page[after]=FAAAAAcwAAAAAAAAAAAAAAAACAA&page[size]=3",
				"last": "/posts?filter[published]=true&page[before]=*&page[size]=3"
			}`, linkUnescape(links))
		})

		// TODO: Test relationship pagination.

		// try range pagination
		tester.Request("GET", "posts?page[after]=X&page[before]=X", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "range pagination not supported"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// try invalid cursor sorting combination
		tester.Request("GET", "posts?page[after]=FAAAAAcwAAAAAAAAAAAAAAAABQA&page[size]=5&sort=-title", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "cursor sorting mismatch"
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestListLimit(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		tester.Assign("", &Controller{
			Model:     &postModel{},
			ListLimit: 5,
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// create some posts
		for i := 0; i < 10; i++ {
			tester.Insert(&postModel{
				Title: fmt.Sprintf("Post %d", i+1),
			})
		}

		// get first page of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			list := gjson.Get(r.Body.String(), "data").Array()
			links := gjson.Get(r.Body.String(), "links").Raw

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, 5, len(list), tester.DebugRequest(rq, r))
			assert.Equal(t, "Post 1", list[0].Get("attributes.title").String(), tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"self": "/posts?page[number]=1&page[size]=5",
				"first": "/posts?page[number]=1&page[size]=5",
				"last": "/posts?page[number]=2&page[size]=5",
				"next": "/posts?page[number]=2&page[size]=5"
			}`, linkUnescape(links))
		})

		// try bigger page size than limit
		tester.Request("GET", "posts?page[size]=7", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [{
					"status": "400",
					"title": "bad request",
					"detail": "max page size exceeded",
					"source": {
						"parameter": "page[size]"
					}
				}]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestCollectionActions(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		assert.PanicsWithValue(t, `fire: invalid collection action ""`, func() {
			tester.Assign("api", &Controller{
				Model: &postModel{},
				CollectionActions: M{
					"": A("foo", []string{"POST"}, 0, func(ctx *Context) error {
						return nil
					}),
				},
			})
		})

		id := coal.New()
		assert.PanicsWithValue(t, `fire: invalid collection action "`+id+`"`, func() {
			tester.Assign("api", &Controller{
				Model: &postModel{},
				CollectionActions: M{
					id: A("foo", []string{"POST"}, 0, func(ctx *Context) error {
						return nil
					}),
				},
			})
		})

		tester.Assign("api", &Controller{
			Model: &postModel{},
			CollectionActions: M{
				"bytes": A("bytes", []string{"POST"}, 0, func(ctx *Context) error {
					bytes, err := ioutil.ReadAll(ctx.HTTPRequest.Body)
					assert.NoError(t, err)
					assert.Equal(t, []byte("PAYLOAD"), bytes)

					_, err = ctx.ResponseWriter.Write([]byte("RESPONSE"))
					assert.NoError(t, err)

					return nil
				}),
				"empty": A("empty", []string{"POST"}, 0, func(ctx *Context) error {
					bytes, err := ioutil.ReadAll(ctx.HTTPRequest.Body)
					assert.NoError(t, err)
					assert.Empty(t, bytes)

					return nil
				}),
				"error": A("error", []string{"POST"}, 3, func(ctx *Context) error {
					bytes, err := ioutil.ReadAll(ctx.HTTPRequest.Body)
					assert.Error(t, err)
					assert.Equal(t, []byte{}, bytes)

					ctx.ResponseWriter.WriteHeader(http.StatusRequestEntityTooLarge)

					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		// get byte response
		tester.Header["Content-Type"] = "text/plain"
		tester.Header["Accept"] = "text/plain"
		tester.Request("POST", "posts/bytes", "PAYLOAD", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, "text/plain; charset=utf-8", r.Result().Header.Get("Content-Type"), tester.DebugRequest(rq, r))
			assert.Equal(t, "RESPONSE", r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty response
		tester.Header["Content-Type"] = ""
		tester.Header["Accept"] = ""
		tester.Request("POST", "posts/empty", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Empty(t, r.Result().Header.Get("Content-Type"), tester.DebugRequest(rq, r))
			assert.Empty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// error
		tester.Request("POST", "posts/error", "PAYLOAD", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusRequestEntityTooLarge, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Empty(t, r.Result().Header.Get("Content-Type"), tester.DebugRequest(rq, r))
			assert.Empty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestResourceActions(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		assert.PanicsWithValue(t, `fire: invalid resource action ""`, func() {
			tester.Assign("api", &Controller{
				Model: &postModel{},
				ResourceActions: M{
					"": A("foo", []string{"POST"}, 0, func(ctx *Context) error {
						return nil
					}),
				},
			})
		})

		assert.PanicsWithValue(t, `fire: invalid resource action "relationships"`, func() {
			tester.Assign("api", &Controller{
				Model: &postModel{},
				ResourceActions: M{
					"relationships": A("foo", []string{"POST"}, 0, func(ctx *Context) error {
						return nil
					}),
				},
			})
		})

		assert.PanicsWithValue(t, `fire: invalid resource action "note"`, func() {
			tester.Assign("api", &Controller{
				Model: &postModel{},
				ResourceActions: M{
					"note": A("foo", []string{"POST"}, 0, func(ctx *Context) error {
						return nil
					}),
				},
			})
		})

		tester.Assign("api", &Controller{
			Model: &postModel{},
			ResourceActions: M{
				"bytes": A("bytes", []string{"POST"}, 0, func(ctx *Context) error {
					assert.NotEmpty(t, ctx.Model)

					bytes, err := ioutil.ReadAll(ctx.HTTPRequest.Body)
					assert.NoError(t, err)
					assert.Equal(t, []byte("PAYLOAD"), bytes)

					_, err = ctx.ResponseWriter.Write([]byte("RESPONSE"))
					assert.NoError(t, err)

					return nil
				}),
				"empty": A("empty", []string{"POST"}, 0, func(ctx *Context) error {
					assert.NotEmpty(t, ctx.Model)

					bytes, err := ioutil.ReadAll(ctx.HTTPRequest.Body)
					assert.NoError(t, err)
					assert.Empty(t, bytes)

					return nil
				}),
				"error": A("error", []string{"POST"}, 3, func(ctx *Context) error {
					assert.NotEmpty(t, ctx.Model)

					bytes, err := ioutil.ReadAll(ctx.HTTPRequest.Body)
					assert.Error(t, err)
					assert.Equal(t, []byte{}, bytes)
					assert.Equal(t, serve.ErrBodyLimitExceeded, err)

					ctx.ResponseWriter.WriteHeader(http.StatusRequestEntityTooLarge)

					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		post := tester.Insert(&postModel{
			Title: "Post",
		}).(*postModel).ID()

		// get byte response
		tester.Header["Content-Type"] = "text/plain"
		tester.Header["Accept"] = "text/plain"
		tester.Request("POST", "posts/"+post+"/bytes", "PAYLOAD", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, "text/plain; charset=utf-8", r.Result().Header.Get("Content-Type"), tester.DebugRequest(rq, r))
			assert.Equal(t, "RESPONSE", r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get empty response
		tester.Header["Content-Type"] = ""
		tester.Header["Accept"] = ""
		tester.Request("POST", "posts/"+post+"/empty", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Empty(t, r.Result().Header.Get("Content-Type"), tester.DebugRequest(rq, r))
			assert.Empty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get error
		tester.Request("POST", "posts/"+post+"/error", "PAYLOAD", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusRequestEntityTooLarge, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Empty(t, r.Result().Header.Get("Content-Type"), tester.DebugRequest(rq, r))
			assert.Empty(t, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestSoftDelete(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		// missing field on model
		assert.PanicsWithValue(t, `coal: no or multiple fields flagged as "fire-soft-delete" on "fire.missingSoftDeleteField"`, func() {
			type missingSoftDeleteField struct {
				coal.Base `json:"-" bson:",inline" coal:"models"`
				stick.NoValidation
			}

			tester.Assign("", &Controller{
				Model:      &missingSoftDeleteField{},
				SoftDelete: true,
			})
		})

		// invalid field type
		assert.PanicsWithValue(t, `fire: soft delete field "Foo" for model "fire.invalidSoftDeleteFieldType" is not of type "*time.Time"`, func() {
			type invalidSoftDeleteFieldType struct {
				coal.Base `json:"-" bson:",inline" coal:"models"`
				Foo       int `coal:"fire-soft-delete"`
				stick.NoValidation
			}

			tester.Assign("", &Controller{
				Model:      &invalidSoftDeleteFieldType{},
				SoftDelete: true,
			})
		})

		tester.Assign("", &Controller{
			Model:      &postModel{},
			SoftDelete: true,
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		id := tester.Insert(&postModel{
			Title: "Post 1",
		}).ID()

		// get list of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": [
					{
						"type": "posts",
						"id": "`+id+`",
						"attributes": {
							"title": "Post 1",
							"published": false,
							"text-body": ""
						},
						"relationships": {
							"comments": {
								"data": [],
								"links": {
									"self": "/posts/`+id+`/relationships/comments",
									"related": "/posts/`+id+`/comments"
								}
							},
							"selections": {
								"data": [],
								"links": {
									"self": "/posts/`+id+`/relationships/selections",
									"related": "/posts/`+id+`/selections"
								}
							},
							"note": {
								"data": null,
								"links": {
									"self": "/posts/`+id+`/relationships/note",
									"related": "/posts/`+id+`/note"
								}
							}
						}
					}
				],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get single post
		tester.Request("GET", "posts/"+id, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+id+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/comments",
								"related": "/posts/`+id+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/selections",
								"related": "/posts/`+id+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+id+`/relationships/note",
								"related": "/posts/`+id+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// delete post
		tester.Request("DELETE", "posts/"+id, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNoContent, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.Equal(t, "", r.Body.String(), tester.DebugRequest(rq, r))
		})

		// check post
		post := tester.FindLast(&postModel{}).(*postModel)
		assert.NotNil(t, post)
		assert.NotNil(t, post.Deleted)

		// get empty list of posts
		tester.Request("GET", "posts", "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data":[],
				"links": {
					"self": "/posts"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// get missing post
		tester.Request("GET", "posts/"+id, "", func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusNotFound, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "404",
						"title": "not found",
						"detail": "resource not found"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// TODO: Test has one and has many relationships.
	})
}

func TestIdempotentCreate(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		// missing field on model
		assert.PanicsWithValue(t, `coal: no or multiple fields flagged as "fire-idempotent-create" on "fire.missingIdempotentCreateField"`, func() {
			type missingIdempotentCreateField struct {
				coal.Base `json:"-" bson:",inline" coal:"models"`
				stick.NoValidation
			}

			tester.Assign("", &Controller{
				Model:            &missingIdempotentCreateField{},
				IdempotentCreate: true,
			})
		})

		// invalid field type
		assert.PanicsWithValue(t, `fire: idempotent create field "Foo" for model "fire.invalidIdempotentCreateFieldType" is not of type "string"`, func() {
			type invalidIdempotentCreateFieldType struct {
				coal.Base `json:"-" bson:",inline" coal:"models"`
				Foo       int `coal:"fire-idempotent-create"`
				stick.NoValidation
			}

			tester.Assign("", &Controller{
				Model:            &invalidIdempotentCreateFieldType{},
				IdempotentCreate: true,
			})
		})

		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model:            &selectionModel{},
			IdempotentCreate: true,
		}, &Controller{
			Model: &noteModel{},
		})

		// missing create token
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {
					"name": "test"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "400",
						"title": "bad request",
						"detail": "missing idempotent create token"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		var id string

		// create selection
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {
					"name": "Selection 1",
					"create-token": "foo123"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			selection := tester.FindLast(&selectionModel{})
			id = selection.ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "selections",
					"id": "`+id+`",
					"attributes": {
						"name": "Selection 1",
						"create-token": "foo123"
					},
					"relationships": {
						"posts": {
							"data": [],
							"links": {
								"self": "/selections/`+id+`/relationships/posts",
								"related": "/selections/`+id+`/posts"
							}
						}
					}
				},
				"links": {
					"self": "/selections/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to create duplicate
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {
					"name": "Selection 1",
					"create-token": "foo123"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusConflict, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "409",
						"title": "conflict",
						"detail": "existing document with same idempotent create token"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// attempt to change create token
		tester.Request("PATCH", "selections/"+id, `{
			"data": {
				"type": "selections",
				"id": "`+id+`",
				"attributes": {
					"create-token": "bar456"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "400",
						"title": "bad request",
						"detail": "idempotent create token cannot be changed"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestConsistentUpdate(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		// missing field on model
		assert.PanicsWithValue(t, `coal: no or multiple fields flagged as "fire-consistent-update" on "fire.missingConsistentUpdateField"`, func() {
			type missingConsistentUpdateField struct {
				coal.Base `json:"-" bson:",inline" coal:"models"`
				stick.NoValidation
			}

			tester.Assign("", &Controller{
				Model:            &missingConsistentUpdateField{},
				ConsistentUpdate: true,
			})
		})

		// invalid field type
		assert.PanicsWithValue(t, `fire: consistent update field "Foo" for model "fire.invalidConsistentUpdateFieldType" is not of type "string"`, func() {
			type invalidConsistentUpdateFieldType struct {
				coal.Base `json:"-" bson:",inline" coal:"models"`
				Foo       int `coal:"fire-consistent-update"`
				stick.NoValidation
			}

			tester.Assign("", &Controller{
				Model:            &invalidConsistentUpdateFieldType{},
				ConsistentUpdate: true,
			})
		})

		tester.Assign("", &Controller{
			Model: &postModel{},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model:            &selectionModel{},
			ConsistentUpdate: true,
		}, &Controller{
			Model: &noteModel{},
		})

		var id string
		var selection *selectionModel

		// create selection
		tester.Request("POST", "selections", `{
			"data": {
				"type": "selections",
				"attributes": {
					"name": "Selection 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			selection = tester.FindLast(&selectionModel{}).(*selectionModel)
			id = selection.ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "selections",
					"id": "`+id+`",
					"attributes": {
						"name": "Selection 1",
						"update-token": "`+selection.UpdateToken+`"
					},
					"relationships": {
						"posts": {
							"data": [],
							"links": {
								"self": "/selections/`+id+`/relationships/posts",
								"related": "/selections/`+id+`/posts"
							}
						}
					}
				},
				"links": {
					"self": "/selections/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// missing update token
		tester.Request("PATCH", "selections/"+id, `{
			"data": {
				"type": "selections",
				"id": "`+id+`",
				"attributes": {}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "400",
						"title": "bad request",
						"detail": "invalid consistent update token"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// invalid update token
		tester.Request("PATCH", "selections/"+id, `{
			"data": {
				"type": "selections",
				"id": "`+id+`",
				"attributes": {
					"update-token": "bar123"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusBadRequest, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "400",
						"title": "bad request",
						"detail": "invalid consistent update token"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update selection
		tester.Request("PATCH", "selections/"+id, `{
			"data": {
				"type": "selections",
				"id": "`+id+`",
				"attributes": {
					"update-token": "`+selection.UpdateToken+`"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			selection = tester.FindLast(&selectionModel{}).(*selectionModel)

			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "selections",
					"id": "`+id+`",
					"attributes": {
						"name": "Selection 1",
						"update-token": "`+selection.UpdateToken+`"
					},
					"relationships": {
						"posts": {
							"data": [],
							"links": {
								"self": "/selections/`+id+`/relationships/posts",
								"related": "/selections/`+id+`/posts"
							}
						}
					}
				},
				"links": {
					"self": "/selections/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})
	})
}

func TestTransactions(t *testing.T) {
	withTester(t, func(t *testing.T, tester *Tester) {
		group := tester.Assign("", &Controller{
			Model: &postModel{},
			Notifiers: L{
				C("foo", Notifier, All(), func(ctx *Context) error {
					if ctx.Model.(*postModel).Title == "FAIL" {
						return xo.F("foo")
					}
					return nil
				}),
			},
		}, &Controller{
			Model: &commentModel{},
		}, &Controller{
			Model: &selectionModel{},
		}, &Controller{
			Model: &noteModel{},
		})

		var errs []string
		group.reporter = func(err error) {
			errs = append(errs, err.Error())
		}

		var id string

		// create post
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "Post 1"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusCreated, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+id+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": ""
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/comments",
								"related": "/posts/`+id+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/selections",
								"related": "/posts/`+id+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+id+`/relationships/note",
								"related": "/posts/`+id+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// create post error
		tester.Request("POST", "posts", `{
			"data": {
				"type": "posts",
				"attributes": {
					"title": "FAIL"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			post := tester.FindLast(&postModel{})
			id = post.ID()

			assert.Equal(t, http.StatusInternalServerError, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "500",
						"title": "internal server error"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update post
		tester.Request("PATCH", "posts/"+id, `{
			"data": {
				"type": "posts",
				"id": "`+id+`",
				"attributes": {
					"text-body": "Post 1 Text"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusOK, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"data": {
					"type": "posts",
					"id": "`+id+`",
					"attributes": {
						"title": "Post 1",
						"published": false,
						"text-body": "Post 1 Text"
					},
					"relationships": {
						"comments": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/comments",
								"related": "/posts/`+id+`/comments"
							}
						},
						"selections": {
							"data": [],
							"links": {
								"self": "/posts/`+id+`/relationships/selections",
								"related": "/posts/`+id+`/selections"
							}
						},
						"note": {
							"data": null,
							"links": {
								"self": "/posts/`+id+`/relationships/note",
								"related": "/posts/`+id+`/note"
							}
						}
					}
				},
				"links": {
					"self": "/posts/`+id+`"
				}
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		// update post
		tester.Request("PATCH", "posts/"+id, `{
			"data": {
				"type": "posts",
				"id": "`+id+`",
				"attributes": {
					"title": "FAIL"
				}
			}
		}`, func(r *httptest.ResponseRecorder, rq *http.Request) {
			assert.Equal(t, http.StatusInternalServerError, r.Result().StatusCode, tester.DebugRequest(rq, r))
			assert.JSONEq(t, `{
				"errors": [
					{
						"status": "500",
						"title": "internal server error"
					}
				]
			}`, r.Body.String(), tester.DebugRequest(rq, r))
		})

		assert.Equal(t, 1, tester.Count(&postModel{}))
		assert.Equal(t, "Post 1", stick.MustGet(tester.Fetch(&postModel{}, coal.MustFromHex(id)), "Title"))

		assert.Equal(t, []string{"foo", "foo"}, errs)
	})
}
