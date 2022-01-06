package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func GetDB() *sql.DB {
	db, err := sql.Open("mysql", "root@tcp(127.0.0.1:4000)/")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("DROP DATABASE IF EXISTS test99")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("CREATE DATABASE test99")
	if err != nil {
		panic(err)
	}
	return db
}

func AsyncStmtEx(ctx context.Context, name string, threads int, db *sql.DB, producer func(*chan []string), consumer func(int, *[]string)) time.Duration {
	start := time.Now()
	ch := make(chan []string, threads+10)
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(index int) {
			defer func() {
				wg.Done()
			}()
			for {
				select {
				case s, ok := <-ch:
					if ok {
						consumer(index, &s)
					} else {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
	producer(&ch)
	close(ch)
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("OK %v %v \v", name, elapsed.Seconds())
	return elapsed
}

func AsyncStmtEx2(ctx context.Context, name string, threads int, db *sql.DB, producer func(*chan []string), consumer func(int, *[]string), waiter []func(int)) (time.Duration, map[int]time.Duration) {
	start := time.Now()
	ch := make(chan []string, threads+10)
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(index int) {
			defer func() {
				wg.Done()
			}()
			for {
				select {
				case s, ok := <-ch:
					if ok {
						consumer(index, &s)
					} else {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
	producer(&ch)

	var td map[int] time.Duration
	for index, f := range waiter {
		wg.Add(1)
		go func(i int) {
			defer func() {
				wg.Done()
			}()
			start := time.Now()
			f(i)
			end := time.Since(start)
			td[index] = end
		}(index)
	}

	close(ch)
	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("OK %v %v \v", name, elapsed.Seconds())
	return elapsed, td
}

func AsyncStmt(ctx context.Context, name string, threads int, db *sql.DB, insert func(*chan []string)) {
	f := func(index int, s *[]string) {
		for _, ss := range *s {
			fmt.Printf("Handle %v\n", ss)
			_, err := db.Exec(ss)
			if err != nil {
				panic(err)
			}
		}
	}
	AsyncStmtEx(ctx, name, threads, db, insert, f)
}


func TestOncall3996(N int, Replica int) {
	fmt.Println("START TestOncall3996")
	db := GetDB()

	maxTick := 0

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addpartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
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
		for lessThan := S; lessThan < M; lessThan += 1 {
			MustExec(db, "ALTER TABLE test99.addpartition ADD PARTITION (PARTITION pn%v VALUES LESS THAN (%v))", lessThan, lessThan)
			if ok, tick := WaitTableOK(db, "addpartition", timeout, strconv.Itoa(lessThan)); ok {
				if tick > maxTick {
					maxTick = tick
				}
			}
		}
		wg.Done()
	}()

	for i := S; i < M; i += 1 {
		y := i
		wg.Add(1)
		go func() {
			fmt.Printf("Handle %v\n", y)
			MustExec(db, "create table test99.tb%v(z int)", y)
			MustExec(db, "alter table test99.tb%v set tiflash replica %v", y, Replica)
			if ok, tick := WaitTableOK(db, fmt.Sprintf("tb%v", y), timeout, ""); ok {
				if tick > maxTick {
					maxTick = tick
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()
	db.Close()
	fmt.Printf("maxTick %v elapsed %v\n", maxTick, time.Since(start).Seconds())
}



func TestOncal3996_1() {
	// tbl168 will block victim166
	db := GetDB()
	defer db.Close()

	MustExec(db, "drop table if exists test99.victim166")
	MustExec(db, "create table test99.victim166(z int)")

	MustExec(db, "drop table if exists test99.tbl168")
	MustExec(db, "create table test99.tbl168(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.tbl168 set tiflash replica 1")

	fmt.Println("Wait tbl168")
	maxTick := 0
	if ok, tick := WaitTableOK(db, "tbl168", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}

	S := 40
	lessThan := fmt.Sprintf("%v", S)
	MustExec(db, "ALTER TABLE test99.tbl168 ADD PARTITION (PARTITION pn VALUES LESS THAN (%v))", lessThan)
	MustExec(db, "alter table test99.victim166 set tiflash replica 1")
	fmt.Println("Wait 166")
	if b, tick := WaitTableOK(db, "victim166", 15, ""); !b {
		fmt.Println("Error!")
		return
	} else {
		if tick > maxTick {
			maxTick = tick
		}
	}
	fmt.Println("Wait 168")
	if b, tick := WaitTableOK(db, "tbl168", 15, ""); !b {
		fmt.Println("Error!")
		return
	} else {
		if tick > maxTick {
			maxTick = tick
		}
	}
}

func MustExec(db *sql.DB, f string, args ...interface{}) {
	s := fmt.Sprintf(f, args...)
	// fmt.Printf("MustExec %v\n", s)
	_, err := db.Exec(s)
	if err != nil {
		panic(err)
	}
}

func WaitUntil(db *sql.DB, s string, expected int, to int) bool {
	tick := 0
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Printf("WaitUntil %v retry %v\n", s, tick)
			var x int
			s := fmt.Sprintf(s)
			row := db.QueryRow(s)
			if err := row.Scan(&x); err != nil {
				panic(err)
			}
			tick += 1
			if x == expected {
				return true
			}
			if tick >= to {
				return false
			}
		}
	}
}

func WaitTableOK(db *sql.DB, tbn string, to int, tag string) (bool, int) {
	tick := 0
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Printf("Normal check %v tag %v retry %v\n", tbn, tag, tick)
			var x int
			s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = '%v';", tbn)
			row := db.QueryRow(s)
			if err := row.Scan(&x); err != nil {
				panic(err)
			}
			tick += 1
			if x == 1 {
				fmt.Printf("OK check %v tag %v retry %v\n", tbn, tag, tick)
				return true, tick
			}
			if tick >= to {
				fmt.Printf("Fail table %v tag %v retry %v\n", tbn, tag, tick)
				return false, tick
			}
		}
	}
}

func TestPerformanceAddPartition(C int, T int, P int, Replica int) {
	fmt.Println("START TestPerformanceAddPartition C %v T %v P %v R %v", C, T, P, Replica)
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
				fmt.Sprintf("alter table test99.t%v set tiflash replica %v", i, Replica)}
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

func TestPerformance(C int, T int, Offset int, Replica int) {
	fmt.Println("START TestPerformance C %v T %v O %v R %v", C, T, Offset, Replica)
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
	Summary(&collect, &collect2, elapsed)
}

func Summary(collect *[]time.Duration, collect2 *[]time.Duration, elapsed time.Duration) {
	fmt.Printf("Count(alter+sync) %v\n", len(*collect))
	fmt.Printf("Count(sync) %v\n", len(*collect2))
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
	fmt.Printf("Avr(alter+sync)/Avr(sync)/Total  %.3fs/%.3fs/%vs Delta %v l1/l2 %v/%v\n", float64(total*1.0)/float64(l1)/1000.0, float64(total2*1.0)/float64(l2)/1000.0, elapsed.Seconds(), float64(delta*1.0)/float64(l1), l1, l2)

}

func ChangeGCSafePoint(db *sql.DB, t time.Time, enable string, lifeTime string) {
	gcTimeFormat := "20060102-15:04:05 -0700 MST"
	lastSafePoint := t.Format(gcTimeFormat)
	safePointSQL := `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_safe_point', '%[1]s', '')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`

	safePointSQL = fmt.Sprintf(safePointSQL, lastSafePoint)
	fmt.Printf("lastSafePoint %v\n", safePointSQL)
	MustExec(db, safePointSQL)


	safePointSQL = `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_enable','%[1]s','')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`
	safePointSQL = fmt.Sprintf(safePointSQL, enable)
	fmt.Printf("enable %v\n", safePointSQL)
	MustExec(db, safePointSQL)


	safePointSQL = `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_life_time','%[1]s','')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`
	safePointSQL = fmt.Sprintf(safePointSQL, lifeTime)
	fmt.Printf("lifeTime %v\n", safePointSQL)
	MustExec(db, safePointSQL)
}

func TestTruncateTableTombstone(C int, T int, Replica int) {
	fmt.Println("START TestTruncateTableTombstone")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	TestPerformance(C, T, 0, Replica)

	now := time.Now()
	ChangeGCSafePoint(db, now.Add(0 - 24 * time.Hour), "false", "1000m")


	for i := 0; i < C; i++ {
		fmt.Printf("truncate table t%v\n", i)
		MustExec(db, "truncate table test99.t%v", i)
	}
}

func TestOncall3793(C int, N int, T int, Replica int) {
	fmt.Println("START TestOncall3793")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	TestPerformance(C, T, 0, Replica)

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

	TestPerformance(N, T, C,1 )
	fmt.Printf("gc_delete_range count %v, gc_delete_range_done count %v\n", gc_delete_range, gc_delete_range_done)

	ChangeGCSafePoint(db, now, "false", "10m0s")
}

//var c chan int
//
//func TestChannel() {
//	go func () {
//		for {
//			select {
//				case i, ok := <- c:
//					if ok {
//						fmt.Printf("OK %v\n", i)
//					}else {
//						break
//					}
//					default:
//			}
//		}
//
//		select
//	}()
//
//}


const (
	physicalShiftBits = 18
)

func ExtractPhysical(ts uint64) int64 {
	return int64(ts >> physicalShiftBits)
}

func GetTimeFromTS(ts uint64) time.Time {
	ms := ExtractPhysical(ts)
	return time.Unix(ms/1e3, (ms%1e3)*1e6)
}

func GetPhysicTime(ts uint64) {
	fmt.Printf("Physical time %v\n", GetTimeFromTS(ts))
}

func TimeToOracleLowerBound(t time.Time) uint64 {
	physical := uint64((t.UnixNano() / int64(time.Millisecond)) + 0)
	logical := uint64(0)
	return (physical << uint64(physicalShiftBits)) + logical
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
	//MustExec(db, "truncate table test99.addtruncatetable")
	//if ok, tick := WaitTableOK(db, "addtruncatetable", 10, ""); ok {
	//	if tick > maxTick {
	//		maxTick = tick
	//	}
	//}
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
	//MustExec(db, "drop table test99.adddroptable")
	//if ok, tick := WaitTableOK(db, "adddroptable", 30, ""); ok {
	//	if tick > maxTick {
	//		maxTick = tick
	//	}
	//}
}


func TestPlainAddPartition() {
	fmt.Println("START TestPlainAddPartition")
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addpartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
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

	MustExec(db, "create table test99.truncatepartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
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

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")
	MustExec(db, "create table test99.r0(z int)")
	MustExec(db, "alter table test99.r0 set tiflash replica 2")
	MustExec(db, "alter table test99.r0 set tiflash replica 0")
}

func TestPlain() {
	TestPlainSet0()
	TestPlainDropPartition()
	TestPlainAddPartition()
	TestPlainTruncatePartition()
	TestPlainAddTruncateTable()
	TestPlainAddDropTable()
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


func TestBigTable(reuse bool){
	fmt.Println("START TestBigTable")
	db := GetDB()
	if !reuse {

		MustExec(db, "drop database test99")
		MustExec(db, "create database test99")

		MustExec(db, "create table test99.bigtable(z int, t text)")

		X := strings.Repeat("ABCDEFG", 2000)
		total := 30000
		for i := 0; i < total; i++ {
			fmt.Printf("Insert %v\n", i)
			MustExec(db, fmt.Sprintf("insert into test99.bigtable values (%v,'%v')", i, X))
		}
		WaitUntil(db, "select count(*) from test99.bigtable", total, 100)

	}

	time.Sleep(time.Second * 4)
	var size int
	s := fmt.Sprintf("select DATA_LENGTH as data from information_schema.TABLES where table_schema='test99' and table_name='bigtable'")
	row := db.QueryRow(s)
	if err := row.Scan(&size); err != nil {
		panic(err)
	}

	fmt.Printf("!!!! %v Finish size %v MB\n", size, float64(size) / 1024.0 / 1024.0)
	MustExec(db, "alter table test99.bigtable set tiflash replica 1")
	maxTick := 0
	if ok, tick := WaitTableOK(db, "bigtable", 100, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}


func main() {
	TestTruncateTableTombstone(40, 4, 1)
	//TestBigTable(false)

	//// Single
	//TestPerformance(10, 1, 0, 1)
	//// Multi
	//TestPerformance(100, 10, 0, 1)
	//TestPerformance(40, 4, 0, 1)
	//TestPerformance(20, 2, 0, 1)
	//TODO
	//TestOncall3996(60, 1)

	//TestOncall3793(200, 100, 10, 1)

	//TestPlain()

	// 50 table add 2 partition with 10 threads
	//TestPerformanceAddPartition(50, 10, 2, 1)

	//TestPerformanceAddPartition(50, 10, 2, 2)

	//TestMultiTiFlash()

	//x := uint64(429772939013390339)
	//y := TimeToOracleUpperBound(GetTimeFromTS(x))
	//fmt.Printf("%v %v %v\n", x, y, x - y)
	//TestChannel()

	//TestPlainSet0()
}