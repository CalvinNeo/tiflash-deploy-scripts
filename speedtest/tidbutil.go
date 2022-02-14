package main

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

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

func GetStatsHelper(tbID int64) string {
	startKey := GenTableRecordPrefix(tbID)
	endKey := EncodeTablePrefix(tbID + 1)
	startKey = EncodeBytes([]byte{}, startKey)
	endKey = EncodeBytes([]byte{}, endKey)

	p := fmt.Sprintf("/pd/api/v1/stats/region?start_key=%s&end_key=%s",
		url.QueryEscape(string(startKey)),
		url.QueryEscape(string(endKey)))
	fmt.Printf("===== AAAAA %v\n", p)
	return p
}

// PDRegionStats is the json response from PD.
type PDRegionStats struct {
	Count            int            `json:"count"`
	EmptyCount       int            `json:"empty_count"`
	StorageSize      int64          `json:"storage_size"`
	StorageKeys      int64          `json:"storage_keys"`
	StoreLeaderCount map[uint64]int `json:"store_leader_count"`
	StorePeerCount   map[uint64]int `json:"store_peer_count"`
}


// GetGroupRules to get all placement rule in a certain group.
func (h *PDHelper) GetPDRegionRecordStats(tableID int64, stats *PDRegionStats) error {
	startKey := GenTableRecordPrefix(tableID)
	endKey := EncodeTablePrefix(tableID + 1)
	startKey = EncodeBytes([]byte{}, startKey)
	endKey = EncodeBytes([]byte{}, endKey)

	getURL := fmt.Sprintf("%s://%s/pd/api/v1/stats/region?start_key=%s&end_key=%s",
		h.InternalHTTPSchema,
		h.PDAddr,
		url.QueryEscape(string(startKey)),
		url.QueryEscape(string(endKey)),
	)
	fmt.Printf("B %v\n", getURL)

	resp, err := h.InternalHTTPClient.Get(getURL)
	if err != nil {
		return errors.New("fail get")
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {

		}
	}()

	if resp.StatusCode != http.StatusOK {
		return errors.New("GetGroupRules returns error")
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return errors.New("fail")
	}

	err = json.Unmarshal(buf.Bytes(), stats)
	fmt.Printf("ffffff %v\n", string(buf.Bytes()))
	if err != nil {
		return errors.New("fail parse")
	}

	return nil
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
