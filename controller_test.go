package fire

import (
	"net/http"
	"testing"

	"github.com/gonfire/jsonapi"
	"github.com/labstack/echo/engine"
	"github.com/labstack/echo/test"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mgo.v2/bson"
)

func TestBasicOperations(t *testing.T) {
	server, db := buildServer()

	// get empty list of posts
	testRequest(server, "GET", "/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data":[],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})

	var id string

	// create post
	testRequest(server, "POST", "/posts", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
		"data": {
			"type": "posts",
			"attributes": {
				"title": "Post 1"
			}
		}
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		post := findLastModel(db, &Post{})
		id = post.ID().Hex()

		assert.Equal(t, http.StatusCreated, r.Status())
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
					}
				}
			},
			"links": {
				"self": "/posts/`+id+`"
			}
		}`, r.Body.String())
	})

	// get list of posts
	testRequest(server, "GET", "/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
						}
					}
				}
			],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})

	// update post
	testRequest(server, "PATCH", "/posts/"+id, map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
		"data": {
			"type": "posts",
			"id": "`+id+`",
			"attributes": {
				"text-body": "Post 1 Text"
			}
		}
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
					}
				}
			},
			"links": {
				"self": "/posts/`+id+`"
			}
		}`, r.Body.String())
	})

	// get single post
	testRequest(server, "GET", "/posts/"+id, map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
					}
				}
			},
			"links": {
				"self": "/posts/`+id+`"
			}
		}`, r.Body.String())
	})

	// delete post
	testRequest(server, "DELETE", "/posts/"+id, map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusNoContent, r.Status())
		assert.Equal(t, "", r.Body.String())
	})

	// get empty list of posts
	testRequest(server, "GET", "/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data":[],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})
}

func TestFiltering(t *testing.T) {
	server, db := buildServer()

	// create posts
	post1 := saveModel(db, &Post{
		Title:     "post-1",
		Published: true,
	}).ID().Hex()
	post2 := saveModel(db, &Post{
		Title:     "post-2",
		Published: false,
	}).ID().Hex()
	post3 := saveModel(db, &Post{
		Title:     "post-3",
		Published: true,
	}).ID().Hex()

	// create selection
	selection := saveModel(db, &Selection{
		PostIDs: []bson.ObjectId{
			bson.ObjectIdHex(post1),
			bson.ObjectIdHex(post2),
			bson.ObjectIdHex(post3),
		},
	}).ID().Hex()

	// get posts with single value filter
	testRequest(server, "GET", "/posts?filter[title]=post-1", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [
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
						}
					}
				}
			],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})

	// get posts with multi value filter
	testRequest(server, "GET", "/posts?filter[title]=post-2,post-3", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [
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
						}
					}
				}
			],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})

	// get posts with boolean
	testRequest(server, "GET", "/posts?filter[published]=true", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [
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
						}
					}
				}
			],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})

	// get posts with boolean
	testRequest(server, "GET", "/posts?filter[published]=false", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [
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
						}
					}
				}
			],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})

	// get to many posts with boolean
	testRequest(server, "GET", "/selections/"+selection+"/posts?filter[published]=false", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [
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
						}
					}
				}
			],
			"links": {
				"self": "/selections/`+selection+`/posts"
			}
		}`, r.Body.String())
	})
}

func TestSorting(t *testing.T) {
	server, db := buildServer()

	// create posts in random order
	post2 := saveModel(db, &Post{
		Title: "post-2",
	}).ID().Hex()
	post1 := saveModel(db, &Post{
		Title: "post-1",
	}).ID().Hex()
	post3 := saveModel(db, &Post{
		Title: "post-3",
	}).ID().Hex()

	// get posts in ascending order
	testRequest(server, "GET", "/posts?sort=title", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [
				{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
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
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
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
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post3+`",
					"attributes": {
						"title": "post-3",
						"published": false,
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
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/selections",
								"related": "/posts/`+post3+`/selections"
							}
						}
					}
				}
			],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})

	// get posts in descending order
	testRequest(server, "GET", "/posts?sort=-title", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [
				{
					"type": "posts",
					"id": "`+post3+`",
					"attributes": {
						"title": "post-3",
						"published": false,
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
							"data": [],
							"links": {
								"self": "/posts/`+post3+`/relationships/selections",
								"related": "/posts/`+post3+`/selections"
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
						}
					}
				},
				{
					"type": "posts",
					"id": "`+post1+`",
					"attributes": {
						"title": "post-1",
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
							"data": [],
							"links": {
								"self": "/posts/`+post1+`/relationships/selections",
								"related": "/posts/`+post1+`/selections"
							}
						}
					}
				}
			],
			"links": {
				"self": "/posts"
			}
		}`, r.Body.String())
	})
}

