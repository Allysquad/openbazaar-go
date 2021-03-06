package db

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/phoreproject/btcd/chaincfg"
	"github.com/phoreproject/btcutil"
	"github.com/phoreproject/openbazaar-go/pb"
	"github.com/phoreproject/wallet-interface"
)

var purdb PurchasesDB
var contract *pb.RicardianContract

func init() {
	conn, _ := sql.Open("sqlite3", ":memory:")
	initDatabaseTables(conn, "")
	purdb = PurchasesDB{
		db: conn,
	}
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
	payment.Address = "PK5fSKzv5nGqzFT1mbEK21U8wf2Sj8QqQd"
	order.Payment = payment
	contract.BuyerOrder = order
}

func TestPurchasesDB_Count(t *testing.T) {
	err := purdb.Put("orderID", *contract, 0, false)
	if err != nil {
		t.Error(err)
	}
	i := purdb.Count()
	if i != 1 {
		t.Error("Returned incorrect number of purchases")
	}
}

func TestPutPurchase(t *testing.T) {
	err := purdb.Put("orderID", *contract, 0, false)
	if err != nil {
		t.Error(err)
	}
	stmt, _ := purdb.db.Prepare("select orderID, contract, state, read, timestamp, total, thumbnail, vendorID, vendorHandle, title, shippingName, shippingAddress from purchases where orderID=?")
	defer stmt.Close()

	var orderID string
	var c []byte
	var state int
	var read int
	var date int
	var total int
	var thumbnail string
	var vendorID string
	var vendorHandle string
	var title string
	var shippingName string
	var shippingAddress string
	err = stmt.QueryRow("orderID").Scan(&orderID, &c, &state, &read, &date, &total, &thumbnail, &vendorID, &vendorHandle, &title, &shippingName, &shippingAddress)
	if err != nil {
		t.Error(err)
	}
	if orderID != "orderID" {
		t.Errorf(`Expected %s got %s`, "orderID", orderID)
	}
	if state != 0 {
		t.Errorf(`Expected 0 got %d`, state)
	}
	if read != 0 {
		t.Errorf(`Expected 0 got %d`, read)
	}
	if date != int(contract.BuyerOrder.Timestamp.Seconds) {
		t.Errorf("Expected %d got %d", int(contract.BuyerOrder.Timestamp.Seconds), date)
	}
	if total != int(contract.BuyerOrder.Payment.Amount) {
		t.Errorf("Expected %d got %d", int(contract.BuyerOrder.Payment.Amount), total)
	}
	if thumbnail != contract.VendorListings[0].Item.Images[0].Tiny {
		t.Errorf("Expected %s got %s", contract.VendorListings[0].Item.Images[0].Tiny, thumbnail)
	}
	if vendorID != contract.VendorListings[0].VendorID.PeerID {
		t.Errorf(`Expected %s got %s`, contract.VendorListings[0].VendorID.PeerID, vendorID)
	}
	if vendorHandle != contract.VendorListings[0].VendorID.Handle {
		t.Errorf(`Expected %s got %s`, contract.VendorListings[0].VendorID.Handle, vendorHandle)
	}
	if title != contract.VendorListings[0].Item.Title {
		t.Errorf(`Expected %s got %s`, contract.VendorListings[0].Item.Title, title)
	}
	if shippingName != strings.ToLower(contract.BuyerOrder.Shipping.ShipTo) {
		t.Errorf(`Expected %s got %s`, strings.ToLower(contract.BuyerOrder.Shipping.ShipTo), shippingName)
	}
	if shippingAddress != strings.ToLower(contract.BuyerOrder.Shipping.Address) {
		t.Errorf(`Expected %s got %s`, strings.ToLower(contract.BuyerOrder.Shipping.Address), shippingAddress)
	}
}

