package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/atomic"
	"io"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func TestOncall3996(N int, Replica int) bool {
	fmt.Println("START TestOncall3996")
	db := GetDB()

	failed := 0
	maxTick := 0

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addpartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10))")
	MustExec(db, "alter table test99.addpartition set tiflash replica %v", Replica)
	if ok, tick := WaitTableOK(db, "addpartition", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}

	start := time.Now()
	var wg sync.WaitGroup
	S := 40
	M := S + N
	timeout := 100
	go func() {
		wg.Add(1)
		db1 := db
		//defer db1.Close()
		for lessThan := S; lessThan < M; lessThan += 1 {
			MustExec(db1, "ALTER TABLE test99.addpartition ADD PARTITION (PARTITION pn%v VALUES LESS THAN (%v))", lessThan, lessThan)
			if ok, tick := WaitTableOK(db1, "addpartition", timeout, strconv.Itoa(lessThan)); ok {
				if tick > maxTick {
					maxTick = tick
				}
			} else {
				failed += 1
			}
		}
		wg.Done()
	}()

	for i := S; i < M; i += 1 {
		y := i
		wg.Add(1)
		go func() {
			db1 := db
			//defer db1.Close()
			fmt.Printf("Handle %v\n", y)
			MustExec(db1, "create table test99.tb%v(z int)", y)
			MustExec(db1, "alter table test99.tb%v set tiflash replica %v", y, Replica)
			if ok, tick := WaitTableOK(db1, fmt.Sprintf("tb%v", y), timeout, ""); ok {
				if tick > maxTick {
					maxTick = tick
				}
			} else {
				failed += 1
			}
			wg.Done()
		}()
	}

	wg.Wait()
	db.Close()
	fmt.Printf("maxTick %v elapsed %v\n", maxTick, time.Since(start).Seconds())

	return failed == 0
}



func TestPerformanceAddPartition(C int, T int, P int) {
	fmt.Println("START TestPerformanceAddPartition C %v T %v P %v R %v", C, T, P, *ReplicaNum)
	runtime.GOMAXPROCS(T)
	ctx := context.Background()
	db := GetDB()
	defer db.Close()

	fCreate := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			*ch <- []string{fmt.Sprintf("create table test99.t%v(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (-10))", i)}
		}
	}
	AsyncStmt(ctx, "create", 10, db, fCreate)

	fAlter := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			*ch <- []string{
				fmt.Sprintf("alter table test99.t%v set tiflash replica %v", i, *ReplicaNum)}
		}
	}
	AsyncStmt(ctx, "alter", 10, db, fAlter)


	collect := make([]time.Duration, 0)
	collect2 := make([]time.Duration, 0)
	fCommander := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			var s = make([]string, 0)
			y := i
			for j := 0; j < P; j ++ {
				z := j * 10
				ss := fmt.Sprintf("alter table test99.t%v add partition (PARTITION pn%v VALUES LESS THAN (%v))", y, z, z)
				fmt.Printf("Add %v of %v/%v\n", ss, j, P)
				s = append(s, ss)
			}
			ss := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = 't%v';", y)
			s = append(s, ss)
			*ch <- s
		}
	}
	fRunner := func(index int, s *[]string) {
		var x int
		// Run ddl.
		l := len(*s)
		start := time.Now()
		for i := 0; i < l - 1; i++ {
			fmt.Printf("[index %v@%v] Handle %v length %v\n", index, time.Now(), (*s)[0], l)
			_, err := db.Exec((*s)[i])
			if err != nil {
				panic(err)
			}
		}

		// Wait ready
		fmt.Printf("[index %v@%v] Pending %v\n", index, time.Now(), (*s)[0])
		start2 := time.Now()
		for {
			row := db.QueryRow((*s)[l-1])
			if err := row.Scan(&x); err != nil {
				panic(err)
			}

			if x == 1 {
				t := time.Since(start)
				t2 := time.Since(start2)
				fmt.Printf("[index %v@%v] Finish %v Cost(alter+sync) %v Cost(sync) %v\n", index, time.Now(), (*s)[0], t, t2)
				collect = append(collect, t)
				collect2 = append(collect2, t2)
				runtime.Gosched()
				return
			}
			runtime.Gosched()
		}
	}
	elapsed := AsyncStmtEx(ctx, "replica", T, db, fCommander, fRunner)
	Summary(&collect, &collect2, elapsed)
}

