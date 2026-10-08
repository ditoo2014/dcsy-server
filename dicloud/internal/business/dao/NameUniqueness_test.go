//go:build cgo

package dao

import (
	"testing"

	"dicloud/internal/business/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openNameUniquenessTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(&model.Warehouse{}, &model.Site{}); err != nil {
		t.Fatalf("migrate name uniqueness tables: %v", err)
	}
	return db
}

func TestWarehouseExistNameTrimsAndExcludesCurrentRecord(t *testing.T) {
	db := openNameUniquenessTestDB(t)
	warehouse := model.Warehouse{Name: "库房甲", WarehouseCode: "WH-001"}
	if err := db.Create(&warehouse).Error; err != nil {
		t.Fatalf("create warehouse: %v", err)
	}
	dao := WarehouseDao{db: db}

	if _, exists := dao.ExistName(model.Warehouse{Name: "  库房甲  "}); !exists {
		t.Fatal("trimmed warehouse name should exist")
	}
	if _, exists := dao.ExistName(model.Warehouse{ID: warehouse.ID, Name: "库房甲"}); exists {
		t.Fatal("current warehouse should be excluded during update validation")
	}
}

func TestSiteNameExistTrimsAndExcludesCurrentRecord(t *testing.T) {
	db := openNameUniquenessTestDB(t)
	site := model.Site{SiteName: "站点甲", SiteSerialNumber: "SITE-001"}
	if err := db.Create(&site).Error; err != nil {
		t.Fatalf("create site: %v", err)
	}
	dao := SiteDAO{db: db}

	if _, exists := dao.SiteNameExist(model.Site{SiteName: "  站点甲  "}); !exists {
		t.Fatal("trimmed site name should exist")
	}
	if _, exists := dao.SiteNameExist(model.Site{ID: site.ID, SiteName: "站点甲"}); exists {
		t.Fatal("current site should be excluded during update validation")
	}
}

func TestSiteSerialNumberExistTrimsAndExcludesCurrentRecord(t *testing.T) {
	db := openNameUniquenessTestDB(t)
	site := model.Site{SiteName: "站点甲", SiteSerialNumber: "SITE-001"}
	if err := db.Create(&site).Error; err != nil {
		t.Fatalf("create site: %v", err)
	}
	otherSite := model.Site{SiteName: "站点乙", SiteSerialNumber: "SITE-002"}
	if err := db.Create(&otherSite).Error; err != nil {
		t.Fatalf("create other site: %v", err)
	}
	dao := SiteDAO{db: db}

	if _, exists := dao.ExistSiteSearial(model.Site{SiteSerialNumber: "  SITE-001  "}); !exists {
		t.Fatal("trimmed site serial number should exist")
	}
	if _, exists := dao.ExistSiteSearial(model.Site{ID: site.ID, SiteSerialNumber: "SITE-001"}); exists {
		t.Fatal("current site should be excluded during update validation")
	}
	if _, exists := dao.ExistSiteSearial(model.Site{ID: site.ID, SiteSerialNumber: "SITE-002"}); !exists {
		t.Fatal("another site's serial number should be rejected during update validation")
	}
}
