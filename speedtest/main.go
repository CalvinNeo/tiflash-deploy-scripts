package main

import (
	"context"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
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


func TestBigTable(total int){
	fmt.Println("START TestBigTable")
	db := GetDB()
	if !*ReuseDB {

		MustExec(db, "drop database test99")
		MustExec(db, "create database test99")

		MustExec(db, "create table test99.bigtable(z int, t1 text, t2 text, t3 text, t4 text, t5 text, t6 text, t7 text, t8 text)")

		X := strings.Repeat("ABCDEFG", 2000)

		for i := 0; i < total; i++ {
			fmt.Printf("Insert %v\n", i)
			s := fmt.Sprintf("insert into test99.bigtable values (%v,'%v','%v','%v','%v','%v','%v','%v','%v')", i, X,X,X,X,X,X,X,X)
			_, err := db.Exec(s)
			if err != nil {
				panic(err)
			}
		}
		WaitUntil(db, "select count(*) from test99.bigtable", total, 100)

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
	}
	start := time.Now()
	fmt.Printf("start %v\n", start)
	MustExec(db, "alter database testmany set tiflash replica %v", *ReplicaNum)
	fmt.Printf("since all finish ddl1 %v at %v\n", time.Since(start), time.Now())
	WaitAllTableOKEx(db, "testmany", 1000000, "testmany", 0, 20, 200)
	fmt.Printf("quit cost %v at %v\n", time.Since(start), time.Now())
}

func main() {
	flag.Parse()

	//TestManyTable(5000, 50, 100)
	//TestManyTable(5000, 50, 100)
	// TestPDRuleMultiSession(5, 1, false, 100)
	//TestSchemaPerformance(1000, 1, 1, 1)
	//SetPlacementRuleForTable(os.Args[1], os.Args[2], os.Args[3])
	//TestPlainAlterTableDDL()

	//TestSetPlacementRule()
	//TestPlainSet0()
	//TestPlainAddTableReplica()
	//TestPDRuleMultiSession(5, 1)
	//TestPlain()
	//TestPlacementRules()
	//PrintPD()
	//TestTruncateTableTombstone(40, 4, 1)
	TestBigTable(1000)
	//TestBigTable(false, 30000, 1)

	// TestPlain()

	// 50 table add 2 partition with 10 threads
	//TestPerformanceAddPartition(50, 10, 2, 1)

	//TestPerformanceAddPartition(50, 10, 2, 2)

	//TestMultiTiFlash()
	//TestPlainSet0()
}