func TestSchemaPerformance(C int, T int, Offset int) string {
	fmt.Println("START TestSchemaPerformance total count %v threads %v start from %v replica %v", C, T, Offset)
	runtime.GOMAXPROCS(T)
	ctx := context.Background()
	db := GetDB()
	defer db.Close()

	fCreate := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			*ch <- []string{fmt.Sprintf("create table test99.t%v(z int)", i + Offset)}
		}
	}
	AsyncStmt(ctx, "create", 10, db, fCreate)

	startAlter := time.Now()
	MustExec(db, "alter database test99 set tiflash replica %v", *ReplicaNum)
	costStartAlter := time.Since(startAlter)
	fmt.Printf("alter database cost %v\n", costStartAlter.Seconds())

	time.Sleep(time.Second)

	collect2 := make([]time.Duration, 0)
	fCommander := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			y := i
			*ch <- []string{
				fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = 't%v';", y + Offset),
			}
		}
	}
	fRunner := func(index int, s *[]string) {
		var x int
		// Wait ready
		fmt.Printf("[index %v@%v] Pending %v\n", index, time.Now(), (*s)[0])
		start2 := time.Now()
		for {
			row := db.QueryRow((*s)[0])
			if err := row.Scan(&x); err != nil {
				panic(err)
			}

			if x == 1 {
				t2 := time.Since(start2)
				fmt.Printf("[index %v@%v] Finish %v Cost(sync) %v\n", index, time.Now(), (*s)[0], t2)
				collect2 = append(collect2, t2)
				runtime.Gosched()
				return
			}
			runtime.Gosched()
		}
	}
	elapsed := AsyncStmtEx(ctx, "replica", T, db, fCommander, fRunner)

	return fmt.Sprintf("TestSchemaPerformance C %v T %v O %v R %v\n%v\nAlter Database%v", C, T, Offset, *ReplicaNum, Summary(&collect2, &collect2, elapsed), costStartAlter.Seconds())

}

func TestPerformance(C int, T int, Offset int, Replica int) string {
	fmt.Println("START TestPerformance total count %v threads %v start from %v replica %v", C, T, Offset, Replica)
	runtime.GOMAXPROCS(T)
	ctx := context.Background()
	db := GetDB()
	defer db.Close()

	fCreate := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			*ch <- []string{fmt.Sprintf("create table test99.t%v(z int)", i + Offset)}
		}
	}
	AsyncStmt(ctx, "create", 10, db, fCreate)


	collect := make([]time.Duration, 0)
	collect2 := make([]time.Duration, 0)
	fCommander := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			y := i
			*ch <- []string{
				fmt.Sprintf("alter table test99.t%v set tiflash replica %v", y + Offset, Replica),
				fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = 't%v';", y + Offset),
			}
		}
	}
	fRunner := func(index int, s *[]string) {
		var x int
		// Run ddl.
		fmt.Printf("[index %v@%v] Handle %v\n", index, time.Now(), (*s)[0])
		start := time.Now()
		_, err := db.Exec((*s)[0])
		if err != nil {
			panic(err)
		}
		// Wait ready
		fmt.Printf("[index %v@%v] Pending %v\n", index, time.Now(), (*s)[0])
		start2 := time.Now()
		for {
			row := db.QueryRow((*s)[1])
			if err = row.Scan(&x); err != nil {
				panic(err)
			}

			if x == 1 {
				t := time.Since(start)
				t2 := time.Since(start2)
				fmt.Printf("[index %v@%v] Finish %v Cost(alter+sync) %v Cost(sync) %v\n", index, time.Now(), (*s)[0], t, t2)
				collect = append(collect, t)
				collect2 = append(collect2, t2)
				runtime.Gosched()
				return
			}
			runtime.Gosched()
		}
	}
	elapsed := AsyncStmtEx(ctx, "replica", T, db, fCommander, fRunner)

	return fmt.Sprintf("TestPerformance C %v T %v O %v R %v\n%v", C, T, Offset, Replica, Summary(&collect, &collect2, elapsed))
}

