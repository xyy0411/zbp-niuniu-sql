package main

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type NiuNiuManager struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement"`
	CreatedAt time.Time `gorm:"column:created_at"`
	NiuID     uuid.UUID `gorm:"column:niu_id;type:varchar(36);uniqueIndex"`
	Status    int       `gorm:"column:status;default:0"`
}

type NewUserInfo struct {
	ID        uint           `gorm:"column:id;primaryKey;autoIncrement"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`

	UserID   int64     `gorm:"column:user_id;index"`
	NiuID    uuid.UUID `gorm:"column:niu_id;type:char(36);index"`
	Length   float64   `gorm:"column:length;default:1"`
	WeiGe    int       `gorm:"column:wei_ge;default:0"`
	MeiYao   int       `gorm:"column:mei_yao;default:0"`
	Artifact int       `gorm:"column:artifact;default:0"`
	ShenJi   int       `gorm:"column:shen_ji;default:0"`
	Buff2    int       `gorm:"column:buff2;default:0"`
	Buff3    int       `gorm:"column:buff3;default:0"`
	Buff4    int       `gorm:"column:buff4;default:0"`
	Buff5    int       `gorm:"column:buff5;default:0"`
}

type NewAuctionInfo struct {
	ID        uint      `gorm:"column:id;primaryKey;autoIncrement"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`

	UserID int64     `gorm:"column:user_id;index"`
	NiuID  uuid.UUID `gorm:"column:niu_id;type:varchar(36);uniqueIndex"`
	Length float64   `gorm:"column:length;default:0.01"`
	Money  int       `gorm:"column:money"`
}

type OldUserInfo struct {
	UID       int64   `gorm:"column:UID"`
	Length    float64 `gorm:"column:Length"`
	UserCount int     `gorm:"column:UserCount"`
	WeiGe     int     `gorm:"column:WeiGe"`
	Philter   int     `gorm:"column:Philter"`
	Artifact  int     `gorm:"column:Artifact"`
	ShenJi    int     `gorm:"column:ShenJi"`
	Buff1     int     `gorm:"column:Buff1"`
	Buff2     int     `gorm:"column:Buff2"`
	Buff3     int     `gorm:"column:Buff3"`
	Buff4     int     `gorm:"column:Buff4"`
	Buff5     int     `gorm:"column:Buff5"`
}

type OldAuctionInfo struct {
	ID     int     `gorm:"primaryKey"`
	UserID int64   `gorm:"column:user_id"`
	Length float64 `gorm:"column:length"`
	Money  int     `gorm:"column:money"`
}

func createUserTableIfNotExists(db *gorm.DB, tableName string) error {
	createSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  updated_at DATETIME,
  deleted_at DATETIME,
  user_id INTEGER,
  niu_id CHAR(36),
  length REAL DEFAULT 1,
  wei_ge INTEGER DEFAULT 0,
  mei_yao INTEGER DEFAULT 0,
  artifact INTEGER DEFAULT 0,
  shen_ji INTEGER DEFAULT 0,
  buff2 INTEGER DEFAULT 0,
  buff3 INTEGER DEFAULT 0,
  buff4 INTEGER DEFAULT 0,
  buff5 INTEGER DEFAULT 0
);`, tableName)

	if err := db.Exec(createSQL).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_user_id_idx ON %s(user_id);", tableName, tableName)).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_niu_id_idx ON %s(niu_id);", tableName, tableName)).Error; err != nil {
		return err
	}
	return nil
}

func createAuctionTableIfNotExists(db *gorm.DB, tableName string) error {
	createSQL := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  updated_at DATETIME,
  user_id INTEGER,
  niu_id VARCHAR(36),
  length REAL DEFAULT 0.01,
  money INTEGER
);`, tableName)

	if err := db.Exec(createSQL).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s_user_id_idx ON %s(user_id);", tableName, tableName)).Error; err != nil {
		return err
	}
	if err := db.Exec(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s_niu_id_uniq ON %s(niu_id);", tableName, tableName)).Error; err != nil {
		return err
	}
	return nil
}

func ensureNiuNiuManagerTable(db *gorm.DB) error {
	if !db.Migrator().HasTable("niu_niu_managers") {
		createSQL := `
CREATE TABLE IF NOT EXISTS niu_niu_managers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at DATETIME,
  niu_id VARCHAR(36),
  status INTEGER DEFAULT 0
);`
		if err := db.Exec(createSQL).Error; err != nil {
			return err
		}
		if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS niu_niu_managers_niu_id_uniq ON niu_niu_managers(niu_id);").Error; err != nil {
			return err
		}
		return nil
	}

	if !db.Migrator().HasColumn(&NiuNiuManager{}, "niu_id") {
		if err := db.Migrator().AddColumn(&NiuNiuManager{}, "NiuID"); err != nil {
			return err
		}
	}
	if !db.Migrator().HasColumn(&NiuNiuManager{}, "status") {
		if err := db.Migrator().AddColumn(&NiuNiuManager{}, "Status"); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	sdb, err := gorm.Open(sqlite.Open("data/niuniu/niuniu.db"), &gorm.Config{})
	if err != nil {
		fmt.Println("连接数据库失败,ERROR:", err)
		return
	}

	if err = ensureNiuNiuManagerTable(sdb); err != nil {
		fmt.Println("ERROR:", err)
		return
	}

	userInfoTables, auctionInfoTables, err := getGroupTables(sdb)
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}

	for _, groupNum := range userInfoTables {
		oldTableName := groupNum
		newTableName := fmt.Sprintf("group_%s_user_info", groupNum)

		err = migrateUserTable(sdb, oldTableName, newTableName)
		if err != nil {
			log.Printf("迁移表 %s 到 %s 失败: %v", oldTableName, newTableName, err)
			continue
		}
		log.Printf("成功迁移表 %s 到 %s", oldTableName, newTableName)
	}

	for _, oldTableName := range auctionInfoTables {
		newTableName := fmt.Sprintf("group_%s_auction_info", oldTableName[8:])

		err := migrateAuctionTable(sdb, oldTableName, newTableName)
		if err != nil {
			log.Printf("迁移表 %s 到 %s 失败: %v", oldTableName, newTableName, err)
			continue
		}
		log.Printf("成功迁移表 %s 到 %s", oldTableName, newTableName)
	}
}

