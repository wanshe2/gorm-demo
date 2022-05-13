package main

import (
	"context"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm/dal/model"
	"gorm/dal/query"
	"math/rand"
	"strconv"
	"time"
)

type User struct {
	UUID    string `gorm:"primary_key"`
	Name    string
	Age     int
	Version int
}

func main() {
	// open Database
	dsn := "root:123456@tcp(127.0.0.1:3306)/gormtest?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// 将 User struct 建表 -> {tableName: users}
	err = db.AutoMigrate(&User{})
	if err != nil {
		panic("autoMigrate error")
	}

	// 指定生成代码的具体(相对)目录，默认为：./query
	// 默认情况下需要使用 WithContext 之后才可以查询，但可以通过设置 gen.WithoutContext 避免这个操作
	g := gen.NewGenerator(gen.Config{
		// 最终 package不能设置为 model ，在有数据库表同步的情况下会产生冲突，若一定要使用可以单独指定model package的新名字
		OutPath:      "dal/query",
		ModelPkgPath: "dal/model", // 默认情况下会跟随OutPath参数，在同目录下生成model目录
		/* Mode: gen.WithoutContext,*/
	})

	// 复用工程原本使用的 SQL 连接配置 db(*gorm.DB)
	// 非必需，但如果需要复用连接时的 gorm.Config 或需要连接数据库同步表信息则必须设置
	g.UseDB(db)

	// peopleTbl := g.GenerateModelAs("users", "People") // 指定对应表格的结构体名称
	// 为指定的结构体或表格生成基础CRUD查询方法，ApplyInterface生成效果的子集
	//g.ApplyBasic(
	//	model.User{},
	//	peopleTbl,
	//)

	// 为指定的数据库表实现除基础方法外的相关方法, 同时也会生成ApplyBasic对应的基础方法
	// 可以认为ApplyInterface方法是ApplyBasic的扩展版
	//g.ApplyInterface(func(model.SearchByTenantMethod, model.UpdateByTenantMethod) {}, // 指定方法interface，可指定多个
	//	model.Order{},
	//	g.GenerateModel("Company"), // 在这里调用也会生成ApplyBasic对应的基础方法
	//)

	// apply basic crud api on structs or table models which is specified by table name with function
	// GenerateModel/GenerateModelAs. And generator will generate table models' code when calling Excute.
	// 想对已有的model生成crud等基础方法可以直接指定model struct ，例如model.User{}
	// 如果是想直接生成表的model和crud方法，则可以指定标名称，例如g.GenerateModel("company")
	// 想自定义某个表生成特性，比如struct的名称/字段类型/tag等，可以指定opt，例如g.GenerateModel("company",gen.FieldIgnore("address")), g.GenerateModelAs("people", "Person", gen.FieldIgnore("address"))
	g.ApplyBasic(g.GenerateModelAs("users", "People"))

	// 冲突时什么也不做
	//db.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.People{})

	// 随机种子
	rand.Seed(time.Now().Unix())

	peoples := []model.People{{UUID: "1", Name: "wan", Age: 18, Version: 1}}
	for i := 1; i <= 100; i++ {
		val := rand.Intn(50)
		people := model.People{UUID: strconv.Itoa(val), Name: "wan", Age: 18, Version: 1}
		peoples = append(peoples, people)
	}

	// Update columns to default value on `id` conflict
	// 对于 MySQL来说，只能判断主键重复，所以第一行是无用的
	// 相当于如果主键重复，不插入，对已有的那个主键所在数据行执行 UPDATE 操作
	// 对于数据更新有两种，一种是表达式 gorm.Expr,一种是默认值 "value"
	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "uuid"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"version": gorm.Expr("version + 1")}),
	}).Create(&peoples)
	//MERGE INTO "users" USING *** WHEN NOT MATCHED THEN INSERT *** WHEN MATCHED THEN UPDATE SET ***; SQL Server
	//INSERT INTO `users` *** ON DUPLICATE KEY UPDATE ***; MySQL

	// Use SQL expression
	//db.Clauses(clause.OnConflict{
	//	Columns:   []clause.Column{{Name: "id"}},
	//	DoUpdates: clause.Assignments(map[string]interface{}{"count": "value"}),
	//}).Create(&users)
	// INSERT INTO `users` *** ON DUPLICATE KEY UPDATE `count`=GREATEST(count, VALUES(count));

	// Update columns to new value on `id` conflict
	//db.Clauses(clause.OnConflict{
	//Columns:   []clause.Column{{Name: "id"}},
	//DoUpdates: clause.AssignmentColumns([]string{"name", "age"}),
	//}).Create(&users)
	// MERGE INTO "users" USING *** WHEN NOT MATCHED THEN INSERT *** WHEN MATCHED THEN UPDATE SET "name"="excluded"."name"; SQL Server
	// INSERT INTO "users" *** ON CONFLICT ("id") DO UPDATE SET "name"="excluded"."name", "age"="excluded"."age"; PostgreSQL
	// INSERT INTO `users` *** ON DUPLICATE KEY UPDATE `name`=VALUES(name),`age=VALUES(age); MySQL

	// Update all columns, except primary keys, to new value on conflict
	//db.Clauses(clause.OnConflict{
	//	UpdateAll: true,
	//}).Create(&users)
	// INSERT INTO "users" *** ON CONFLICT ("id") DO UPDATE SET "name"="excluded"."name", "age"="excluded"."age", ...;

	// apply diy interfaces on structs or table models
	// 如果想给某些表或者model生成自定义方法，可以用ApplyInterface，第一个参数是方法接口，可以参考DIY部分文档定义
	// Method 文件是自己定义的接口，调用这个方法自动实现接口，后面的是一些参数，可以自己改变，也可以不写
	g.ApplyInterface(func(method model.Method) {}, g.GenerateModelAs("users", "People"))
	// SELECT COUNT(*) FROM users GROUP BY version HAVING version=MAX(version)
	// SELECT version, COUNT(*) FROM users GROUP BY version ORDER BY version DESC LIMIT 1;

	// 获取 ctx，通过 Dao 调用方法
	ctx, cancel := context.WithCancel(context.Background())
	u := query.Use(db)
	sz, err := u.WithContext(ctx).People.FindMaxVersionCount()
	if err != nil {
		cancel()
	}
	fmt.Println(sz)
	// 执行并生成代码
	g.Execute()
}