func Summary(collect *[]time.Duration, collect2 *[]time.Duration, elapsed time.Duration) string {
	total := int64(0)
	for _, t := range *collect {
		total += t.Milliseconds()
	}
	total2 := int64(0)
	for _, t := range *collect2 {
		total2 += t.Milliseconds()
	}
	delta := int64(0)
	for i, _ := range *collect {
		delta += (*collect2)[i].Milliseconds() - (*collect)[i].Milliseconds()
	}
	l1 := len(*collect)
	l2 := len(*collect2)
	S1 := fmt.Sprintf("Count(alter+sync) %v\nCount(sync) %v\n", len(*collect), len(*collect2))
	S2 := fmt.Sprintf("Avr(alter+sync)/Avr(sync)/Total  %.3fs/%.3fs/%vs Delta %v l1/l2 %v/%v\n", float64(total*1.0)/float64(l1)/1000.0, float64(total2*1.0)/float64(l2)/1000.0, elapsed.Seconds(), float64(delta*1.0)/float64(l1), l1, l2)
	S := fmt.Sprintf("%v%v", S1, S2)
	fmt.Printf("%v", S)
	return S
}


func TestTruncateTableTombstone(C int, T int) {
	fmt.Println("START TestTruncateTableTombstone")
	db := GetDB()
	defer db.Close()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	TestPerformance(C, T, 0, *ReplicaNum)

	now := time.Now()
	ChangeGCSafePoint(db, now.Add(0 - 24 * time.Hour), "false", "1000m")


	for i := 0; i < C; i++ {
		fmt.Printf("truncate table t%v\n", i)
		MustExec(db, "truncate table test99.t%v", i)
	}
}

func TestOncall3793(C int, N int, T int) {
	fmt.Println("START TestOncall3793")
	db := GetDB()
	defer db.Close()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	TestPerformance(C, T, 0, *ReplicaNum)

	now := time.Now()
	ChangeGCSafePoint(db, now.Add(0 - 24 * time.Hour), "true", "1000m")

	for i := 0; i < C; i++ {
		fmt.Printf("Drop table t%v\n", i)
		MustExec(db, "drop table test99.t%v", i)
	}
	var gc_delete_range int
	var gc_delete_range_done int
	row := db.QueryRow(fmt.Sprintf("select count(*) from mysql.gc_delete_range where ts > %v", TimeToOracleLowerBound(now)))
	if err := row.Scan(&gc_delete_range); err != nil {
		panic(err)
	}
	row = db.QueryRow(fmt.Sprintf("select count(*) from mysql.gc_delete_range_done where ts > %v", TimeToOracleLowerBound(now)))
	if err := row.Scan(&gc_delete_range_done); err != nil {
		panic(err)
	}
	fmt.Printf("gc_delete_range count %v, gc_delete_range_done count %v\n", gc_delete_range, gc_delete_range_done)

	TestPerformance(N, T, C,*ReplicaNum )
	fmt.Printf("gc_delete_range count %v, gc_delete_range_done count %v\n", gc_delete_range, gc_delete_range_done)

	ChangeGCSafePoint(db, now, "false", "10m0s")
}


func TestMultiTiFlash() {
	//// Single
	//TestPerformance(10, 1, 0, 2)
	//// Multi
	//TestPerformance(100, 10, 0, 2)
	//TestPerformance(40, 4, 0, 2)
	//TestPerformance(20, 2, 0, 2)
	//TestPerformance(20, 2, 0, 2)

	TestOncall3996(60, 2)
	//TestOncall3793(200, 2)
}

func GetTotalLine(db *sql.DB, schema string, table string) int {
	s := fmt.Sprintf("select count(*) from %v.%v", schema, table)
	x := 0
	row := db.QueryRow(s)
	if err := row.Scan(&x); err != nil {
		panic(err)
	}
	return x
}

