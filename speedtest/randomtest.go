package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Table struct {
	ReplicaCount int
	Dropped bool
	RuleCount int
	// Rule count for each table partition, will increase when truncated.
	PartitionRuleCount *map[int]int
	// Is true when the table partition is drooped.
	PartitionDropped *map[int]bool
}

type Tables struct {
	sync.Mutex

	Ts map[int]*Table
	Replica int
}


func RandomWrite(db *sql.DB, table string, n int, start int) {
	for i := start; i < start + n; i++{
		MustExec(db, "insert into test99.%v values (%v)", table, i)
	}
}

func (t *Tables) SetTiFlashReplica(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if !v.Dropped && v.ReplicaCount == 0 {
			v.ReplicaCount = *ReplicaNum
			if v.PartitionRuleCount != nil {
				for pk, _ := range *(v.PartitionRuleCount) {
					if !(*(v.PartitionDropped))[pk] {
						(*(v.PartitionRuleCount))[pk] += 1
					}
				}
			} else {
				v.RuleCount += 1
			}
			s := fmt.Sprintf("alter table test98.t%v set tiflash replica %v", k, t.Replica)
			MustExec(db, s)
			t.PrintGather(pd)
			return s
		}
	}
	return ""
}

func (t *Tables) RemoveTiFlashReplica(pd *PDHelper, db *sql.DB) string {
	t.Lock()
	defer t.Unlock()

	for k, v := range t.Ts {
		if !v.Dropped && v.ReplicaCount != 0 {
			t.Ts[k].ReplicaCount = 0
			if v.PartitionRuleCount != nil {
				for pk, _ := range *(v.PartitionRuleCount) {
					if !(*(v.PartitionDropped))[pk] {
						(*(v.PartitionRuleCount))[pk] -= 1
					}
				}
			} else {
				v.RuleCount -= 1
			}
			s := fmt.Sprintf("alter table test98.t%v set tiflash replica 0", k)
			MustExec(db, s)
			t.PrintGather(pd)
			return s
		}
	}
	return ""
}

func addPartitionHelper(tb *Table) int {
	nn := len(*(tb.PartitionRuleCount))
	(*(tb.PartitionRuleCount))[nn] = 1
	(*(tb.PartitionDropped))[nn] = false
	return nn
}

func addGivenPartitionHelper(tb *Table, i int) {
	(*(tb.PartitionRuleCount))[i] = 1
	(*(tb.PartitionDropped))[i] = false
}

