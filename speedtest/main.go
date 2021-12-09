package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"runtime"
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

	MustExec(db, "drop database test99")
	MustExec(db, "create database test99")

	MustExec(db, "create table test99.addpartition(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.addpartition set tiflash replica 1")
	WaitTableOK(db, "addpartition", 10)

	var wg sync.WaitGroup
	M := 50
	go func() {
		wg.Add(1)
		for lessThan := 40; lessThan < M; lessThan += 1 {
			MustExec(db, "ALTER TABLE test99.addpartition ADD PARTITION (PARTITION pn%v VALUES LESS THAN (%v))", lessThan, lessThan)
			WaitTableOK(db, "addpartition", 60)
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
			WaitTableOK(db, fmt.Sprintf("tb%v", y), 60)
			wg.Done()
		}()
	}

	wg.Wait()
	db.Close()
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
	WaitTableOK(db, "tbl168", 10)


	lessThan := "40"
	MustExec(db, "ALTER TABLE test99.tbl168 ADD PARTITION (PARTITION pn VALUES LESS THAN (%v))", lessThan)
	MustExec(db, "alter table test99.victim166 set tiflash replica 1")
	fmt.Println("Wait 166")
	if b := WaitTableOK(db, "victim166", 15); !b {
		fmt.Println("Error!")
		return
	}
	fmt.Println("Wait 168")
	if b := WaitTableOK(db, "tbl168", 15); !b {
		fmt.Println("Error!")
		return
	}
}

func MustExec(db *sql.DB, f string, args ...interface{}) {
	s := fmt.Sprintf(f, args...)
	_, err := db.Exec(s)
	if err != nil {
		panic(err)
	}
}

func WaitTableOK(db *sql.DB, tbn string, to int) bool {
	tick := 0
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Printf("Normal check %v at %v\n", tbn, tick)
			var x int
			s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = '%v';", tbn)
			row := db.QueryRow(s)
			if err := row.Scan(&x); err != nil {
				panic(err)
			}
			tick += 1
			if x == 1 {
				fmt.Printf("OK check %v at %v\n", tbn, tick)
				return true
			}
			if tick >= to {
				fmt.Println("Fail")
				return false
			}
		}
	}
}

func TestPerformance(C int, T int) {
	runtime.GOMAXPROCS(5)
	ctx := context.Background()
	db := GetDB()
	defer db.Close()

	fCreate := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			*ch <- []string{fmt.Sprintf("create table test99.t%v(z int)", i)}
		}
	}
	AsyncStmt(ctx, "create", 10, db, fCreate)


	collect := make([]time.Duration, 0)
	collect2 := make([]time.Duration, 0)
	fCommander := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			y := i
			*ch <- []string{
				fmt.Sprintf("alter table test99.t%v set tiflash replica 1", y),
				fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = 't%v';", y),
			}
		}
	}
	fRunner := func(index int, s *[]string) {
		var x int
		fmt.Printf("[index %v@%v] Handle %v\n", index, time.Now(), (*s)[0])
		start := time.Now()
		_, err := db.Exec((*s)[0])
		if err != nil {
			panic(err)
		}
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


func main() {
	// Single
	//TestPerformance(1, 10)
	// Multi
	TestPerformance(100, 10)
	//TestPerformance(40, 4)
	//TestPerformance(20, 2)
	//TODO
	//TestOncall3996()

	//TestChannel()
}