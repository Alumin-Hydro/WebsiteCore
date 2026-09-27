package jinzhu

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/BZYA-Community/WebsiteCore/internal/dao/jinzhu/dbr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestPublicTagsByKeywordExcludesHiddenPosts(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "ut_suggest_" + hex.EncodeToString(buf) + "_",
			SingularTable: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&dbr.Post{}, &dbr.Tag{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Migrator().DropTable(&dbr.Post{}, &dbr.Tag{})
	})

	for _, name := range []string{"public-marker", "private-marker", "pending-marker"} {
		if err := db.Create(&dbr.Tag{Model: &dbr.Model{}, Tag: name, QuoteNum: 99}).Error; err != nil {
			t.Fatal(err)
		}
	}
	posts := []*dbr.Post{
		{Model: &dbr.Model{}, Tags: "public-marker", AuditStatus: dbr.PostAuditApproved, Visibility: dbr.PostVisitPublic},
		{Model: &dbr.Model{}, Tags: "private-marker", AuditStatus: dbr.PostAuditApproved, Visibility: dbr.PostVisitPrivate},
		{Model: &dbr.Model{}, Tags: "pending-marker", AuditStatus: dbr.PostAuditPending, Visibility: dbr.PostVisitPublic},
	}
	for _, post := range posts {
		if err := db.Create(post).Error; err != nil {
			t.Fatal(err)
		}
	}

	got, err := publicTagsByKeyword(db, "marker", 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Tag != "public-marker" || got[0].QuoteNum != 1 {
		t.Fatalf("got %#v, want only public-marker with quote_num=1", got)
	}
}
