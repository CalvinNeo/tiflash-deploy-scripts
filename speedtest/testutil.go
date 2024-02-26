package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"
)

var DBAddr = flag.String("a", "127.0.0.1:4000", "addr of tidb")
var PDAddr = flag.String("p", "127.0.0.1:2379", "addr of pd")
var ReuseDB = flag.Bool("r", false, "reuse")
var ReplicaNum = flag.Int("num", 1, "replica")
var Print = flag.Bool("print", false, "print")
var RowSize = flag.Int("rowsize", 1000, "row size of a table")
var PrelimRowSize = flag.Int("prelimsize", 1000, "prelim row size of a table")

func GetSession() *sql.DB {
	addr := fmt.Sprintf("root@tcp(%v)/", *DBAddr)
	fmt.Printf("Addr %v\n", addr)
	db, err := sql.Open("mysql", addr)
	if err != nil {
		panic(err)
	}
	return db
}

func GetDB() *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(%v)/", *DBAddr))
	if err != nil {
		panic(err)
	}

	if !*ReuseDB {
		_, err = db.Exec("DROP DATABASE IF EXISTS test99")
		if err != nil {
			panic(err)
		}

		_, err = db.Exec("CREATE DATABASE test99")
		if err != nil {
			panic(err)
		}
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

func Exec(db *sql.DB, f string, args ...interface{}) error {
	s := fmt.Sprintf(f, args...)
	fmt.Printf("MustExec %v\n", s)
	_, err := db.Exec(s)
	return err
}

func MustExec(db *sql.DB, f string, args ...interface{}) {
	s := fmt.Sprintf(f, args...)
	if *Print {
		fmt.Printf("MustExec %v\n", s)
	}
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

func WaitAllTableOKEx(db *sql.DB, dbn string, to int, tag string, noReplica int, gap int, repeat int) bool {
	tick := 0
	for {
		select {
		case <-time.After(time.Duration(gap) * time.Second):
			var x int
			s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 0 and table_schema = '%v';", dbn)
			row := db.QueryRow(s)
			if err := row.Scan(&x); err != nil {
				panic(err)
			}
			tick += gap
			if x == 0 {
				fmt.Printf("OK check db %v tag %v cost sec %v noReplica count %v\n", dbn, tag, tick, noReplica)
				return true
			}
			if tick >= to {
				fmt.Printf("Fail db %v remain %v tag %v cost sec %v noReplica count %v\n", dbn, x, tag, tick, noReplica)
				return false
			}
			if repeat != 0 && tick % repeat == 0 {
				fmt.Printf("Pending db %v remain %v tag %v cost sec %v noReplica count %v\n", dbn, x, tag, tick, noReplica)
			}
		}
	}
}

func WaitAllTableOK(db *sql.DB, dbn string, to int, tag string, noReplica int) bool {
	return WaitAllTableOKEx(db, dbn, to,tag, noReplica, 1, 0)
}

func WaitTableOK(db *sql.DB, tbn string, to int, tag string) (bool, int) {
	return WaitTableOKWithDBName(db, "test99", tbn, to, tag)
}

func WaitTableOKWithDBName(db *sql.DB, dbname string, tbn string, to int, tag string) (bool, int) {
	tick := 0
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Printf("Normal check %v tag %v retry %v\n", tbn, tag, tick)
			var x int
			s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = '%v' and TABLE_NAME = '%v';", dbname, tbn)
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

func checkFileIsExist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
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

