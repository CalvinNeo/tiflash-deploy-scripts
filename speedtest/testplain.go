package main

import (
	"fmt"
	"time"
)

func TestPlainAddTableReplica() {
	fmt.Println("START TestPlainAddTruncateTable")
	db := GetDB()

	MustExec(db, "drop database if exists test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addreplica(z int)")
	MustExec(db, "alter table test99.addreplica set tiflash replica 1")
	maxTick := 0
	if ok, tick := WaitTableOK(db, "addreplica", 40, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}

	MustExec(db, "alter table test99.addreplica set tiflash replica 2")
	var x int
	s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = '%v'", "addreplica")
	row := db.QueryRow(s)
	if err := row.Scan(&x); err != nil {
		panic(err)
	}
}

func TestPlainAddTruncateTable() {
	fmt.Println("START TestPlainAddTruncateTable")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addtruncatetable(z int)")
	MustExec(db, "alter table test99.addtruncatetable set tiflash replica 2")
	maxTick := 0
	if ok, tick := WaitTableOK(db, "addtruncatetable", 40, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}

func TestPlainAddDropTable() {
	fmt.Println("START TestPlainAddDropTable")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.adddroptable(z int)")
	MustExec(db, "alter table test99.adddroptable set tiflash replica 2")
	maxTick := 0
	if ok, tick := WaitTableOK(db, "adddroptable", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
	RandomWrite(db, "adddroptable", 100, 1)
	MustExec(db, "drop table test99.adddroptable")
	MustExec(db, "flashback table test99.adddroptable")
	if ok, tick := WaitTableOK(db, "adddroptable", 30, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}


func TestPlainAddPartition() {
	fmt.Println("START TestPlainAddPartition")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addpartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10))")
	MustExec(db, "alter table test99.addpartition set tiflash replica 2")
	maxTick := 0
	MustExec(db, "alter table test99.addpartition ADD PARTITION (PARTITION pn40 VALUES LESS THAN (40))")
	if ok, tick := WaitTableOK(db, "addpartition", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}

func TestPlainTruncatePartition() {
	fmt.Println("START TestPlainTruncatePartition")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.truncatepartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10))")
	MustExec(db, "alter table test99.truncatepartition set tiflash replica 2")
	maxTick := 0

	MustExec(db, "insert into test99.truncatepartition VALUES(9)")
	MustExec(db, "alter table test99.truncatepartition truncate PARTITION p0")
	if ok, tick := WaitTableOK(db, "truncatepartition", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}


}

func TestPlainDropPartition() {
	fmt.Println("START TestPlainDropPartition")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.droppartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.droppartition set tiflash replica 2")
	maxTick := 0
	MustExec(db, "alter table test99.droppartition drop partition p0")
	if ok, tick := WaitTableOK(db, "droppartition", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}

func TestPlainSet0() {
	db := GetDB()

	MustExec(db, "drop database if exists test99")
	MustExec(db, "create database test99")
	MustExec(db, "create table test99.r0(z int)")
	MustExec(db, "alter table test99.r0 set tiflash replica 2")
	MustExec(db, "alter table test99.r0 set tiflash replica 0")

	time.Sleep(2 * time.Second)

	var x int
	s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where table_schema = 'test99' and TABLE_NAME = '%v';", "r0")
	row := db.QueryRow(s)
	if err := row.Scan(&x); err != nil {
		panic(err)
	}

	if x != 0 {
		panic("Fail TestPlainSet0")
	}
}

func TestPlain() {
	db := GetDB()
	defer db.Close()
	ChangeGCSafePoint(db, time.Now(), "true", "10m0s")
	ChangeGCSafeState(db, time.Now(), "10m")
	TestPlainSet0()
	TestPlainDropPartition()
	TestPlainAddPartition()
	TestPlainTruncatePartition()
	TestPlainAddTruncateTable()
	TestPlainAddDropTable()
}

func TestPlainAlterTableDDL() {
	db := GetDB()
	MustExec(db, "drop database if exists test99")
	MustExec(db, "create database test99")
	for i := 0; i < 500; i++ {
		MustExec(db, "create table test99.r%v(z int)", i)
	}
	start := time.Now()
	for i := 0; i < 500; i++ {
		MustExec(db, "alter table test99.r%v set tiflash replica 1", i)
	}
	fmt.Printf( "AAA %v", time.Since(start))
}