func TestDeletePurchase(t *testing.T) {
	purdb.Put("orderID", *contract, 0, false)
	err := purdb.Delete("orderID")
	if err != nil {
		t.Error("Purchase delete failed")
	}

	stmt, _ := purdb.db.Prepare("select orderID, contract, state, read from purchases where orderID=?")
	defer stmt.Close()

	var orderID string
	var contract []byte
	var state int
	var read int
	err = stmt.QueryRow("orderID").Scan(&orderID, &contract, &state, &read)
	if err == nil {
		t.Error("Purchase delete failed")
	}
}

func TestMarkPurchaseAsRead(t *testing.T) {
	purdb.Put("orderID", *contract, 0, false)
	err := purdb.MarkAsRead("orderID")
	if err != nil {
		t.Error(err)
	}
	stmt, _ := purdb.db.Prepare("select read from purchases where orderID=?")
	defer stmt.Close()

	var read int
	err = stmt.QueryRow("orderID").Scan(&read)
	if err != nil {
		t.Error("Purchase query failed")
	}
	if read != 1 {
		t.Error("Failed to mark purchase as read")
	}
}

func TestMarkPurchaseAsUnread(t *testing.T) {
	purdb.Put("orderID", *contract, 0, false)
	err := purdb.MarkAsRead("orderID")
	if err != nil {
		t.Error(err)
	}

	err = purdb.MarkAsUnread("orderID")
	if err != nil {
		t.Error(err)
	}
	stmt, _ := purdb.db.Prepare("select read from purchases where orderID=?")
	defer stmt.Close()

	var read int
	err = stmt.QueryRow("orderID").Scan(&read)
	if err != nil {
		t.Error("Purchase query failed")
	}
	if read != 0 {
		t.Error("Failed to mark purchase as read")
	}
}