func WriteBigTable(db *sql.DB, schema string, total int) {
	X := strings.Repeat("ABCDEFG", 2000)
	before := GetTotalLine(db, schema, "bigtable")
	for i := before; i < before + total; i++ {
		fmt.Printf("Insert %v\n", i)
		s := fmt.Sprintf("insert into %v.bigtable values (%v,'%v','%v','%v','%v','%v','%v','%v','%v')", schema, i, X,X,X,X,X,X,X,X)
		_, err := db.Exec(s)
		if err != nil {
			panic(err)
		}
	}
	WaitUntil(db, fmt.Sprintf("select count(*) from %v.bigtable", schema), before + total, 100)
}

func MakeBigTable(db *sql.DB, schema string, total int) {
	MustExec(db, "create table %v.bigtable(z int, t1 text, t2 text, t3 text, t4 text, t5 text, t6 text, t7 text, t8 text)", schema)
	WriteBigTable(db, schema, total)
}

func TestBigTableAgain(total int) {
	fmt.Println("START TestBigTableAgain")
	db := GetDB()
	WriteBigTable(db, "test99", total)
}

func TestBigTable(total int){
	fmt.Println("START TestBigTable")
	db := GetDB()
	if !*ReuseDB {
		MustExec(db, "drop database test99")
		MustExec(db, "create database test99")
		MakeBigTable(db, "test99", total)
	}

	time.Sleep(time.Second * 4)
	var avr_size int
	s := fmt.Sprintf("select AVG_ROW_LENGTH from information_schema.TABLES where table_schema='test99' and table_name='bigtable'")
	row := db.QueryRow(s)
	if err := row.Scan(&avr_size); err != nil {
		panic(err)
	}

	size := avr_size * total
	fmt.Printf("!!!! avr_size %v Finish size %v MB\n", avr_size, float64(size) / 1024.0 / 1024.0)
	MustExec(db, "alter table test99.bigtable set tiflash replica %v", *ReplicaNum)
	maxTick := 0
	if ok, tick := WaitTableOK(db, "bigtable", 100, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}

func TestPlacementRules() {
	pd := NewPDHelper(*PDAddr)
	rules, err := pd.GetGroupRules("tiflash")

	if err != nil {
		panic(err)
	}

	oriRule := len(rules)
	fmt.Printf("origin %v\n", oriRule)

	N := 10
	TestOncall3996(N, 1)

	// N new partitions + N new tables + 1 origin partition
	expectedCount := N + N + 1
	rules, err = pd.GetGroupRules("tiflash")
	if err != nil {
		panic(err)
	}
	laterRule := len(rules)
	fmt.Printf("later %v expected %v\n", laterRule, expectedCount)
	for _, r := range rules {
		fmt.Printf("==> In %v\n", r.ID)
	}

	db := GetDB()
	now := time.Now()
	fmt.Printf("begin to remove rules %v\n", time.Now())
	// lastRun -> lastSafePoint -> delete
	// safePoint = now - gcLfeTime
	// should be: safePoint > lastSafePoint
	// lastRun -> lastSafePoint -> delete
	//							-> now - gcLifeTime
	// -1h -> -20m ->
	ChangeGCSafePoint(db, now.Add(-20 * time.Minute), "true", "10m0s")
	ChangeGCSafeState(db, now.Add(-time.Hour), "1s")
	time.Sleep(2 * time.Second)
	fmt.Printf("begin to delete %v\n", time.Now())
	MustExec(db, "drop database test99")
	fmt.Printf("begin to wait %v\n", time.Now())
	time.Sleep(10 * time.Second)
	fmt.Printf("end wait %v\n", time.Now())

	rules, err = pd.GetGroupRules("tiflash")
	if err != nil {
		panic(err)
	}
	deletedRule := len(rules)
	fmt.Printf("delete %v expected %v\n", deletedRule, oriRule)
	for _, r := range rules {
		fmt.Printf("==> In %v\n", r.ID)
	}
	if deletedRule != oriRule {
		panic("Fail TestPlacementRules")
	}
}


func Routine() {
	filename := "result.txt"
	f, _ := os.Create(filename)
	defer f.Close()
	s := ""
	// Single
	s = TestPerformance(10, 1, 0, 1)
	_, _ = io.WriteString(f, s)
	// Multi
	TestPerformance(100, 10, 0, 1)
	_, _ = io.WriteString(f, s)
	TestPerformance(40, 4, 0, 1)
	_, _ = io.WriteString(f, s)
	TestPerformance(20, 2, 0, 1)
	_, _ = io.WriteString(f, s)

	Ns := []int{60}
	Replicas := []int{1, 2}
	for _, Replica := range Replicas {
		for _, N := range Ns{
			if TestOncall3996(N, Replica) {
				io.WriteString(f, fmt.Sprintf("TestOncall3996 %v OK\n", N, Replica))
			}else {
				io.WriteString(f, fmt.Sprintf("TestOncall3996 %v FAIL\n", N, Replica))
			}
		}
		TestOncall3793(200, 100, 10)
	}

}

func TestSetPlacementRule() {
	db := GetDB()

	MustExec(db, "drop database if exists test99")
	MustExec(db, "create database test99")
	MustExec(db, "create table test99.r0(z int)")
	time.Sleep(2 * time.Second)

	SetPlacementRuleForTable("127.0.0.1:4000", "test99", "r0")
}

func TestManyTable(total int, totalPart int, PartCount int){
	fmt.Printf("START TestManyTable %v\n", *ReplicaNum)
	db := GetSession()

	if !*ReuseDB {
		MustExec(db, "drop database if exists testmany")
		MustExec(db, "create database testmany")

		for i := 0; i < total; i++ {
			MustExec(db, "create table testmany.t%v(z int, t text)", i)
		}
		for i := 0; i < totalPart; i++ {
			MustExec(db, "create table testmany.pt%v(z int, t text) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (0))", i)
		}
		for i := 0; i < totalPart; i++ {
			for j := 0; j < PartCount; j ++ {
				lessThan := j * 10 + 10
				MustExec(db, "alter table testmany.pt%v ADD PARTITION (PARTITION pn%v VALUES LESS THAN (%v))", i, lessThan, lessThan)
			}
		}

		MakeBigTable(db, "testmany", 50)
	}

	start := time.Now()
	fmt.Printf("start %v\n", start)
	MustExec(db, "alter database testmany set tiflash replica %v", 1)
	fmt.Printf("[1] since all finish ddl1 %v at %v\n", time.Since(start), time.Now())
	WaitAllTableOKEx(db, "testmany", 1000000, "testmany", 0, 20, 200)
	fmt.Printf("[1] quit cost %v at %v\n", time.Since(start), time.Now())


	MustExec(db, "alter database testmany set tiflash replica 2")
	fmt.Printf("[2] since all finish ddl1 %v at %v\n", time.Since(start), time.Now())
	WaitAllTableOKEx(db, "testmany", 1000000, "testmany", 0, 20, 200)
	fmt.Printf("[2] quit cost %v at %v\n", time.Since(start), time.Now())


	MustExec(db, "alter database testmany set tiflash replica 0")
	fmt.Printf("[0] since all finish ddl1 %v at %v\n", time.Since(start), time.Now())
	WaitAllTableOKEx(db, "testmany", 1000000, "testmany", 0, 20, 200)
	fmt.Printf("[0] quit cost %v at %v\n", time.Since(start), time.Now())
}

func MakeSnapshotMetric() {
	db := GetSession()
	MustExec(db, "drop database if exists testmetric")
	MustExec(db, "create database testmetric")
	for i := 0; i < 100; i++ {
		MustExec(db, "create table testmetric.t%v(z int, t text)", i)
	}
	MustExec(db, "alter database testmetric set tiflash replica %v", 1)
	WaitAllTableOK(db, "testmetric", 10, "testmetric", 0)
	MustExec(db, "drop database testmetric")

	MustExec(db, "drop database if exists testmetric2")
	MustExec(db, "create database testmetric2")
	for i := 0; i < 2; i++ {
		MustExec(db, "create table testmetric2.t%v(z int, t text)", i)
	}
	WaitAllTableOK(db, "testmetric2", 10, "testmetric2", 0)
}

func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}

