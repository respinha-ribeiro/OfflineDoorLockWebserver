package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"./rfid_db"
	_ "github.com/mattn/go-sqlite3"
)

type UserKeys struct {
	Admin bool
	Date  string
	Key   string
}

type UserReq struct {
	Admin     bool
	Date      string
	User      string
	Duration  int
	Lockalias string
}

type User struct {
	User string
}

type LockKeyDates struct {
	Admin bool
	Lock  string
	Key   string
	Dates string
}

type UserLock struct {
	User string
	Lock string
}

var conn *sql.DB

func main() {

	conn = rfid_db.InitConn()
	defer conn.Close()

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/login", Login)
	http.HandleFunc("/req_keys", RequestKeys)
	http.HandleFunc("/update_master_key", UpdateMasterkey)

	/*http.HandleFunc("/register_user", RegisterUser)*/

	//http.HandleFunc("/submit_logs", SubmitLogs)

	fmt.Println("Listening on port 8000...")
	// err := http.ListenAndServeTLS(":8000", "server.crt", "server.key", nil)

	/*rfid_db.ComputeKeys(conn, "2017-May-31", "John Doe", "lock 1", 5)
	rfid_db.ComputeKeys(conn, "2017-Aug-04", "John Doe", "lock 1", 7)
	fmt.Println(rfid_db.GetUpdatedKeys(conn, "John Doe"))*/

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		//log.Fatal("ListenAndServe: ", err)
		fmt.Println("ERROR")
		fmt.Println(err)
	}
}

func hello(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello world!\n")
}

func basicAuth(w http.ResponseWriter, r *http.Request) bool {

	if r == nil {
		return false
	}

	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	fmt.Println("Split")

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	fmt.Println("Decoded")

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}

	/*if rfid_db.GetClientID(conn, pair[0]) == -1 {
		return false
	}*/

	fmt.Println("requested auth!\n")
	// pass := rfid_db.GetClientPassword(conn, pair[0])
	return rfid_db.MatchPassword(conn, pair[0], pair[1])

}

func RequestKeys(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	decoder := json.NewDecoder(r.Body)

	req := UserReq{}
	err := decoder.Decode(&req)
	if err != nil {
		panic(err)
	}

	start := req.Date
	username := req.User
	duration := req.Duration
	lockalias := req.Lockalias

	// staticKey := rfid_db.GetStaticKey(conn, username, lockalias)

	fmt.Println("Requested for ", start, "with", duration)

	dates, keys, userHash := rfid_db.ComputeKeys(conn, start, username, lockalias, duration)

	if dates == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	size := len(keys)

	jsonElem := make([]UserKeys, size)
	roles := make([]bool, size)

	if size > duration {

		// twice the amount
		// client and admin keys
		for i := 0; i < size/2; i++ {
			roles[i] = false
		}
		for i := size / 2; i < size; i++ {
			roles[i] = true
		}

	} else {

		for i := 0; i < size; i++ {
			roles[i] = false
		}

	}

	dateIdx := 0
	for i := 0; i < size; i++ {

		if dateIdx == len(dates) {
			dateIdx = 0
		}

		base64key := base64.StdEncoding.EncodeToString(keys[i])
		jsonElem[i] = UserKeys{Admin: roles[i], Date: dates[dateIdx], Key: base64key}

		str := hex.EncodeToString(keys[i])
		fmt.Println(str)

		dateIdx++
	}

	// static := UserKeys{Admin: true, Date: "2015-11-10", Key: staticKey}
	// jsonElem = append(jsonElem, static)

	fmt.Println("Done!")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonElem)

	userHashB64 := base64.StdEncoding.EncodeToString(userHash)
	fmt.Println(len([]byte(userHashB64)))
	w.Write([]byte(userHashB64))
}

/*func GetUpdatedKeys(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	decoder := json.NewDecoder(r.Body)

	req := User{}
	err := decoder.Decode(&req)
	if err != nil {
		panic(err)
	}

	username := req.User

	fmt.Println("User ", username)

	keys := rfid_db.GetUpdatedKeys(conn, username)

	// jsonElem := make([]LockKeyDates, len(keys))

	jsonString, err := json.Marshal(keys)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonString)

}*/

func Login(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	decoder := json.NewDecoder(r.Body)

	req := User{}
	err := decoder.Decode(&req)
	if err != nil {
		panic(err)
	}

	// admin := req.Admin
	user := req.User

	fmt.Println("User is", user)
	keys := rfid_db.GetUpdatedKeys(conn, user)

	/*jsonString, err := json.Marshal(keys)
	if err != nil {

		panic(err)
	}*/

	// w.Header().Set("Content-Type", "application/json")

	userid := rfid_db.SearchUser(conn, user)
	w.Write([]byte(strconv.Itoa(userid)))
	w.Write([]byte("\n"))

	json.NewEncoder(w).Encode(keys)

	userHash := rfid_db.GetUserHash(conn, user)

	fmt.Println("USER HASH", hex.EncodeToString(userHash))
	userHashB64 := base64.StdEncoding.EncodeToString(userHash)

	w.Write([]byte(userHashB64))
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()

	name := r.Form.Get("User")
	pass := r.Form.Get("Pass")

	fmt.Println("name ", name)
	rfid_db.RegisterUser(conn, name, pass)
}

func SubmitLogs(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	err := r.ParseForm()
	if err != nil {

		panic(err)
	}

	username := r.Form.Get("User")
	lockalias := r.Form.Get("Lock")
	logs := r.Form.Get("Logs")

	if !rfid_db.IsAdmin(conn, username, lockalias) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("User has no admin privileges or lock doesn't exist\n"))
		return
	}

	f, err := ioutil.TempFile("", "file.log")
	if err != nil {
		panic(err)
	}

	_, err = f.Write([]byte(logs))
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {

		panic(err)
	}

	// TODO: test
}

func UpdateMasterkey(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	decoder := json.NewDecoder(r.Body)

	req := UserLock{}
	err := decoder.Decode(&req)
	if err != nil {
		panic(err)
	}

	username := req.User
	lockalias := req.Lock

	masterkey := rfid_db.GenerateMasterKey(conn)
	userid := rfid_db.SearchUser(conn, username)
	lockid := rfid_db.SearchLock(conn, lockalias)

	userlockid, typeid := rfid_db.SearchUserLock(conn, userid, lockid)
	usertype := rfid_db.SearchUserTypeByID(conn, typeid)

	if usertype != "Admin" {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("User has no admin privileges or lock doesn't exist\n"))
		return
	}

	if userlockid == -1 {

		w.WriteHeader(404)
		w.Write([]byte("No user lock instance found for " + lockalias))
	}

	fmt.Println("Updating master key ", hex.EncodeToString([]byte(masterkey)))
	rfid_db.UpdateMasterkey(conn, lockid, masterkey)

	masterkey = base64.StdEncoding.EncodeToString([]byte(masterkey))

	fmt.Println("Base64 masterkey", masterkey)
	w.Write([]byte(masterkey))
}