func TestUpdatePurchaseFunding(t *testing.T) {
	err := purdb.Put("orderID", *contract, 1, false)
	if err != nil {
		t.Error(err)
	}
	record := &wallet.TransactionRecord{
		Txid: "abc123",
	}
	records := []*wallet.TransactionRecord{record}
	err = purdb.UpdateFunding("orderID", true, records)
	if err != nil {
		t.Error(err)
	}
	addr, err := btcutil.DecodeAddress(contract.BuyerOrder.Payment.Address, &chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	_, _, funded, rcds, err := purdb.GetByPaymentAddress(addr)
	if err != nil {
		t.Error(err)
		return
	}
	if !funded {
		t.Error("Update funding failed to update the funded bool")
		return
	}
	if len(rcds) == 0 {
		t.Error("Failed to return transaction records")
		return
	}
	if rcds[0].Txid != "abc123" {
		t.Error("Failed to return correct txid on record")
	}
}

func TestPurchasePutAfterFundingUpdate(t *testing.T) {
	err := purdb.Put("orderID", *contract, 1, false)
	if err != nil {
		t.Error(err)
	}
	record := &wallet.TransactionRecord{
		Txid: "abc123",
	}
	records := []*wallet.TransactionRecord{record}
	err = purdb.UpdateFunding("orderID", true, records)
	if err != nil {
		t.Error(err)
	}
	err = purdb.Put("orderID", *contract, 3, false)
	if err != nil {
		t.Error(err)
	}
	addr, err := btcutil.DecodeAddress(contract.BuyerOrder.Payment.Address, &chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	_, _, funded, rcds, err := purdb.GetByPaymentAddress(addr)
	if err != nil {
		t.Error(err)
		return
	}
	if !funded {
		t.Error("Update funding failed to update the funded bool")
		return
	}
	if len(rcds) == 0 {
		t.Error("Failed to return transaction records")
		return
	}
	if rcds[0].Txid != "abc123" {
		t.Error("Failed to return correct txid on record")
	}
}

func TestPurchasesGetByPaymentAddress(t *testing.T) {
	purdb.Put("orderID", *contract, 0, false)
	addr, err := btcutil.DecodeAddress(contract.BuyerOrder.Payment.Address, &chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	_, _, _, _, err = purdb.GetByPaymentAddress(addr)
	if err != nil {
		t.Error(err)
	}
	addr, err = btcutil.DecodeAddress("PUxo8xZwGYYasHGmkdQo3YnE7ZTyZuwwzK", &chaincfg.MainNetParams)
	if err != nil {
		t.Error(err)
	}
	_, _, _, _, err = purdb.GetByPaymentAddress(addr)
	if err == nil {
		t.Error("Get by unknown address failed to return error")
	}

}

func TestPurchasesGetByOrderId(t *testing.T) {
	purdb.Put("orderID", *contract, 0, false)
	_, _, _, _, _, err := purdb.GetByOrderId("orderID")
	if err != nil {
		t.Error(err)
	}
	_, _, _, _, _, err = purdb.GetByOrderId("fasdfas")
	if err == nil {
		t.Error("Get by unknown orderId failed to return error")
	}
}

func TestPurchasesDB_GetAll(t *testing.T) {
	c0 := *contract
	ts, _ := ptypes.TimestampProto(time.Now())
	c0.BuyerOrder.Timestamp = ts
	purdb.Put("orderID", c0, 0, false)
	c1 := *contract
	ts, _ = ptypes.TimestampProto(time.Now().Add(time.Minute))
	c1.BuyerOrder.Timestamp = ts
	purdb.Put("orderID2", c1, 1, false)
	c2 := *contract
	ts, _ = ptypes.TimestampProto(time.Now().Add(time.Hour))
	c2.BuyerOrder.Timestamp = ts
	purdb.Put("orderID3", c2, 1, false)
	// Test no offset no limit
	purchases, ct, err := purdb.GetAll([]pb.OrderState{}, "", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(purchases) != 3 {
		t.Error("Returned incorrect number of purchases")
	}
	if ct != 3 {
		t.Error("Returned incorrect number of query purchases")
	}

	// Test no offset limit 1
	purchases, ct, err = purdb.GetAll([]pb.OrderState{}, "", false, false, 1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(purchases) != 1 {
		t.Error("Returned incorrect number of purchases")
	}
	if ct != 3 {
		t.Error("Returned incorrect number of query purchases")
	}

	// Test offset no limit
	purchases, ct, err = purdb.GetAll([]pb.OrderState{}, "", false, false, -1, []string{"orderID"})
	if err != nil {
		t.Error(err)
	}
	if len(purchases) != 2 {
		t.Error("Returned incorrect number of purchases")
	}
	if ct != 3 {
		t.Error("Returned incorrect number of query purchases")
	}

	// Test no offset no limit with state filter
	purchases, ct, err = purdb.GetAll([]pb.OrderState{pb.OrderState_AWAITING_PAYMENT}, "", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(purchases) != 2 {
		t.Error("Returned incorrect number of purchases")
	}
	if ct != 2 {
		t.Error("Returned incorrect number of query purchases")
	}

	// Test offset no limit with state filter
	purchases, ct, err = purdb.GetAll([]pb.OrderState{pb.OrderState_AWAITING_PAYMENT}, "", false, false, -1, []string{"orderID3"})
	if err != nil {
		t.Error(err)
	}
	if len(purchases) != 1 {
		t.Error("Returned incorrect number of purchases")
	}
	if ct != 2 {
		t.Error("Returned incorrect number of query purchases")
	}

	// Test no offset no limit with multiple state filters
	purchases, ct, err = purdb.GetAll([]pb.OrderState{pb.OrderState_AWAITING_PAYMENT, pb.OrderState_PENDING}, "", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(purchases) != 3 {
		t.Error("Returned incorrect number of purchases")
	}
	if ct != 3 {
		t.Error("Returned incorrect number of query purchases")
	}

	// Test no offset no limit with search term
	purchases, ct, err = purdb.GetAll([]pb.OrderState{}, "orderid2", false, false, -1, []string{})
	if err != nil {
		t.Error(err)
	}
	if len(purchases) != 1 {
		t.Error("Returned incorrect number of purchases")
	}
	if ct != 1 {
		t.Error("Returned incorrect number of query purchases")
	}
}
