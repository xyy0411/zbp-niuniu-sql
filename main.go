package main

import (
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"log"
	"strconv"
	"sync"
	"time"
)

type OldUserInfo struct {
	UID       int64   `gorm:"column:UID"`
	Length    float64 `gorm:"column:Length"`
	UserCount int     `gorm:"column:UserCount"`
	WeiGe     int     `gorm:"column:WeiGe"`    // 伟哥
	Philter   int     `gorm:"column:Philter"`  // 媚药
	Artifact  int     `gorm:"column:Artifact"` // 击剑神器
	ShenJi    int     `gorm:"column:ShenJi"`   // 击剑神稽
	Buff1     int     `gorm:"column:Buff1"`    // 暂定
	Buff2     int     `gorm:"column:Buff2"`    // 暂定
	Buff3     int     `gorm:"column:Buff3"`    // 暂定
	Buff4     int     `gorm:"column:Buff4"`    // 暂定
	Buff5     int     `gorm:"column:Buff5"`    // 暂定
}

type OldAuctionInfo struct {
	ID     int     `gorm:"primaryKey"`
	UserID int64   `gorm:"column:user_id"`
	Length float64 `gorm:"column:length"`
	Money  int     `gorm:"column:money"`
}

type NiuNiuManager struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	NiuID     uuid.UUID `gorm:"type:varchar(36);unique"`
	Status    int       `gorm:"column:status;default:0"` // 0正常 1拍卖 2注销
}

type NewUserInfo struct {
	gorm.Model
	UserID   int64     `gorm:"column:user_id;index"`
	NiuID    uuid.UUID `gorm:"type:char(36);index"`
	Length   float64   `gorm:"default:1"`
	WeiGe    int       `gorm:"default:0"`
	MeiYao   int       `gorm:"default:0"`
	Artifact int       `gorm:"default:0"`
	ShenJi   int       `gorm:"default:0"`
	Buff2    int       `gorm:"default:0"`
	Buff3    int       `gorm:"default:0"`
	Buff4    int       `gorm:"default:0"`
	Buff5    int       `gorm:"default:0"`
}

type NewAuctionInfo struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID int64     `gorm:"column:user_id;index"`
	NiuID  uuid.UUID `gorm:"type:varchar(36);uniqueIndex"`
	Length float64   `gorm:"default:0.01"`
	Money  int
}

func main() {
	sdb, err := gorm.Open(sqlite.Open("data/niuniu/niuniu.db"), &gorm.Config{})
	if err != nil {
		fmt.Println("连接数据库失败,ERROR:", err)
		return
	}

	if err = sdb.AutoMigrate(NiuNiuManager{}); err != nil {
		fmt.Println("ERROR:", err)
		return
	}

	userInfoTables, auctionInfoTables, err := getGroupTables(sdb)
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}

	// 迁移用户表
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

	// 迁移拍卖表
	for _, groupNum := range auctionInfoTables {
		oldTableName := groupNum
		n := groupNum[8:]
		newTableName := fmt.Sprintf("group_%s_auction_info", n)

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

		if name == "sqlite_sequence" || name == "niu_niu_managers" {
			log.Printf("排除系统表:%s\n", name)
			continue
		}

		_, err1 := strconv.ParseInt(name, 10, 64)
		if err1 != nil {
			auctionInfoTables = append(auctionInfoTables, name)
			continue
		}
		userInfoTables = append(userInfoTables, name)
	}
	return userInfoTables, auctionInfoTables, nil
}

func migrateUserTable(db *gorm.DB, oldTableName, newTableName string) error {

	// 从旧表读取所有数据
	var oldUsers []OldUserInfo
	if err := db.Table(oldTableName).Find(&oldUsers).Error; err != nil {
		return fmt.Errorf("读取旧表数据失败: %v", err)
	}

	if db.Migrator().HasTable(newTableName) {
		if err := db.Migrator().DropTable(newTableName); err != nil {
			return fmt.Errorf("删除已存在的新表失败: %v", err)
		}
	}

	if err := db.Table(newTableName).AutoMigrate(&NewUserInfo{}); err != nil {
		return fmt.Errorf("创建新表失败: %v", err)
	}

	lock := sync.Mutex{}
	// 转换并插入数据
	for _, oldUser := range oldUsers {
		lock.Lock()
		niuID := uuid.New()
		lock.Unlock()

		// 插入到niuNiuManager表
		niuManager := NiuNiuManager{
			NiuID:  niuID,
			Status: 0, // 默认状态正常
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

	// 删除旧表
	if err := db.Migrator().DropTable(oldTableName); err != nil {
		return fmt.Errorf("删除旧表失败: %v", err)
	}

	return nil
}

func migrateAuctionTable(db *gorm.DB, oldTableName, newTableName string) error {
	// 从旧表读取所有数据
	var oldAuctions []OldAuctionInfo
	if err := db.Table(oldTableName).Find(&oldAuctions).Error; err != nil {
		return fmt.Errorf("读取旧表数据失败: %v", err)
	}

	if db.Migrator().HasTable(newTableName) {
		if err := db.Migrator().DropTable(newTableName); err != nil {
			return fmt.Errorf("删除已存在的新表失败: %v", err)
		}
	}

	// 创建新表
	if err := db.Table(newTableName).AutoMigrate(&NewAuctionInfo{}); err != nil {
		return fmt.Errorf("创建新表失败: %v", err)
	}

	lock := sync.Mutex{}
	// 转换并插入数据
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

		// 更新niuNiuManager状态为拍卖中(1)
		if err := db.Model(&NiuNiuManager{}).Create(&niuNiuManager).Error; err != nil {
			return fmt.Errorf("更新niuNiuManager状态失败: %v", err)
		}
	}

	// 删除旧表
	if err := db.Migrator().DropTable(oldTableName); err != nil {
		return fmt.Errorf("删除旧表失败: %v", err)
	}

	return nil
}