func TestSparseFieldsets(t *testing.T) {
	server, db := buildServer()

	// create posts
	post := saveModel(db, &Post{
		Title: "Post 1",
	}).ID().Hex()

	// get posts with single value filter
	testRequest(server, "GET", "/posts/"+post+"?fields[posts]=title", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": {
				"type": "posts",
				"id": "`+post+`",
				"attributes": {
					"title": "Post 1"
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
					}
				}
			},
			"links": {
				"self": "/posts/`+post+`"
			}
		}`, r.Body.String())
	})
}

func TestHasManyRelationship(t *testing.T) {
	server, db := buildServer()

	// create existing post & comment
	existingPost := saveModel(db, &Post{
		Title: "Post 1",
	})
	saveModel(db, &Comment{
		Message: "Comment 1",
		PostID:  existingPost.ID(),
	})

	// create new post
	post := saveModel(db, &Post{
		Title: "Post 2",
	}).ID().Hex()

	// get single post
	testRequest(server, "GET", "/posts/"+post, map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
					}
				}
			},
			"links": {
				"self": "/posts/`+post+`"
			}
		}`, r.Body.String())
	})

	// get empty list of related comments
	testRequest(server, "GET", "/posts/"+post+"/comments", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data":[],
			"links": {
				"self": "/posts/`+post+`/comments"
			}
		}`, r.Body.String())
	})

	var comment string

	// create related comment
	testRequest(server, "POST", "/comments", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
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
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		comment = findLastModel(db, &Comment{}).ID().Hex()

		assert.Equal(t, http.StatusCreated, r.Status())
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
		}`, r.Body.String())
	})

	// get list of related comments
	testRequest(server, "GET", "/posts/"+post+"/comments", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
		}`, r.Body.String())
	})

	// get only relationship links
	testRequest(server, "GET", "/posts/"+post+"/relationships/comments", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
		}`, r.Body.String())
	})
}

