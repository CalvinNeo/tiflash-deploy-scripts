package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"runtime"
	"strconv"
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

func AsyncStmtEx(ctx context.Context, name string, threads int, db *sql.DB, producer func(*chan []string), consumer func(int, *[]string)) {
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
	fmt.Printf("OK %v %v \v", name, time.Since(start))
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


func TestOncall3996() {
	db := GetDB()

	maxTick := 0

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addpartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.addpartition set tiflash replica 1")
	if ok, tick := WaitTableOK(db, "addpartition", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}

	start := time.Now()
	var wg sync.WaitGroup
	M := 100
	timeout := 40
	go func() {
		wg.Add(1)
		for lessThan := 40; lessThan < M; lessThan += 1 {
			MustExec(db, "ALTER TABLE test99.addpartition ADD PARTITION (PARTITION pn%v VALUES LESS THAN (%v))", lessThan, lessThan)
			if ok, tick := WaitTableOK(db, "addpartition", timeout, strconv.Itoa(lessThan)); ok {
				if tick > maxTick {
					maxTick = tick
				}
			}
		}
		wg.Done()
	}()

	for i := 40; i < M; i += 1 {
		y := i
		wg.Add(1)
		go func() {
			fmt.Printf("Handle %v\n", y)
			MustExec(db, "create table test99.tb%v(z int)", y)
			MustExec(db, "alter table test99.tb%v set tiflash replica 1", y)
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
	fmt.Printf("maxTick %v elapsed %v\n", maxTick, time.Since(start))
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


	lessThan := "40"
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
	_, err := db.Exec(s)
	if err != nil {
		panic(err)
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

func TestPerformance(C int, T int, Offset int) {
	runtime.GOMAXPROCS(5)
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
				fmt.Sprintf("alter table test99.t%v set tiflash replica 1", y + Offset),
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
	AsyncStmtEx(ctx, "replica", T, db, fCommander, fRunner)
	Summary(&collect, &collect2)
}

func Summary(collect *[]time.Duration, collect2 *[]time.Duration) {
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
	fmt.Printf("Avr(alter+sync) %v Avr(sync) %v Delta %v\n", float64(total*1.0)/float64(len(*collect)), float64(total2*1.0)/float64(len(*collect2)), float64(delta*1.0)/float64(len(*collect)))

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

func TestOncall3793() {
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	C := 20

	TestPerformance(C, 1, 0)

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

	deltaC := 40
	TestPerformance(deltaC, 4, C)
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

func TestPlainAddTable() {

}


func TestPlainDropTable() {

}


func TestPlainTruncateTable() {

}


func TestPlainAddPartition() {
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addpartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.addpartition set tiflash replica 1")
	maxTick := 0
	MustExec(db, "alter table test99.addpartition ADD PARTITION (PARTITION pn40 VALUES LESS THAN (40))")
	if ok, tick := WaitTableOK(db, "addpartition", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}

func TestPlainTruncatePartition() {
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.truncatepartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.truncatepartition set tiflash replica 1")
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
	db := GetDB()

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.droppartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.droppartition set tiflash replica 1")
	maxTick := 0
	MustExec(db, "alter table test99.droppartition drop partition p0")
	if ok, tick := WaitTableOK(db, "droppartition", 10, ""); ok {
		if tick > maxTick {
			maxTick = tick
		}
	}
}


func TestPlain() {
	TestPlainDropPartition()
	TestPlainAddPartition()
	TestPlainTruncatePartition()
}

func main() {
	// Single
	//TestPerformance(10, 1, 0)
	// Multi
	//TestPerformance(100, 10, 0)
	//TestPerformance(40, 4, 0)
	//TestPerformance(20, 2, 0)
	//TODO
	//TestOncall3996()

	//TestOncall3793()

	TestPlain()

	//x := uint64(429772939013390339)
	//y := TimeToOracleUpperBound(GetTimeFromTS(x))
	//fmt.Printf("%v %v %v\n", x, y, x - y)
	//TestChannel()
}