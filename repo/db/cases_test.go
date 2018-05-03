package db

import (
	"bytes"
	"database/sql"
	"gx/ipfs/QmT6n4mspWYEya864BhCUJEgyxiRfmiSY9ruQwTUNpRKaM/protobuf/proto"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/phoreproject/jsonpb"
	"github.com/phoreproject/openbazaar-go/pb"
	"github.com/phoreproject/openbazaar-go/repo"
	"github.com/phoreproject/openbazaar-go/schema"
	"github.com/golang/protobuf/ptypes"
	"sync"
)

var casesdb repo.CaseStore

var buyerTestOutpoints []*pb.Outpoint = []*pb.Outpoint{{"hash1", 0, 5}}
var vendorTestOutpoints []*pb.Outpoint = []*pb.Outpoint{{"hash2", 1, 11}}

func init() {
	conn, _ := sql.Open("sqlite3", ":memory:")
	initDatabaseTables(conn, "")
	casesdb = NewCaseStore(conn, new(sync.Mutex))
	contract = new(pb.RicardianContract)
	listing := new(pb.Listing)
	item := new(pb.Listing_Item)
	item.Title = "Test listing"
	listing.Item = item
	vendorID := new(pb.ID)
	vendorID.PeerID = "vendor id"
	vendorID.Handle = "@testvendor"
	listing.VendorID = vendorID
	image := new(pb.Listing_Item_Image)
	image.Tiny = "test image hash"
	listing.Item.Images = []*pb.Listing_Item_Image{image}
	contract.VendorListings = []*pb.Listing{listing}
	order := new(pb.Order)
	buyerID := new(pb.ID)
	buyerID.PeerID = "buyer id"
	buyerID.Handle = "@testbuyer"
	order.BuyerID = buyerID
	shipping := new(pb.Order_Shipping)
	shipping.Address = "1234 test ave."
	shipping.ShipTo = "buyer name"
	order.Shipping = shipping
	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return
	}
	order.Timestamp = ts
	payment := new(pb.Order_Payment)
	payment.Amount = 10
	payment.Method = pb.Order_Payment_DIRECT
	payment.Address = "3BDbGsH5h5ctDiFtWMmZawcf3E7iWirVms"
	order.Payment = payment
	contract.BuyerOrder = order
}

func TestCasesDB_Count(t *testing.T) {
	err := casesdb.Put("caseID", 5, true, "blah")
	if err != nil {
		t.Error(err)
	}
	i := casesdb.Count()
	if i != 1 {
		t.Error("Returned incorrect number of cases")
	}
}

func TestPutCase(t *testing.T) {
	err := casesdb.Put("caseID", 0, true, "blah")
	if err != nil {
		t.Error(err)
	}
	stmt, err := casesdb.PrepareQuery("select caseID, state, read, buyerOpened, claim from cases where caseID=?")
	defer stmt.Close()

	var caseID string
	var state int
	var read int
	var buyerOpened int
	var claim string
	err = stmt.QueryRow("caseID").Scan(&caseID, &state, &read, &buyerOpened, &claim)
	if err != nil {
		t.Error(err)
	}
	if caseID != "caseID" {
		t.Errorf(`Expected %s got %s`, "caseID", caseID)
	}
	if state != 0 {
		t.Errorf(`Expected 0 got %d`, state)
	}
	if read != 0 {
		t.Errorf(`Expected 0 got %d`, read)
	}
	if buyerOpened != 1 {
		t.Errorf(`Expected 0 got %d`, buyerOpened)
	}
	if claim != strings.ToLower("blah") {
		t.Errorf(`Expected %s got %s`, strings.ToLower("blah"), claim)
	}
}