func Oncall4822() {
	sql := "CREATE TABLE `rpt_report_result_archive` (\n`archive_type` varchar(5) COLLATE utf8mb4_general_ci NOT NULL COMMENT '数据类型（01-归档数据、02-回灌数据）',\n`index_no` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '指标标识',\n`data_date` varchar(8) COLLATE utf8mb4_general_ci NOT NULL COMMENT '数据日期',\n`org_no` varchar(32) COLLATE utf8mb4_general_ci NOT NULL COMMENT '机构标识',\n`currency` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '币种',\n`index_val` decimal(20,5) DEFAULT NULL COMMENT '指标值',\n`template_id` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '模板ID',\n`sys_time` varchar(32) COLLATE utf8mb4_general_ci DEFAULT NULL COMMENT '操作日期',\nPRIMARY KEY (`archive_type`,`index_no`,`data_date`,`org_no`) /*T![clustered_index] NONCLUSTERED */\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci/*!90000 SHARD_ROW_ID_BITS=4 */ COMMENT='报表结果表（保存 归档、回灌数据）'"
	db := GetSession()
	MustExec(db, "use test")
	if !*ReuseDB {
		MustExec(db, "drop table if exists rpt_report_result_archive")
		MustExec(db, sql)
	}
	MustExec(db, "alter table rpt_report_result_archive set tiflash replica 1")

	insert1 := func () {
		si := fmt.Sprintf("insert into rpt_report_result_archive values ('%v', '%v', '%v', '%v', '%v', %v, '%v', '%v')",
			RandomString(4),
			RandomString(30),
			RandomString(4),
			RandomString(30),
			RandomString(30),
			100,
			RandomString(20),
			RandomString(20),
		)
		MustExec(db, si)
	}

	for i := 0; i < 10000; i ++ {
		insert1()
		if i % 10000 == 0 {
			fmt.Printf("!!! %v\n", i)
		}
	}

	row := db.QueryRow(fmt.Sprintf("select * from rpt_report_result_archive limit 1"))
	var archive_type string
	var index_no string
	var data_date string
	var org_no string
	var currency string
	var index_val float64
	var template_id string
	var sys_time string
	if err := row.Scan(&archive_type, &index_no, &data_date, &org_no, &currency, &index_val, &template_id, &sys_time); err != nil {
		panic(err)
	}
}

