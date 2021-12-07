package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"sync"
	"time"
)

func main() {
	db, err := sql.Open("mysql", "root@tcp(127.0.0.1:4000)/")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_,err = db.Exec("DROP DATABASE IF EXISTS test99")
	if err != nil {
		panic(err)
	}

	_,err = db.Exec("CREATE DATABASE test99")
	if err != nil {
		panic(err)
	}

	C := 100

	ctx := context.Background()
	threads := 10

	ch := make(chan string, threads+10)
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
						fmt.Printf("Handle %v\n", s)
						_,err = db.Exec(s)
						if err != nil {
							panic(err)
						}
					} else {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
	for i := 0; i < C; i++ {
		ch <- fmt.Sprintf("create table test99.t%v(z int)", i)
	}
	close(ch)
	wg.Wait()


	start := time.Now()
	ch = make(chan string, threads+10)
	var wg2 sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg2.Add(1)
		go func(index int) {
			defer func() {
				wg2.Done()
			}()
			for {
				select {
				case s, ok := <-ch:
					if ok {
						fmt.Printf("Handle %v\n", s)
						_,err = db.Exec(s)
						if err != nil {
							panic(err)
						}
					} else {
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
	for i := 0; i < C; i++ {
		ch <- fmt.Sprintf("alter table test99.t%v set tiflash replica 1", i)
	}
	close(ch)
	wg2.Wait()

	for {
		var x int
		row := db.QueryRow(`select count(*) from INFORMATION_SCHEMA.tables where TABLE_SCHEMA="test99"`)
		if err != nil {
			panic(err)
		}
		if err = row.Scan(&x); err != nil {
			panic(err)
		}
		if x == C {
			fmt.Printf("OK %v \v", time.Since(start))
			break
		}
		fmt.Printf("---> %v\n", x)
	}

}
