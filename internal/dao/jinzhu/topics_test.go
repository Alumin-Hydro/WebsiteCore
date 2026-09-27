package jinzhu

import (
	"testing"

	"github.com/BZYA-Community/WebsiteCore/internal/core/cs"
	"github.com/BZYA-Community/WebsiteCore/internal/dao/jinzhu/dbr"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicTopicListsExcludePostsGuestsCannotRead(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&dbr.Post{}, &dbr.Tag{}, &dbr.User{}); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	createTag := func(name string, persistedQuotes int64) *dbr.Tag {
		t.Helper()
		tag := &dbr.Tag{Model: &dbr.Model{}, UserID: 1, Tag: name, QuoteNum: persistedQuotes}
		if err := db.Create(tag).Error; err != nil {
			t.Fatalf("create tag %q: %v", name, err)
		}
		return tag
	}
	approved := createTag("approved", 99)
	shared := createTag("shared", 99)
	pending := createTag("pending", 99)
	rejected := createTag("rejected", 99)
	private := createTag("private", 99)
	deleted := createTag("deleted", 99)

	createPost := func(tags string, audit dbr.PostAuditT, visibility dbr.PostVisibleT) *dbr.Post {
		t.Helper()
		post := &dbr.Post{
			Model:       &dbr.Model{},
			UserID:      1,
			Tags:        tags,
			AuditStatus: audit,
			Visibility:  visibility,
		}
		if err := db.Create(post).Error; err != nil {
			t.Fatalf("create post %q: %v", tags, err)
		}
		return post
	}

	createPost("approved,shared", dbr.PostAuditApproved, dbr.PostVisitPublic)
	createPost("shared", dbr.PostAuditApproved, dbr.PostVisitPublic)
	createPost("pending,shared", dbr.PostAuditPending, dbr.PostVisitPublic)
	createPost("rejected", dbr.PostAuditRejected, dbr.PostVisitPublic)
	createPost("private", dbr.PostAuditApproved, dbr.PostVisitPrivate)
	deletedPost := createPost("deleted", dbr.PostAuditApproved, dbr.PostVisitPublic)
	if err := db.Model(deletedPost).Update("is_del", 1).Error; err != nil {
		t.Fatalf("soft-delete post: %v", err)
	}

	service := newTopicService(db).(*topicSrv)
	hot, err := service.GetHotTags(-1, 50, 0)
	if err != nil {
		t.Fatalf("get hot topics: %v", err)
	}
	assertTopics(t, hot, []topicExpectation{
		{id: shared.ID, tag: "shared", quoteNum: 2},
		{id: approved.ID, tag: "approved", quoteNum: 1},
	})

	newest, err := service.GetNewestTags(-1, 50, 0)
	if err != nil {
		t.Fatalf("get newest topics: %v", err)
	}
	assertTopics(t, newest, []topicExpectation{
		{id: shared.ID, tag: "shared", quoteNum: 2},
		{id: approved.ID, tag: "approved", quoteNum: 1},
	})

	for _, hidden := range []*dbr.Tag{pending, rejected, private, deleted} {
		for _, topic := range append(hot, newest...) {
			if topic.ID == hidden.ID {
				t.Fatalf("topic %q from a non-public post leaked into discovery", hidden.Tag)
			}
		}
	}
}

type topicExpectation struct {
	id       int64
	tag      string
	quoteNum int64
}

func assertTopics(t *testing.T, got cs.TagList, want []topicExpectation) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("topic count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i, expected := range want {
		if got[i].ID != expected.id || got[i].Tag != expected.tag || got[i].QuoteNum != expected.quoteNum {
			t.Errorf("topic[%d] = {id:%d tag:%q quote_num:%d}, want {id:%d tag:%q quote_num:%d}",
				i, got[i].ID, got[i].Tag, got[i].QuoteNum,
				expected.id, expected.tag, expected.quoteNum)
		}
	}
}