func MakeConsistentTable() {
	db := GetSession()
	if !*ReuseDB {
		_, err := db.Exec("DROP DATABASE IF EXISTS db3")
		if err != nil {
			panic(err)
		}

		_, err = db.Exec("CREATE DATABASE db3")
		if err != nil {
			panic(err)
		}
		sql := "CREATE TABLE `db3`.`common_handle4` (\n  `a` bigint(20) NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */,\n  `b` int(11) DEFAULT NULL,\n  `c` varchar(100) NOT NULL,\n  `d` char(100) DEFAULT NULL,\n  `e` float DEFAULT NULL,\n  `g` double DEFAULT NULL,\n  `h` decimal(8,4) DEFAULT NULL,\n  `i` date DEFAULT NULL,\n  `j` datetime DEFAULT NULL,\n  PRIMARY KEY (`a`,`c`) /*T![clustered_index] CLUSTERED */\n);"
		MustExec(db, sql)
		sql = "CREATE TABLE `db3`.`common_handle5` (\n  `a` bigint(20) NOT NULL /*T![auto_rand] AUTO_RANDOM(5) */,\n  `b` int(11) DEFAULT NULL,\n  `c` varchar(100) NOT NULL,\n  `d` char(100) DEFAULT NULL,\n  `e` float DEFAULT NULL,\n  `g` double DEFAULT NULL,\n  `h` decimal(8,4) DEFAULT NULL,\n  `i` date DEFAULT NULL,\n  `j` datetime DEFAULT NULL,\n  PRIMARY KEY (`a`,`c`) /*T![clustered_index] CLUSTERED */\n);"
		MustExec(db, sql)
	}
	db.Exec("use db3")
}