func TestToOneRelationship(t *testing.T) {
	server, db := buildServer()

	// create posts
	post1 := saveModel(db, &Post{
		Title: "Post 1",
	}).ID().Hex()
	post2 := saveModel(db, &Post{
		Title: "Post 2",
	}).ID().Hex()

	// create comment
	comment1 := saveModel(db, &Comment{
		Message: "Comment 1",
		PostID:  bson.ObjectIdHex(post1),
	}).ID().Hex()

	var comment2 string

	// create relating post
	testRequest(server, "POST", "/comments", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
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
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		comment2 = findLastModel(db, &Comment{}).ID().Hex()

		assert.Equal(t, http.StatusCreated, r.Status())
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
		}`, r.Body.String())
	})

	// get related post
	testRequest(server, "GET", "/comments/"+comment2+"/post", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
					}
				}
			},
			"links": {
				"self": "/comments/`+comment2+`/post"
			}
		}`, r.Body.String())
	})

	// get related post id only
	testRequest(server, "GET", "/comments/"+comment2+"/relationships/post", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": {
				"type": "posts",
				"id": "`+post1+`"
			},
			"links": {
				"self": "/comments/`+comment2+`/relationships/post",
				"related": "/comments/`+comment2+`/post"
			}
		}`, r.Body.String())
	})

	// replace relationship
	testRequest(server, "PATCH", "/comments/"+comment2+"/relationships/post", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
		"data": {
			"type": "comments",
			"id": "`+post2+`"
		}
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusNoContent, r.Status())
		assert.Equal(t, "", r.Body.String())
	})

	// fetch replaced relationship
	testRequest(server, "GET", "/comments/"+comment2+"/relationships/post", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": {
				"type": "posts",
				"id": "`+post2+`"
			},
			"links": {
				"self": "/comments/`+comment2+`/relationships/post",
				"related": "/comments/`+comment2+`/post"
			}
		}`, r.Body.String())
	})

	// unset relationship
	testRequest(server, "PATCH", "/comments/"+comment2+"/relationships/parent", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
			"data": null
		}`, func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusNoContent, r.Status())
		assert.Equal(t, "", r.Body.String())
	})

	// fetch unset relationship
	testRequest(server, "GET", "/comments/"+comment2+"/relationships/parent", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": null,
			"links": {
				"self": "/comments/`+comment2+`/relationships/parent",
				"related": "/comments/`+comment2+`/parent"
			}
		}`, r.Body.String())
	})

	// fetch unset related resource
	testRequest(server, "GET", "/comments/"+comment2+"/parent", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": null,
			"links": {
				"self": "/comments/`+comment2+`/parent"
			}
		}`, r.Body.String())
	})
}

func TestToManyRelationship(t *testing.T) {
	server, db := buildServer()

	// create posts
	post1 := saveModel(db, &Post{
		Title: "Post 1",
	}).ID().Hex()
	post2 := saveModel(db, &Post{
		Title: "Post 2",
	}).ID().Hex()
	post3 := saveModel(db, &Post{
		Title: "Post 3",
	}).ID().Hex()

	var selection string

	// create selection
	testRequest(server, "POST", "/selections", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
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
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		selection = findLastModel(db, &Selection{}).ID().Hex()

		assert.Equal(t, http.StatusCreated, r.Status())
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
		}`, r.Body.String())
	})

	// get related post
	testRequest(server, "GET", "/selections/"+selection+"/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
						}
					}
				}
			],
			"links": {
				"self": "/selections/`+selection+`/posts"
			}
		}`, r.Body.String())
	})

	// get related post ids only
	testRequest(server, "GET", "/selections/"+selection+"/relationships/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
		}`, r.Body.String())
	})

	// update relationship
	testRequest(server, "PATCH", "/selections/"+selection+"/relationships/posts", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
		"data": [
			{
				"type": "comments",
				"id": "`+post3+`"
			}
		]
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusNoContent, r.Status())
		assert.Equal(t, "", r.Body.String())
	})

	// get updated related post ids only
	testRequest(server, "GET", "/selections/"+selection+"/relationships/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
		}`, r.Body.String())
	})

	// add relationship
	testRequest(server, "POST", "/selections/"+selection+"/relationships/posts", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
		"data": [
			{
				"type": "comments",
				"id": "`+post1+`"
			}
		]
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusNoContent, r.Status())
		assert.Equal(t, "", r.Body.String())
	})

	// get related post ids only
	testRequest(server, "GET", "/selections/"+selection+"/relationships/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
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
		}`, r.Body.String())
	})

	// remove relationship
	testRequest(server, "DELETE", "/selections/"+selection+"/relationships/posts", map[string]string{
		"Accept":       jsonapi.MediaType,
		"Content-Type": jsonapi.MediaType,
	}, `{
		"data": [
			{
				"type": "comments",
				"id": "`+post3+`"
			},
			{
				"type": "comments",
				"id": "`+post1+`"
			}
		]
	}`, func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusNoContent, r.Status())
		assert.Equal(t, "", r.Body.String())
	})

	// get empty related post ids list
	testRequest(server, "GET", "/selections/"+selection+"/relationships/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data": [],
			"links": {
				"self": "/selections/`+selection+`/relationships/posts",
				"related": "/selections/`+selection+`/posts"
			}
		}`, r.Body.String())
	})
}

func TestEmptyToManyRelationship(t *testing.T) {
	server, db := buildServer()

	// create posts
	post := saveModel(db, &Post{
		Title: "Post 1",
	}).ID().Hex()

	// create selection
	selection := saveModel(db, &Selection{
		Name: "Selection 1",
	}).ID().Hex()

	// get related posts
	testRequest(server, "GET", "/selections/"+selection+"/posts", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data":[],
			"links": {
				"self": "/selections/`+selection+`/posts"
			}
		}`, r.Body.String())
	})

	// get related selections
	testRequest(server, "GET", "/posts/"+post+"/selections", map[string]string{
		"Accept": jsonapi.MediaType,
	}, "", func(r *test.ResponseRecorder, rq engine.Request) {
		assert.Equal(t, http.StatusOK, r.Status())
		assert.JSONEq(t, `{
			"data":[],
			"links": {
				"self": "/posts/`+post+`/selections"
			}
		}`, r.Body.String())
	})
}