func getGroupTables(sdb *gorm.DB) ([]string, []string, error) {
	var tableNames []string
	err := sdb.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tableNames).Error
	if err != nil {
		return nil, nil, err
	}
	var userInfoTables []string
	var auctionInfoTables []string

	for _, name := range tableNames {
		if name == "sqlite_sequence" || name == "niu_niu_managers" || strings.HasPrefix(name, "group_") {
			continue
		}

		_, err1 := strconv.ParseInt(name, 10, 64)
		if err1 != nil {
			if strings.HasPrefix(name, "auction_") {
				auctionInfoTables = append(auctionInfoTables, name)
			}
			continue
		}
		userInfoTables = append(userInfoTables, name)
	}
	return userInfoTables, auctionInfoTables, nil
}

func migrateUserTable(db *gorm.DB, oldTableName, newTableName string) error {
	var oldUsers []OldUserInfo
	if err := db.Table(oldTableName).Find(&oldUsers).Error; err != nil {
		return fmt.Errorf("读取旧表数据失败: %v", err)
	}

	if !db.Migrator().HasTable(newTableName) {
		if err := createUserTableIfNotExists(db, newTableName); err != nil {
			return fmt.Errorf("创建新表失败: %v", err)
		}
	} else {
		log.Printf("目标表 %s 已存在，直接向其插入数据（不 Drop）", newTableName)
	}

	lock := sync.Mutex{}
	for _, oldUser := range oldUsers {
		lock.Lock()
		niuID := uuid.New()
		lock.Unlock()

		niuManager := NiuNiuManager{
			NiuID:  niuID,
			Status: 0,
		}

		newUser := NewUserInfo{
			UserID:   oldUser.UID,
			NiuID:    niuID,
			Length:   oldUser.Length,
			WeiGe:    oldUser.WeiGe,
			MeiYao:   oldUser.Philter,
			Artifact: oldUser.Artifact,
			ShenJi:   oldUser.ShenJi,
			Buff2:    oldUser.Buff2,
			Buff3:    oldUser.Buff3,
			Buff4:    oldUser.Buff4,
			Buff5:    oldUser.Buff5,
		}

		if err := db.Table(newTableName).Create(&newUser).Error; err != nil {
			return fmt.Errorf("插入数据失败: %v", err)
		}

		if err := db.Create(&niuManager).Error; err != nil {
			return fmt.Errorf("插入niuNiuManager表失败: %v", err)
		}
	}

	if err := db.Migrator().DropTable(oldTableName); err != nil {
		return fmt.Errorf("删除旧表失败: %v", err)
	}

	return nil
}

func migrateAuctionTable(db *gorm.DB, oldTableName, newTableName string) error {
	var oldAuctions []OldAuctionInfo
	if err := db.Table(oldTableName).Find(&oldAuctions).Error; err != nil {
		return fmt.Errorf("读取旧表数据失败: %v", err)
	}

	if !db.Migrator().HasTable(newTableName) {
		if err := createAuctionTableIfNotExists(db, newTableName); err != nil {
			return fmt.Errorf("创建新表失败: %v", err)
		}
	} else {
		log.Printf("目标表 %s 已存在，直接向其插入数据（不 Drop）", newTableName)
	}

	lock := sync.Mutex{}
	for _, oldAuction := range oldAuctions {
		lock.Lock()
		newAuction := NewAuctionInfo{
			UserID: oldAuction.UserID,
			NiuID:  uuid.New(),
			Length: oldAuction.Length,
			Money:  oldAuction.Money,
		}
		lock.Unlock()
		if err := db.Table(newTableName).Create(&newAuction).Error; err != nil {
			return fmt.Errorf("插入数据失败: %v", err)
		}

		niuNiuManager := NiuNiuManager{
			NiuID:  newAuction.NiuID,
			Status: 1,
		}

		if err := db.Model(&NiuNiuManager{}).Create(&niuNiuManager).Error; err != nil {
			return fmt.Errorf("更新niuNiuManager状态失败: %v", err)
		}
	}

	if err := db.Migrator().DropTable(oldTableName); err != nil {
		return fmt.Errorf("删除旧表失败: %v", err)
	}

	return nil
}