func TestConsistentWithSchemaStatic(){
	fmt.Println("Start TestConsistentWithSchemaStatic")
	MakeConsistentTable()
	db := GetDB()
	for i := 0; i < 9000; i++ {
		MustExec(db, "insert into db3.common_handle4 (a,b,c,d,e,g,h,i,j) values (%v,%v,\"a\",\"a\",1.0,1.0,2.0,\"2008-11-11\",\"2008-11-11 11:11:11\")", i, i)
	}
	MustExec(db, "alter table db3.common_handle4 set tiflash replica %v", *ReplicaNum)
}

func queryWithTx(tx *sql.Tx, q string) ([]string, error) {
	rowValues := make([]string, 0)
	rows, err := tx.Query(q)
	if err != nil {
		return rowValues, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return rowValues, err
	}
	values := make([]interface{}, len(cols))
	results := make([]sql.NullString, len(cols))
	for i := range values {
		values[i] = &results[i]
	}
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			return rowValues, err
		}
		allFields := ""
		for _, v := range results {
			if !v.Valid {
				allFields += "NULL"
				continue
			}
			allFields += v.String
		}
		rowValues = append(rowValues, allFields)
	}
	if err := rows.Err(); err != nil {
		return rowValues, err
	}
	return rowValues, nil
}

func testEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestConsistentWithSchemaDynamic(prelimtotal int, total int){
	fmt.Println("Start TestConsistentWithSchemaDynamic")
	db := GetSession()
	MakeConsistentTable()
	MustExec(db, "set @@allow_auto_random_explicit_insert = true;")
	for i := 0; i < prelimtotal; i++ {
		if i % 1000 == 0 {
			fmt.Printf("finish prelim add %v\n", i)
		}
		MustExec(db, "insert into db3.common_handle4 (a,b,c,d,e,g,h,i,j) values (%v,%v,\"a\",\"a\",1.0,1.0,2.0,\"2008-11-11\",\"2008-11-11 11:11:11\")", i, i)
		MustExec(db, "insert into db3.common_handle5 (a,b,c,d,e,g,h,i,j) values (%v,%v,\"a\",\"a\",1.0,1.0,2.0,\"2008-11-11\",\"2008-11-11 11:11:11\")", i, i)
	}
	MustExec(db, "alter table db3.common_handle4 set tiflash replica %v", *ReplicaNum)
	MustExec(db, "alter table db3.common_handle5 set tiflash replica %v", *ReplicaNum)

	if ok, _ := WaitTableOKWithDBName(db, "db3", "common_handle4", 30, ""); ok {
		fmt.Printf("wait finished \n")
	} else {
		panic("wait fail")
	}

	if ok, _ := WaitTableOKWithDBName(db, "db3", "common_handle5", 30, ""); ok {
		fmt.Printf("wait finished \n")
	} else {
		panic("wait fail")
	}

	time.Sleep(time.Second * 2)
	var wg sync.WaitGroup
	var finished atomic.Bool
	finished.Store(false)
	go func() {
		wg.Add(1)
		db1 := db
		MustExec(db1, "set @@allow_auto_random_explicit_insert = true;")
		for i := prelimtotal; i < prelimtotal + total; i++ {
			// insert into db3.common_handle4 (a,b,c,d,e,g,h,i,j) values (99999,99999,"a","a",1.0,1.0,2.0,"2008-11-11","2008-11-11 11:11:11")
			if i % 1000 == 0 {
				fmt.Printf("finish add %v\n", i)
			}
			MustExec(db1, "insert into db3.common_handle4 (a,b,c,d,e,g,h,i,j) values (%v,%v,\"a\",\"a\",1.0,1.0,2.0,\"2008-11-11\",\"2008-11-11 11:11:11\")", i, i)
			MustExec(db1, "insert into db3.common_handle5 (a,b,c,d,e,g,h,i,j) values (%v,%v,\"a\",\"a\",1.0,1.0,2.0,\"2008-11-11\",\"2008-11-11 11:11:11\")", i, i)
		}
		finished.Store(true)
		wg.Done()
	}()

	for {
		if finished.Load() == true {
			fmt.Printf("finished === \n")
			break
		}
		ctx := context.Background()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			panic("error tx")
		}

		qind1 := fmt.Sprintf("select /*+ use_index(%s,PRIMARY) */  count(*) from %s", "db3.common_handle4", "db3.common_handle4")
		qnoind1 := fmt.Sprintf("select /*+ ignore_index(%s,PRIMARY) */  count(*) from %s", "db3.common_handle4", "db3.common_handle4")
		qind2 := fmt.Sprintf("select /*+ use_index(%s,PRIMARY) */  count(*) from %s", "db3.common_handle5", "db3.common_handle5")
		qnoind2 := fmt.Sprintf("select /*+ ignore_index(%s,PRIMARY) */  count(*) from %s", "db3.common_handle5", "db3.common_handle5")
		qs := []string{qind1, qnoind1, qind2, qnoind2}
		var tso uint64
		for _, q := range qs {
			if err := tx.QueryRow("select @@tidb_current_ts").Scan(&tso); err != nil {
				panic("error tso")
			}
			if _, err = tx.Exec("set @@tidb_isolation_read_engines='tiflash';"); err != nil {
				panic(fmt.Errorf("error tiflash %v", err))
			}
			rowValuesTiFlash, err := queryWithTx(tx, q)
			if err != nil {
				panic(fmt.Errorf("error tiflash %v", err))
			}
			if _, err = tx.Exec("set @@tidb_isolation_read_engines='tikv';"); err != nil {
				panic(fmt.Errorf("error tikv %v", err))
			}
			rowValuesTiKV, err := queryWithTx(tx, q)
			if err != nil {
				panic(fmt.Errorf("error tikv %v", err))
			}
			if !testEq(rowValuesTiKV, rowValuesTiFlash) {
				fmt.Printf("tso %d, tikv: %v, tiflash %v \n", tso, rowValuesTiKV, rowValuesTiFlash)
			} else {
				fmt.Printf("verify successfully tikv %v tiflash %v\n", rowValuesTiKV, rowValuesTiFlash)
			}
		}
		err = tx.Commit()
		if err != nil {
			panic("error commit")
		}
		time.Sleep(time.Second * 1)
	}

	wg.Wait()
	db.Close()
}

