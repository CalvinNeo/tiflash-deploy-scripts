package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"math/rand"
	"net/http"
	"os"
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

	//_, err = db.Exec("DROP DATABASE IF EXISTS test99 if exists")
	//if err != nil {
	//	panic(err)
	//}
	//
	//_, err = db.Exec("CREATE DATABASE test99")
	//if err != nil {
	//	panic(err)
	//}
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

func Exec(db *sql.DB, f string, args ...interface{}) error {
	s := fmt.Sprintf(f, args...)
	fmt.Printf("MustExec %v\n", s)
	_, err := db.Exec(s)
	return err
}

func MustExec(db *sql.DB, f string, args ...interface{}) {
	s := fmt.Sprintf(f, args...)
	fmt.Printf("MustExec %v\n", s)
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

func WaitAllTableOK(db *sql.DB, dbn string, to int, tag string) bool {
	tick := 0
	for {
		select {
		case <-time.After(1 * time.Second):
			var x int
			s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 0 and table_schema = '%v';", dbn)
			row := db.QueryRow(s)
			if err := row.Scan(&x); err != nil {
				panic(err)
			}
			tick += 1
			if x == 0 {
				fmt.Printf("OK check db %v tag %v retry %v\n", dbn, tag, tick)
				return true
			}
			if tick >= to {
				fmt.Printf("Fail db %v count %v tag %v retry %v\n", dbn, x, tag, tick)
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

func TestPerformance(C int, T int, Offset int, Replica int) string {
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

	fmt.Println("START ", )
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


func ChangeGCSafeState(db *sql.DB, last time.Time, interval string) {
	gcTimeFormat := "20060102-15:04:05 -0700 MST"
	lastSafePoint := last.Format(gcTimeFormat)

	s := `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_last_run_time', '%[1]s', '')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`

	s = fmt.Sprintf(s, lastSafePoint)
	fmt.Printf("lastRun %v\n", s)
	MustExec(db, s)

	s = `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_run_interval', '%[1]s', '')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`

	s = fmt.Sprintf(s, interval)
	fmt.Printf("Interval %v\n", s)
	MustExec(db, s)
}

func ChangeGCSafePoint(db *sql.DB, t time.Time, enable string, lifeTime string) {
	gcTimeFormat := "20060102-15:04:05 -0700 MST"
	lastSafePoint := t.Format(gcTimeFormat)
	s := `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_safe_point', '%[1]s', '')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`

	s = fmt.Sprintf(s, lastSafePoint)
	fmt.Printf("lastSafePoint %v\n", s)
	MustExec(db, s)


	s = `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_enable','%[1]s','')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`
	s = fmt.Sprintf(s, enable)
	fmt.Printf("enable %v\n", s)
	MustExec(db, s)


	s = `INSERT HIGH_PRIORITY INTO mysql.tidb VALUES ('tikv_gc_life_time','%[1]s','')
			       ON DUPLICATE KEY
			       UPDATE variable_value = '%[1]s'`
	s = fmt.Sprintf(s, lifeTime)
	fmt.Printf("lifeTime %v\n", s)
	MustExec(db, s)
}

func TestTruncateTableTombstone(C int, T int, Replica int) {
	fmt.Println("START TestTruncateTableTombstone")
	db := GetDB()
	defer db.Close()

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
	defer db.Close()

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

func RandomWrite(db *sql.DB, table string, n int, start int) {
	for i := start; i < start + n; i++{
		MustExec(db, "insert into test99.%v values (%v)", table, i)
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


func TestBigTable(reuse bool, total int, Replica int){
	fmt.Println("START TestBigTable")
	db := GetDB()
	if !reuse {

		MustExec(db, "drop database test99")
		MustExec(db, "create database test99")

		MustExec(db, "create table test99.bigtable(z int, t text)")

		X := strings.Repeat("ABCDEFG", 2000)

		for i := 0; i < total; i++ {
			fmt.Printf("Insert %v\n", i)
			MustExec(db, fmt.Sprintf("insert into test99.bigtable values (%v,'%v')", i, X))
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
	MustExec(db, "alter table test99.bigtable set tiflash replica %v", Replica)
	maxTick := 0
	if ok, tick := WaitTableOK(db, "bigtable", 100, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}

type PDHelper struct {
	PDAddr string
	InternalHTTPClient *http.Client
	InternalHTTPSchema string
}

func NewPDHelper(addr string) *PDHelper{
	return &PDHelper{
		PDAddr: addr,
		InternalHTTPSchema: "http",
		InternalHTTPClient: http.DefaultClient,
	}
}

type Constraints []Constraint

type ConstraintOp string

// Constraint is used to filter store when trying to place peer of a region.
type Constraint struct {
	Key    string       `json:"key,omitempty"`
	Op     ConstraintOp `json:"op,omitempty"`
	Values []string     `json:"values,omitempty"`
}

type PeerRoleType string

// TiFlashRule extends Rule with other necessary fields.
type TiFlashRule struct {
	GroupID        string       `json:"group_id"`
	ID             string       `json:"id"`
	Index          int          `json:"index,omitempty"`
	Override       bool         `json:"override,omitempty"`
	StartKeyHex    string       `json:"start_key"`
	EndKeyHex      string       `json:"end_key"`
	Role           PeerRoleType `json:"role"`
	Count          int          `json:"count"`
	Constraints    Constraints  `json:"label_constraints,omitempty"`
	LocationLabels []string     `json:"location_labels,omitempty"`
	IsolationLevel string       `json:"isolation_level,omitempty"`
}

// GetGroupRules to get all placement rule in a certain group.
func (h *PDHelper) GetGroupRules(group string) ([]TiFlashRule, error) {
	pdAddr := h.PDAddr

	getURL := fmt.Sprintf("%s://%s/pd/api/v1/config/rules/group/%s",
		h.InternalHTTPSchema,
		pdAddr,
		group,
	)

	resp, err := h.InternalHTTPClient.Get(getURL)
	if err != nil {
		return nil, errors.New("fail get")
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {

		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("GetGroupRules returns error")
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, errors.New("fail read")
	}

	var rules []TiFlashRule
	err = json.Unmarshal(buf.Bytes(), &rules)
	if err != nil {
		return nil, errors.New("fail parse")
	}

	return rules, nil
}

func (h *PDHelper) ClearAllRules(group string) {
	rules, err := h.GetGroupRules("tiflash")
	if err != nil {
		panic(err)
	}
	for _, r := range rules {
		h.DeletePlacementRule(group, r.ID)
	}
}

func (h *PDHelper) DeletePlacementRule(group string, ruleID string) error {
	deleteURL := fmt.Sprintf("%s://%s/pd/api/v1/config/rule/%v/%v",
		h.InternalHTTPSchema,
		h.PDAddr,
		group,
		ruleID,
	)

	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return err
	}

	resp, err := h.InternalHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return errors.New("DeletePlacementRule returns error")
	}
	return nil
}

func (h *PDHelper) GetGroupRulesCount(group string) int  {
	rules, err := h.GetGroupRules("tiflash")
	if err != nil {
		panic(err)
	}
	return len(rules)
}

func PrintPD() {
	pd := NewPDHelper("127.0.0.1:2379")
	rules, err := pd.GetGroupRules("tiflash")

	if err != nil {
		panic(err)
	}

	rules, err = pd.GetGroupRules("tiflash")
	if err != nil {
		panic(err)
	}
	laterRule := len(rules)
	fmt.Printf("count %v\n", laterRule)
	for _, r := range rules {
		fmt.Printf("==> In %v\n", r.ID)
	}
}

func TestPlacementRules() {
	pd := NewPDHelper("127.0.0.1:2379")
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

func checkFileIsExist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
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
		TestOncall3793(200, 100, 10, 1)
	}

}

type Table struct {
	Dropped bool
	RuleCount int
	PartitionRuleCount *map[int]int
	PartitionDropped *map[int]bool
}

type Tables struct {
	sync.Mutex

	Ts map[int]*Table
	Replica int
}

func (t *Tables) AddTable(pd *PDHelper, db *sql.DB, partition bool) []string {
	t.Lock()
	defer t.Unlock()

	n := len(t.Ts)
	if partition {
		m := make(map[int]int)
		m[0] = 1
		m2 := make(map[int]bool)
		m2[0] = false
		t.Ts[n] = &Table{
			Dropped:            false,
			RuleCount:          1,
			PartitionRuleCount: &m,
			PartitionDropped:   &m2,
		}
		ss := []string{fmt.Sprintf("create table test98.t%v (z int) partition by range (z) (partition p0 values less than (0))", n),
			fmt.Sprintf("alter table test98.t%v set tiflash replica %v", n, t.Replica)}
		for _, e := range ss {
			MustExec(db, e)
		}
		t.PrintGather(pd)
		return ss
	} else {
		t.Ts[n] = &Table{
			Dropped:            false,
			RuleCount:          1,
			PartitionRuleCount: nil,
			PartitionDropped:   nil,
		}
		ss := []string{fmt.Sprintf("create table test98.t%v (z int)", n),
			fmt.Sprintf("alter table test98.t%v set tiflash replica %v", n, t.Replica)}
		for _, e := range ss {
			MustExec(db, e)
		}
		t.PrintGather(pd)
		return ss
	}
}

func (t *Tables) TruncateTable(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if !v.Dropped {
			if v.PartitionRuleCount != nil {
				for kk, vv := range *v.PartitionDropped {
					if !vv {
						(*v.PartitionRuleCount)[kk] += 1
					}
				}
			}else{
				v.RuleCount += 1
			}
			s := fmt.Sprintf("truncate table test98.t%v", k)
			MustExec(db, s)
			t.PrintGather(pd)
			return s
		}
	}
	return ""
}

func (t *Tables) AddPartition(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if v.PartitionRuleCount != nil && !v.Dropped {
			n := len(*(v.PartitionRuleCount))
			(*(v.PartitionRuleCount))[n] = 1
			(*(v.PartitionDropped))[n] = false
			s := fmt.Sprintf("alter table test98.t%v add partition (partition p%v values less than (%v))", k, n, n * 10)
			MustExec(db, s)
			t.PrintGather(pd)
			return s
		}
	}
	return ""
}

func (t *Tables) TruncatePartition(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if v.PartitionRuleCount != nil && !v.Dropped {
			for kk, vv := range *v.PartitionDropped {
				if !vv {
					(*v.PartitionRuleCount)[kk] += 1
					s := fmt.Sprintf("alter table test98.t%v truncate partition p%v", k, kk)
					MustExec(db, s)
					t.PrintGather(pd)
					return s
				}
			}
		}
	}
	return ""
}

func (t *Tables) DropPartition(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if v.PartitionRuleCount != nil && !v.Dropped {
			for kk, vv := range *v.PartitionDropped {
				if !vv {
					(*v.PartitionDropped)[kk] = true
					s := fmt.Sprintf("alter table test98.t%v drop partition p%v", k, kk)
					// Cannot remove all partitions, use DROP TABLE instead
					if err := Exec(db, s); err != nil {
						(*v.PartitionDropped)[kk] = false
					}
					t.PrintGather(pd)
					return s
				}
			}
		}
	}
	return ""
}

func (t *Tables) DropTable(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if !v.Dropped {
			t.Ts[k].Dropped = true
			s := fmt.Sprintf("drop table test98.t%v", k)
			MustExec(db, s)
			t.PrintGather(pd)
			return s
		}
	}
	return ""
}

func (t *Tables) FlashbackTable(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if v.Dropped {
			t.Ts[k].Dropped = false
			s := fmt.Sprintf("flashback table test98.t%v", k)
			MustExec(db, s)
			t.PrintGather(pd)
			return s
		}
	}
	return ""
}

func (t *Tables) Gather() int {
	rules := 0
	for _, v := range t.Ts {
		if v.PartitionRuleCount != nil {
			for _, vv := range *v.PartitionRuleCount {
				rules += vv
			}
		} else {
			rules += v.RuleCount
		}
	}
	return rules
}

func (t *Tables) Check(delta int) bool {
	expected := t.Gather()
	fmt.Printf("actual %v expected %v\n", delta, expected)
	return true
}

func (t *Tables) PrintGather(pd *PDHelper){
	e := t.Gather()
	a := pd.GetGroupRulesCount("tiflash")
	fmt.Printf("---> expected %v actual %v delta %v \n", e, a, e - a)
}

func TestPDRuleMultiSession(T int, Replica int) {
	dbm := GetDB()
	MustExec(dbm, "drop database if exists test98")
	MustExec(dbm, "create database test98")
	ChangeGCSafePoint(dbm, time.Now().Add(0 - 24 * time.Hour), "false", "1000m")
	defer dbm.Close()
	time.Sleep(2 * time.Second)

	pd := NewPDHelper("127.0.0.1:2379")
	pd.ClearAllRules("tiflash")

	origin := pd.GetGroupRulesCount("tiflash")
	tables := Tables{
		Ts: make(map[int]*Table),
		Replica: Replica,
	}

	var wg sync.WaitGroup
	for t := 0; t < T; t++ {
		wg.Add(1)
		y := t
		go func(index int) {
			db := GetDB()
			defer db.Close()

			tables.AddTable(pd, db, false) // 1
			tables.AddTable(pd, db,true) // 1
			tables.AddPartition(pd, db) // 1

			for i := 0; i < 25; i ++ {
				in := []int{0,1,2,3,4,5,6,7}
				randomIndex := rand.Intn(len(in))
				pick := in[randomIndex]
				if pick == 0 {
					tables.AddTable(pd, db, false)
				} else if pick == 1 {
					tables.AddTable(pd, db, true)
				} else if pick == 2 {
					tables.AddPartition(pd, db)
				} else if pick == 3 {
					tables.DropPartition(pd, db)
				} else if pick == 4 {
					tables.DropTable(pd, db)
				} else if pick == 5 {
					tables.TruncatePartition(pd, db)
				} else if pick == 6 {
					tables.TruncateTable(pd, db)
				} else if pick == 7 {
					tables.FlashbackTable(pd, db)
				}
			}

			wg.Done()
		}(y)
	}
	wg.Wait()

	if ok := WaitAllTableOK(dbm, "test98", 20, "all"); !ok {
		panic("Some table not ready")
	}

	later := pd.GetGroupRulesCount("tiflash")
	tables.Check(later - origin)

	z, _ := pd.GetGroupRules("tiflash")
	for _, x := range z {
		fmt.Printf("===> %v %v %v\n", x.ID, x.GroupID, x.Constraints)
	}

	//var x int
	//s := fmt.Sprintf("SELECT TABLE_NAME,TIDB_TABLE_ID FROM information_schema.tables where table_schema = 'test98';")
	//row := dbm.QueryRow(s)
	//if err := row.Scan(&x); err != nil {
	//	panic(err)
	//}
	//
	//s := fmt.Sprintf("SELECT TABLE_NAME, FROM information_schema.tables where table_schema = 'test98';")
	//row := dbm.QueryRow(s)
	//if err := row.Scan(&x); err != nil {
	//	panic(err)
	//}
}

func SetPlacementRuleForTable(dblink string, schema string, table string) {
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(%v)/", dblink))
	if err != nil {
		panic(err)
	}
	s := fmt.Sprintf("SELECT TIDB_TABLE_ID from information_schema.tables where TABLE_SCHEMA='%v' and TABLE_NAME='%v'", schema, table)

	var x int64
	row := db.QueryRow(s)
	if err = row.Scan(&x); err != nil {
		panic(err)
	}

	rule := MakeNewRule(x, 1, []string{})
	j, _ := json.Marshal(rule)
	buf := bytes.NewBuffer(j)

	fmt.Printf("%v", buf)
}

func makeBaseRule() TiFlashRule {
	return TiFlashRule{
		GroupID:  "tiflash",
		ID:       "",
		Index:    120,
		Override: false,
		Role:     "learner",
		Count:    2,
		Constraints: []Constraint{
			{
				Key:    "engine",
				Op:     "in",
				Values: []string{"tiflash"},
			},
		},
	}
}
var (
	tablePrefix     = []byte{'t'}
	recordPrefixSep = []byte("_r")
	indexPrefixSep  = []byte("_i")
	metaPrefix      = []byte{'m'}
)

func GenTableRecordPrefix(tableID int64) []byte {
	buf := make([]byte, 0, len(tablePrefix)+8+len(recordPrefixSep))
	return appendTableRecordPrefix(buf, tableID)
}
func EncodeTablePrefix(tableID int64) []byte {
	var key []byte
	key = append(key, tablePrefix...)
	key = EncodeInt(key, tableID)
	return key
}
func EncodeInt(b []byte, v int64) []byte {
	var data [8]byte
	u := EncodeIntToCmpUint(v)
	binary.BigEndian.PutUint64(data[:], u)
	return append(b, data[:]...)
}
func EncodeIntToCmpUint(v int64) uint64 {
	return uint64(v) ^ signMask
}
func appendTableRecordPrefix(buf []byte, tableID int64) []byte {
	buf = append(buf, tablePrefix...)
	buf = EncodeInt(buf, tableID)
	buf = append(buf, recordPrefixSep...)
	return buf
}

const signMask uint64 = 0x8000000000000000

// MakeNewRule creates a pd rule for TiFlash.
func MakeNewRule(ID int64, Count uint64, LocationLabels []string) *TiFlashRule {
	ruleID := fmt.Sprintf("table-%v-r", ID)
	startKey := GenTableRecordPrefix(ID)
	endKey := EncodeTablePrefix(ID + 1)
	startKey = EncodeBytes([]byte{}, startKey)
	endKey = EncodeBytes([]byte{}, endKey)

	ruleNew := makeBaseRule()
	ruleNew.ID = ruleID
	ruleNew.StartKeyHex = hex.EncodeToString(startKey)
	ruleNew.EndKeyHex = hex.EncodeToString(endKey)
	ruleNew.Count = int(Count)
	ruleNew.LocationLabels = LocationLabels

	return &ruleNew
}

const (
	encGroupSize = 8
	encMarker    = byte(0xFF)
	encPad       = byte(0x0)
)

var (
	pads = make([]byte, encGroupSize)
)

func EncodeBytes(b []byte, data []byte) []byte {
	// Allocate more space to avoid unnecessary slice growing.
	// Assume that the byte slice size is about `(len(data) / encGroupSize + 1) * (encGroupSize + 1)` bytes,
	// that is `(len(data) / 8 + 1) * 9` in our implement.
	dLen := len(data)
	reallocSize := (dLen/encGroupSize + 1) * (encGroupSize + 1)
	result := reallocBytes(b, reallocSize)
	for idx := 0; idx <= dLen; idx += encGroupSize {
		remain := dLen - idx
		padCount := 0
		if remain >= encGroupSize {
			result = append(result, data[idx:idx+encGroupSize]...)
		} else {
			padCount = encGroupSize - remain
			result = append(result, data[idx:]...)
			result = append(result, pads[:padCount]...)
		}

		marker := encMarker - byte(padCount)
		result = append(result, marker)
	}

	return result
}

func reallocBytes(b []byte, n int) []byte {
	newSize := len(b) + n
	if cap(b) < newSize {
		bs := make([]byte, len(b), newSize)
		copy(bs, b)
		return bs
	}

	// slice b has capability to store n bytes
	return b
}

func TestSetPlacementRule() {
	db := GetDB()

	MustExec(db, "drop database if exists test99")
	MustExec(db, "create database test99")
	MustExec(db, "create table test99.r0(z int)")
	time.Sleep(2 * time.Second)

	SetPlacementRuleForTable("127.0.0.1:4000", "test99", "r0")
}

func main() {
	SetPlacementRuleForTable(os.Args[1], os.Args[2], os.Args[3])

	//TestSetPlacementRule()
	//TestPlainSet0()
	//TestPlainAddTableReplica()
	//TestPDRuleMultiSession(5, 1)
	//TestPlain()
	//TestPlacementRules()

	//PrintPD()

	//TestTruncateTableTombstone(40, 4, 1)
	//TestBigTable(false, 30000, 1)
	//TestBigTable(false, 30000, 1)

	//TODO

	// TestPlain()

	// 50 table add 2 partition with 10 threads
	//TestPerformanceAddPartition(50, 10, 2, 1)

	//TestPerformanceAddPartition(50, 10, 2, 2)

	//TestMultiTiFlash()

	//x := uint64(429772939013390339)
	//y := TimeToOracleUpperBound(GetTimeFromTS(x))
	//fmt.Printf("%v %v %v\n", x, y, x - y)

	//TestPlainSet0()
}