func TestUpdateWithNil(t *testing.T) {
	err := casesdb.Put("caseID", 0, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateBuyerInfo("caseID", nil, []string{"someError", "anotherError"}, "addr1", nil)
	if err != nil {
		t.Error(err)
	}
	_, _, _, _, buyerOutpoints, _, _, err := casesdb.GetPayoutDetails("caseID")
	if err != nil {
		t.Error(err)
	}
	buyerContract, _, _, _, _, _, _, _, _, _, err := casesdb.GetCaseMetadata("caseID")
	if buyerContract != nil {
		t.Error("Vendor contract was not nil")
	}
	if buyerOutpoints != nil {
		t.Error("Vendor outpoints was not nil")
	}
}

func TestDeleteCase(t *testing.T) {
	err := casesdb.Put("caseID", 0, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.Delete("caseID")
	if err != nil {
		t.Error("Case delete failed")
	}

	stmt, _ := casesdb.PrepareQuery("select caseID from cases where caseID=?")
	defer stmt.Close()

	var caseID string
	err = stmt.QueryRow("caseID").Scan(&caseID)
	if err == nil {
		t.Error("Case delete failed")
	}
}

func TestMarkCaseAsRead(t *testing.T) {
	err := casesdb.Put("caseID", 0, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.MarkAsRead("caseID")
	if err != nil {
		t.Error(err)
	}
	stmt, _ := casesdb.PrepareQuery("select read from cases where caseID=?")
	defer stmt.Close()

	var read int
	err = stmt.QueryRow("caseID").Scan(&read)
	if err != nil {
		t.Error("Case query failed")
	}
	if read != 1 {
		t.Error("Failed to mark case as read")
	}
}

func TestMarkCaseAsUnread(t *testing.T) {
	err := casesdb.Put("caseID", 0, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.MarkAsRead("caseID")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.MarkAsUnread("caseID")
	if err != nil {
		t.Error(err)
	}
	stmt, _ := casesdb.PrepareQuery("select read from cases where caseID=?")
	defer stmt.Close()

	var read int
	err = stmt.QueryRow("caseID").Scan(&read)
	if err != nil {
		t.Error("Case query failed")
	}
	if read != 0 {
		t.Error("Failed to mark case as read")
	}
}

func TestUpdateBuyerInfo(t *testing.T) {
	err := casesdb.Put("caseID", 0, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateBuyerInfo("caseID", contract, []string{"someError", "anotherError"}, "addr1", buyerTestOutpoints)
	if err != nil {
		t.Error(err)
	}

	stmt, err := casesdb.PrepareQuery("select caseID, buyerContract, buyerValidationErrors, buyerPayoutAddress, buyerOutpoints from cases where caseID=?")
	defer stmt.Close()

	var caseID string
	var buyerCon []byte
	var buyerErrors []byte
	var buyerAddr string
	var buyerOuts []byte
	err = stmt.QueryRow("caseID").Scan(&caseID, &buyerCon, &buyerErrors, &buyerAddr, &buyerOuts)
	if err != nil {
		t.Error(err)
	}
	if caseID != "caseID" {
		t.Errorf(`Expected %s got %s`, "caseID", caseID)
	}
	if len(buyerCon) <= 0 {
		t.Error(`Invalid contract returned`)
	}
	if buyerAddr != "addr1" {
		t.Errorf("Expected address %s got %s", "addr1", buyerAddr)
	}
	if string(buyerErrors) != `["someError","anotherError"]` {
		t.Errorf("Expected %s, got %s", `["someError","anotherError"]`, string(buyerErrors))
	}
	if string(buyerOuts) != `[{"hash":"hash1","value":5}]` {
		t.Errorf("Expected %s got %s", `[{"hash":"hash1","value":5}]`, string(buyerOuts))
	}
}

func TestUpdateVendorInfo(t *testing.T) {
	err := casesdb.Put("caseID", 0, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateVendorInfo("caseID", contract, []string{"someError", "anotherError"}, "addr2", vendorTestOutpoints)
	if err != nil {
		t.Error(err)
	}

	stmt, err := casesdb.PrepareQuery("select caseID, vendorContract, vendorValidationErrors, vendorPayoutAddress, vendorOutpoints from cases where caseID=?")
	defer stmt.Close()

	var caseID string
	var vendorCon []byte
	var vendorErrors []byte
	var vendorAddr string
	var vendorOuts []byte
	err = stmt.QueryRow("caseID").Scan(&caseID, &vendorCon, &vendorErrors, &vendorAddr, &vendorOuts)
	if err != nil {
		t.Error(err)
	}
	if caseID != "caseID" {
		t.Errorf(`Expected %s got %s`, "caseID", caseID)
	}
	if len(vendorCon) <= 0 {
		t.Error(`Invalid contract returned`)
	}
	if vendorAddr != "addr2" {
		t.Errorf("Expected address %s got %s", "addr2", vendorAddr)
	}
	if string(vendorErrors) != `["someError","anotherError"]` {
		t.Errorf("Expected %s, got %s", `["someError","anotherError"]`, string(vendorErrors))
	}
	if string(vendorOuts) != `[{"hash":"hash2","index":1,"value":11}]` {
		t.Errorf("Expected %s got %s", `[{"hash":"hash2",index:1,value":11}]`, string(vendorOuts))
	}
}

func TestCasesGetCaseMetaData(t *testing.T) {
	err := casesdb.Put("caseID", pb.OrderState_DISPUTED, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateBuyerInfo("caseID", contract, []string{"someError", "anotherError"}, "addr1", buyerTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateVendorInfo("caseID", contract, []string{"someError", "anotherError"}, "addr1", vendorTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	buyerContract, vendorContract, buyerValidationErrors, vendorValidationErrors, state, read, date, buyerOpened, claim, resolution, err := casesdb.GetCaseMetadata("caseID")
	ser, _ := proto.Marshal(contract)
	buyerSer, _ := proto.Marshal(buyerContract)
	vendorSer, _ := proto.Marshal(vendorContract)

	if !bytes.Equal(ser, buyerSer) || !bytes.Equal(ser, vendorSer) {
		t.Error("Failed to fetch case contract from db")
	}
	if len(buyerValidationErrors) <= 0 || buyerValidationErrors[0] != "someError" || buyerValidationErrors[1] != "anotherError" {
		t.Error("Incorrect buyer validator errors")
	}
	if len(vendorValidationErrors) <= 0 || vendorValidationErrors[0] != "someError" || vendorValidationErrors[1] != "anotherError" {
		t.Error("Incorrect buyer validator errors")
	}
	if state != pb.OrderState_DISPUTED {
		t.Errorf("Expected state %s got %s", pb.OrderState_DISPUTED, state)
	}
	if read != false {
		t.Errorf("Expected read=%s got %s", false, read)
	}
	if date.After(time.Now()) || date.Equal(time.Time{}) {
		t.Error("Case timestamp invalid")
	}
	if !buyerOpened {
		t.Errorf("Expected buyerOpened=%s got %s", true, buyerOpened)
	}
	if claim != "blah" {
		t.Errorf("Expected claim=%s got %s", "blah", claim)
	}
	if resolution != nil {
		t.Error("Resolution should be nil")
	}
	_, _, _, _, _, _, _, _, _, _, err = casesdb.GetCaseMetadata("afasdfafd")
	if err == nil {
		t.Error("Get by unknown caseID failed to return error")
	}
}

func TestGetPayoutDetails(t *testing.T) {
	err := casesdb.Put("caseID", pb.OrderState_DISPUTED, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateBuyerInfo("caseID", contract, []string{"someError", "anotherError"}, "addr1", buyerTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateVendorInfo("caseID", contract, []string{"someError", "anotherError"}, "addr2", vendorTestOutpoints)
	if err != nil {
		t.Error(err)
	}

	buyerContract, vendorContract, buyerAddr, vendorAddr, buyerOutpoints, vendorOutpoints, state, err := casesdb.GetPayoutDetails("caseID")
	if err != nil {
		t.Error(err)
	}
	ser, _ := proto.Marshal(contract)
	buyerSer, _ := proto.Marshal(buyerContract)
	vendorSer, _ := proto.Marshal(vendorContract)

	if !bytes.Equal(ser, buyerSer) || !bytes.Equal(ser, vendorSer) {
		t.Error("Failed to fetch case contract from db")
	}
	if buyerAddr != "addr1" {
		t.Errorf("Expected address %s got %s", "addr1", buyerAddr)
	}
	if vendorAddr != "addr2" {
		t.Errorf("Expected address %s got %s", "addr2", vendorAddr)
	}
	if len(buyerOutpoints) != len(buyerTestOutpoints) {
		t.Error("Incorrect number of buyer outpoints returned")
	}
	for i, o := range buyerTestOutpoints {
		if o.Hash != buyerTestOutpoints[i].Hash {
			t.Errorf("Expected outpoint hash %s got %s", o.Hash, buyerTestOutpoints[i].Hash)
		}
		if o.Index != buyerTestOutpoints[i].Index {
			t.Errorf("Expected outpoint index %s got %s", o.Index, buyerTestOutpoints[i].Index)
		}
		if o.Value != buyerTestOutpoints[i].Value {
			t.Errorf("Expected outpoint value %s got %s", o.Value, buyerTestOutpoints[i].Value)
		}
	}
	if len(vendorOutpoints) != len(vendorTestOutpoints) {
		t.Error("Incorrect number of buyer outpoints returned")
	}
	for i, o := range vendorTestOutpoints {
		if o.Hash != vendorTestOutpoints[i].Hash {
			t.Errorf("Expected outpoint hash %s got %s", o.Hash, vendorTestOutpoints[i].Hash)
		}
		if o.Index != vendorTestOutpoints[i].Index {
			t.Errorf("Expected outpoint index %s got %s", o.Index, vendorTestOutpoints[i].Index)
		}
		if o.Value != vendorTestOutpoints[i].Value {
			t.Errorf("Expected outpoint value %s got %s", o.Value, vendorTestOutpoints[i].Value)
		}
	}
	if state != pb.OrderState_DISPUTED {
		t.Errorf("Expected state %s got %s", pb.OrderState_DISPUTED, state)
	}
}

func TestMarkAsClosed(t *testing.T) {
	err := casesdb.Put("caseID", pb.OrderState_DISPUTED, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateBuyerInfo("caseID", contract, []string{"someError", "anotherError"}, "addr1", buyerTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateVendorInfo("caseID", contract, []string{"someError", "anotherError"}, "addr2", vendorTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	d := new(pb.DisputeResolution)
	d.Resolution = "Case closed"
	err = casesdb.MarkAsClosed("caseID", d)
	if err != nil {
		t.Error(err)
	}
	_, _, _, _, state, _, _, _, _, resolution, err := casesdb.GetCaseMetadata("caseID")
	if err != nil {
		t.Error(err)
	}
	if state != pb.OrderState_RESOLVED {
		t.Error("Mark as closed failed to set state to resolved")
	}
	if resolution.Resolution != d.Resolution {
		t.Error("Failed to save correct dispute resolution")
	}
}

func TestCasesDB_GetAll(t *testing.T) {
	err := casesdb.Put("caseID", 10, true, "blah")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateBuyerInfo("caseID", contract, []string{"someError", "anotherError"}, "addr1", buyerTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateVendorInfo("caseID", contract, []string{"someError", "anotherError"}, "addr2", vendorTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Second)
	err = casesdb.Put("caseID2", 11, true, "asdf")
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateBuyerInfo("caseID2", contract, []string{"someError", "anotherError"}, "addr1", buyerTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	err = casesdb.UpdateVendorInfo("caseID2", contract, []string{"someError", "anotherError"}, "addr2", vendorTestOutpoints)
	if err != nil {
		t.Error(err)
	}
	cases, ct, err := casesdb.GetAll([]pb.OrderState{}, "", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(cases) != 2 {
		t.Error("Returned incorrect number of cases")
	}
	if ct != 2 {
		t.Error("Returned incorrect number of query cases")
	}
	cases, ct, err = casesdb.GetAll([]pb.OrderState{}, "", false, false, 1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(cases) != 1 {
		t.Error("Returned incorrect number of cases")
	}
	if ct != 2 {
		t.Error("Returned incorrect number of query cases")
	}
	cases, ct, err = casesdb.GetAll([]pb.OrderState{}, "", true, false, -1, []string{"caseID"})
	if err != nil {
		t.Error(err)
	}
	if len(cases) != 1 {
		t.Error("Returned incorrect number of cases")
	}
	if ct != 2 {
		t.Error("Returned incorrect number of query cases")
	}
	cases, ct, err = casesdb.GetAll([]pb.OrderState{pb.OrderState_DISPUTED}, "", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(cases) != 1 {
		t.Error("Returned incorrect number of cases")
	}
	if ct != 1 {
		t.Error("Returned incorrect number of query cases")
	}
	cases, ct, err = casesdb.GetAll([]pb.OrderState{pb.OrderState_DECIDED}, "", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(cases) != 1 {
		t.Error("Returned incorrect number of cases")
	}
	if ct != 1 {
		t.Error("Returned incorrect number of query cases")
	}
	cases, ct, err = casesdb.GetAll([]pb.OrderState{pb.OrderState_DISPUTED, pb.OrderState_DECIDED}, "", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(cases) != 2 {
		t.Error("Returned incorrect number of cases")
	}
	if ct != 2 {
		t.Error("Returned incorrect number of query cases")
	}
	cases, ct, err = casesdb.GetAll([]pb.OrderState{}, "caseid2", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(cases) != 1 {
		t.Error("Returned incorrect number of cases")
	}
	if ct != 1 {
		t.Error("Returned incorrect number of query cases")
	}
}

func TestGetDisputesForDisputeExpiryReturnsRelevantRecords(t *testing.T) {
	database, _ := sql.Open("sqlite3", ":memory:")
	setupSQL := []string{
		schema.PragmaKey(""),
		schema.CreateTableDisputedCasesSQL,
	}
	_, err := database.Exec(strings.Join(setupSQL, " "))
	if err != nil {
		t.Fatal(err)
	}
	// Artificially start disputes 50 days ago
	var (
		now        = time.Unix(time.Now().Unix(), 0)
		timeStart  = now.Add(time.Duration(-50*24) * time.Hour)
		nowData, _ = ptypes.TimestampProto(now)
		order      = &pb.Order{
			BuyerID: &pb.ID{
				PeerID: "buyerID",
				Handle: "@buyerID",
			},
			Shipping: &pb.Order_Shipping{
				Address: "1234 Test Ave",
				ShipTo:  "Buyer Name",
			},
			Payment: &pb.Order_Payment{
				Amount:  10,
				Method:  pb.Order_Payment_DIRECT,
				Address: "3BDbGsH5h5ctDiFtWMmZawcf3E7iWirVms",
			},
			Timestamp: nowData,
		}
		expectedImagesOne = []*pb.Listing_Item_Image{{Tiny: "tinyimagehashOne", Small: "smallimagehashOne"}}
		contract          = &pb.RicardianContract{
			VendorListings: []*pb.Listing{
				{Item: &pb.Listing_Item{Images: expectedImagesOne}},
			},
			BuyerOrder: order,
		}
		neverNotified = &repo.DisputeCaseRecord{
			CaseID:           "neverNotified",
			Timestamp:        timeStart,
			LastNotifiedAt:   time.Unix(0, 0),
			BuyerContract:    contract,
			VendorContract:   contract,
			IsBuyerInitiated: true,
		}
		initialNotified = &repo.DisputeCaseRecord{
			CaseID:           "initialNotificationSent",
			Timestamp:        timeStart,
			LastNotifiedAt:   timeStart,
			BuyerContract:    contract,
			VendorContract:   contract,
			IsBuyerInitiated: true,
		}
		finallyNotified = &repo.DisputeCaseRecord{
			CaseID:           "finalNotificationSent",
			Timestamp:        timeStart,
			LastNotifiedAt:   time.Now(),
			BuyerContract:    contract,
			VendorContract:   contract,
			IsBuyerInitiated: true,
		}
		existingRecords = []*repo.DisputeCaseRecord{
			neverNotified,
			initialNotified,
			finallyNotified,
		}
	)

	m := jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: true,
		Indent:       "    ",
		OrigName:     false,
	}
	for _, r := range existingRecords {
		var isBuyerInitiated int = 0
		if r.IsBuyerInitiated {
			isBuyerInitiated = 1
		}
		buyerContract, err := m.MarshalToString(r.BuyerContract)
		if err != nil {
			t.Fatal(err)
		}
		vendorContract, err := m.MarshalToString(r.VendorContract)
		if err != nil {
			t.Fatal(err)
		}
		_, err = database.Exec("insert into cases (caseID, buyerContract, vendorContract, timestamp, buyerOpened, lastNotifiedAt) values (?, ?, ?, ?, ?, ?);", r.CaseID, buyerContract, vendorContract, int(r.Timestamp.Unix()), isBuyerInitiated, int(r.LastNotifiedAt.Unix()))
		if err != nil {
			t.Fatal(err)
		}
	}

	casesdb := NewCaseStore(database, new(sync.Mutex))
	cases, err := casesdb.GetDisputesForDisputeExpiryNotification()
	if err != nil {
		t.Fatal(err)
	}

	var sawNeverNotifiedCase, sawInitialNotifiedCase, sawFinallyNotifiedCase bool
	for _, c := range cases {
		switch c.CaseID {
		case neverNotified.CaseID:
			sawNeverNotifiedCase = true
			if reflect.DeepEqual(c, neverNotified) != true {
				t.Error("Expected neverNotified to match, but did not")
				t.Error("Expected:", neverNotified)
				t.Error("Actual:", c)
			}
		case initialNotified.CaseID:
			sawInitialNotifiedCase = true
			if reflect.DeepEqual(c, initialNotified) != true {
				t.Error("Expected initialNotified to match, but did not")
				t.Error("Expected:", initialNotified)
				t.Error("Actual:", c)
			}
		case finallyNotified.CaseID:
			sawFinallyNotifiedCase = true
		default:
			t.Error("Found unexpected dispute case: %+v", c)
		}
	}

	if sawNeverNotifiedCase == false {
		t.Error("Expected to see case which was never notified")
	}
	if sawInitialNotifiedCase == false {
		t.Error("Expected to see case which was initially notified")
	}
	if sawFinallyNotifiedCase == true {
		t.Error("Expected NOT to see case which recieved it's final notification")
	}
}

func TestUpdateDisputeLastNotifiedAt(t *testing.T) {
	database, _ := sql.Open("sqlite3", ":memory:")
	setupSQL := []string{
		schema.PragmaKey(""),
		schema.CreateTableDisputedCasesSQL,
	}
	_, err := database.Exec(strings.Join(setupSQL, " "))
	if err != nil {
		t.Fatal(err)
	}
	// Artificially start disputes 50 days ago
	timeStart := time.Now().Add(time.Duration(-50*24) * time.Hour)
	disputeOne := &repo.DisputeCaseRecord{
		CaseID:         "case1",
		Timestamp:      timeStart,
		LastNotifiedAt: time.Unix(123, 0),
	}
	disputeTwo := &repo.DisputeCaseRecord{
		CaseID:         "case2",
		Timestamp:      timeStart,
		LastNotifiedAt: time.Unix(456, 0),
	}
	_, err = database.Exec("insert into cases (caseID, timestamp, lastNotifiedAt) values (?, ?, ?);", disputeOne.CaseID, disputeOne.Timestamp, disputeOne.LastNotifiedAt)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec("insert into cases (caseID, timestamp, lastNotifiedAt) values (?, ?, ?);", disputeTwo.CaseID, int(disputeTwo.Timestamp.Unix()), int(disputeTwo.LastNotifiedAt.Unix()))
	if err != nil {
		t.Fatal(err)
	}

	disputeOne.LastNotifiedAt = time.Unix(987, 0)
	disputeTwo.LastNotifiedAt = time.Unix(765, 0)
	casesdb := NewCaseStore(database, new(sync.Mutex))
	err = casesdb.UpdateDisputesLastNotifiedAt([]*repo.DisputeCaseRecord{disputeOne, disputeTwo})
	if err != nil {
		t.Fatal(err)
	}

	rows, err := database.Query("select caseID, lastNotifiedAt from cases")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var (
			caseID         string
			lastNotifiedAt int64
		)
		if err = rows.Scan(&caseID, &lastNotifiedAt); err != nil {
			t.Fatal(err)
		}
		switch caseID {
		case disputeOne.CaseID:
			if time.Unix(lastNotifiedAt, 0).Equal(disputeOne.LastNotifiedAt) != true {
				t.Error("Expected disputeOne.LastNotifiedAt to be updated")
			}
		case disputeTwo.CaseID:
			if time.Unix(lastNotifiedAt, 0).Equal(disputeTwo.LastNotifiedAt) != true {
				t.Error("Expected disputeTwo.LastNotifiedAt to be updated")
			}
		default:
			t.Error("Unexpected dispute case encounted")
			t.Error(caseID, lastNotifiedAt)
		}

	}
}
