package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
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

func AsyncStmtEx(ctx context.Context, name string, threads int, db *sql.DB, insert func(*chan []string), checker func(*[]string)) {
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
						checker(&s)
					} else {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
	insert(&ch)
	close(ch)
	wg.Wait()
	fmt.Printf("OK %v %v \v", name, time.Since(start))
}

func AsyncStmt(ctx context.Context, name string, threads int, db *sql.DB, insert func(*chan []string)) {
	f := func(s *[]string) {
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

func main() {
	// Single
	//TestPerformance(1, 10)
	// Multi
	//TODO
	TestOncal3996()
}


func TestOncal3996() {
	// ddltiflash will block ddltiflash_victim
	db := GetDB()
	defer db.Close()

	MustExec(db, "drop table if exists test99.ddltiflash")
	MustExec(db, "create table test99.ddltiflash(z int) PARTITION BY RANGE(z) (PARTITION p0 VALUES LESS THAN (10),PARTITION p1 VALUES LESS THAN (20), PARTITION p2 VALUES LESS THAN (30))")
	MustExec(db, "alter table test99.ddltiflash set tiflash replica 1")

	WaitTableOK(db, "ddltiflash")

	MustExec(db, "drop table if exists test99.ddltiflash_victim")
	MustExec(db, "create table test99.ddltiflash_victim(z int)")

	lessThan := "40"
	MustExec(db, "ALTER TABLE test99.ddltiflash ADD PARTITION (PARTITION pn VALUES LESS THAN (%v))", lessThan)

	MustExec(db, "alter table test99.ddltiflash_victim set tiflash replica 1")
	WaitTableOK(db, "ddltiflash")

}

func MustExec(db *sql.DB, f string, args ...interface{}) {
	s := fmt.Sprintf(f, args...)
	_, err := db.Exec(s)
	if err != nil {
		panic(err)
	}
}

func WaitTableOK(db *sql.DB, tbn string) {
	for {
		var x int
		s := fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = '%v';", tbn)
		row := db.QueryRow(s)
		if err := row.Scan(&x); err != nil {
			panic(err)
		}
		if x == 1 {
			return
		}
	}
}

func TestPerformance(C int, T int) {
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
	fAlterAndWait := func(ch *chan []string) {
		for i := 0; i < C; i++ {
			*ch <- []string{
				fmt.Sprintf("alter table test99.t%v set tiflash replica 1", i),
				fmt.Sprintf("SELECT count(*) FROM information_schema.tiflash_replica where progress = 1 and table_schema = 'test99' and TABLE_NAME = 't%v';", i),
			}
		}
	}
	fChecker := func(s *[]string) {
		var x int
		fmt.Printf("Handle %v\n", (*s)[0])
		start := time.Now()
		_, err := db.Exec((*s)[0])
		if err != nil {
			panic(err)
		}
		start2 := time.Now()
		for {
			row := db.QueryRow((*s)[1])
			if err = row.Scan(&x); err != nil {
				panic(err)
			}

			if x == 1 {
				t := time.Since(start)
				t2 := time.Since(start2)
				fmt.Printf("Finish %v Cost(alter+sync) %v Cost(sync) %v\n", (*s)[1], t, t2)
				collect = append(collect, t)
				collect2 = append(collect2, t2)
				return
			}
		}
	}
	AsyncStmtEx(ctx, "replica", T, db, fAlterAndWait, fChecker)
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