func TestProactiveFlushCompactLog() {
	fmt.Println("Start TestProactiveFlushCompactLog")
	db := GetSession()
	
	if *ReuseDB {
	} else {
		Exec(db, "drop database test97")
		Exec(db, "create database test97")
		Exec(db, "create table test97.t(a int)")
	}
	
	MustExec(db, "set @@allow_auto_random_explicit_insert = true;")
	MustExec(db, "alter table test97.t set tiflash replica %v", *ReplicaNum)
	i := 0
	for {
		if i % 10 == 0 {
			fmt.Printf("====> %v\n", i)
		}
		i += 1
		MustExec(db, "insert into test97.t (a) values (%v)", i)
		time.Sleep(time.Millisecond * 10000)
	}
}

func main() {
	flag.Parse()
	TestProactiveFlushCompactLog();
	// TestConsistentWithSchemaDynamic(*PrelimRowSize, *RowSize)
	// Oncall4822()

	//TestManyTable(1000, 10, 100)
	//TestManyTable(5000, 50, 100)
	// TestPDRuleMultiSession(5, 1, false, 100)
	//TestSchemaPerformance(1000, 1, 1, 1)
	//SetPlacementRuleForTable(os.Args[1], os.Args[2], os.Args[3])
	//TestPlainAlterTableDDL()

	//TestSetPlacementRule()
	//TestPlainSet0()
	//TestPlainAddTableReplica()
	//TestMultiSession(5, 1, true, 500)
	//TestPlain()
	//TestPlacementRules()
	//PrintPD()
	//TestTruncateTableTombstone(40, 4, 1)
	//TestBigTable(*RowSize)
	//TestBigTableAgain(*RowSize)
	//TestBigTable(false, 30000, 1)

	// TestPlain()

	// 50 table add 2 partition with 10 threads
	//TestPerformanceAddPartition(50, 10, 2, 1)

	//TestPerformanceAddPartition(50, 10, 2, 2)

	//TestMultiTiFlash()
	//TestPlainSet0()
}