func (t *Tables) AddTable(pd *PDHelper, db *sql.DB, partition bool, setReplica bool) []string {
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
			RuleCount:          0,
			PartitionRuleCount: &m,
			PartitionDropped:   &m2,
			ReplicaCount:		0,
		}
		ss := []string{fmt.Sprintf("create table test98.t%v (z int) partition by range (z) (partition p0 values less than (0))", n)}
		if setReplica {
			t.Ts[n].ReplicaCount = *ReplicaNum
			addGivenPartitionHelper(t.Ts[n], 0)
			ss = append(ss, fmt.Sprintf("alter table test98.t%v set tiflash replica %v", n, t.Replica))
		}
		for _, e := range ss {
			MustExec(db, e)
		}
		t.PrintGather(pd)
		return ss
	} else {
		t.Ts[n] = &Table{
			Dropped:            false,
			RuleCount:          0,
			PartitionRuleCount: nil,
			PartitionDropped:   nil,
			ReplicaCount:		0,
		}
		ss := []string{fmt.Sprintf("create table test98.t%v (z int)", n)}
		if setReplica {
			t.Ts[n].ReplicaCount = *ReplicaNum
			t.Ts[n].RuleCount = 1
			ss = append(ss, fmt.Sprintf("alter table test98.t%v set tiflash replica %v", n, t.Replica))
		}
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
		if !v.Dropped && v.ReplicaCount != 0 {
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
		if v.PartitionRuleCount != nil && !v.Dropped && v.ReplicaCount != 0 {
			n := addPartitionHelper(v)
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
		if v.PartitionRuleCount != nil && !v.Dropped && v.ReplicaCount != 0 {
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
		if v.PartitionRuleCount != nil && !v.Dropped && v.ReplicaCount != 0 {
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
		if !v.Dropped && v.ReplicaCount != 0 {
			t.Ts[k].Dropped = true
			s := fmt.Sprintf("drop table test98.t%v", k)
			MustExec(db, s)
			t.PrintGather(pd)
			return s
		}
	}
	return ""
}

func (t *Tables) AlterDatabaseSetReplica(pd *PDHelper, db *sql.DB, DBName string) string {
	t.Lock()
	defer t.Unlock()

	for _, v := range t.Ts {
		if !v.Dropped {
			if v.ReplicaCount == 0 {
				if v.PartitionRuleCount != nil {
					for pk, _ := range *(v.PartitionRuleCount) {
						if !(*(v.PartitionDropped))[pk] {
							(*(v.PartitionRuleCount))[pk] += 1
						}
					}
				} else {
					v.RuleCount += 1
				}
			}
			v.ReplicaCount = *ReplicaNum
		}
	}

	s := fmt.Sprintf("alter database %v set tiflash replica %v", DBName, *ReplicaNum)
	MustExec(db, s)
	t.PrintGather(pd)
	return ""
}

func (t *Tables) RemoveDatabaseSetReplica(pd *PDHelper, db *sql.DB, DBName string) string {
	t.Lock()
	defer t.Unlock()

	for _, v := range t.Ts {
		if !v.Dropped {
			if v.ReplicaCount != 0 {
				if v.PartitionRuleCount != nil {
					// If has partition
					for pk, _ := range *(v.PartitionRuleCount) {
						if !(*(v.PartitionDropped))[pk] {
							// Maybe we have already truncated
							(*(v.PartitionRuleCount))[pk] -= 1
						}
					}
				} else {
					// Maybe we have already truncated
					v.RuleCount -= 1
				}
			}
			v.ReplicaCount = 0
		}
	}

	s := fmt.Sprintf("alter database %v set tiflash replica %v", DBName, 0)
	MustExec(db, s)
	t.PrintGather(pd)
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

func (t *Tables) NoReplicaTableCount() int {
	t.Lock()
	defer t.Unlock()

	noReplica := 0
	for i, tb := range t.Ts {
		if !(tb.Dropped) && (tb.ReplicaCount == 0){
			fmt.Printf("Table %v has 0 Replica\n", i)
			noReplica += 1
		}
	}
	return noReplica
}

func TestMultiSession(T int, Replica int, WithAlterDB bool, C int) {
	// Need configure-store-limit
	// https://docs.pingcap.com/zh/tidb/stable/configure-store-limit/
	dbm := GetSession()
	MustExec(dbm, "drop database if exists test98")
	MustExec(dbm, "create database test98")
	MustExec(dbm, "drop database if exists test97")
	MustExec(dbm, "create database test97")
	ChangeGCSafePoint(dbm, time.Now().Add(0 - 24 * time.Hour), "false", "1000m")
	defer dbm.Close()
	time.Sleep(2 * time.Second)

	pd := NewPDHelper(*PDAddr)
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
			db := GetSession()
			MustExec(db, "set SESSION tidb_batch_pending_tiflash_count=5")
			defer db.Close()

			tables.AddTable(pd, db, false, true) // 1
			tables.AddTable(pd, db,true, true) // 1
			tables.AddPartition(pd, db) // 1

			for i := 0; i < C; i ++ {
				//in := []int{0,1,2,3,4,5,6,7,8,9,10,11}
				in := []int{0,1,2,3,4,5,6,7,8,9}
				if WithAlterDB {
					in = []int{-99,-1,-2,0,1,2,3,4,5,6,7,8,9,10,11}
				}
				randomIndex := rand.Intn(len(in))
				pick := in[randomIndex]
				if pick == 0 {
					tables.AddTable(pd, db, false, true)
				} else if pick == 1 {
					tables.AddTable(pd, db, true, true)
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
				} else if pick == -1 {
					tables.AlterDatabaseSetReplica(pd, db, "test98")
				} else if pick == -2 {
					tables.RemoveDatabaseSetReplica(pd, db, "test98")
				} else if pick == 8 {
					//tables.AddTable(pd, db, false, false)
				} else if pick == 9 {
					//tables.AddTable(pd, db, false, false)
				} else if pick == 10 {
					tables.SetTiFlashReplica(pd, db)
				} else if pick == 11 {
					tables.RemoveTiFlashReplica(pd, db)
				} else if pick == -99 {
					noReplica := tables.NoReplicaTableCount()
					if ok := WaitAllTableOK(dbm, "test98", 400, "all", noReplica); !ok {
						panic("Some table not ready")
					}
				}
			}

			wg.Done()
		}(y)
	}
	wg.Wait()

	noReplica := tables.NoReplicaTableCount()
	if ok := WaitAllTableOK(dbm, "test98", 40, "all", noReplica); !ok {
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

