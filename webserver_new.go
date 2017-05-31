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
	"strings"

	"./rfid_db"
	_ "github.com/mattn/go-sqlite3"
)

type ClientKeys struct {
	Date string
	Key  string
}

type UserReq struct {
	Admin     bool
	Date      string
	User      string
	Duration  int
	Lockalias string
}

type LockKeyDates struct {
	Lock  string
	Keys  []string
	Dates []string
}

var conn *sql.DB

func main() {

	conn = rfid_db.InitConn()
	defer conn.Close()

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/login", Login)
	http.HandleFunc("/req_keys", RequestKeys)
	http.HandleFunc("/update_keys", GetUpdatedKeys)
	/*http.HandleFunc("/register_user", RegisterUser)*/

	//http.HandleFunc("/submit_logs", SubmitLogs)
	//http.HandleFunc("/update_masterkey", UpdateMasterkey)

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

	dates, keys, userHash := rfid_db.ComputeKeys(conn, start, username, lockalias, duration)

	if dates == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonElem := make([]ClientKeys, duration)

	for i := 0; i < len(keys); i++ {

		base64key := base64.StdEncoding.EncodeToString(keys[i])
		jsonElem[i] = ClientKeys{Date: dates[i], Key: base64key}

		if err != nil {
			panic(err)
		}

		str := hex.EncodeToString(keys[i])
		fmt.Println(str)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonElem)

	userHashB64 := base64.StdEncoding.EncodeToString(userHash)
	w.Write([]byte(userHashB64))
}

func GetUpdatedKeys(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	/* TODO: acrescentar
	r.ParseForm()
	provider := r.Form.Get("Provider")

	if rfid_db.FindProvider(conn, provider) < 0 {

		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized; request performed by a non-provider user \n"))
		return
	}*/

	decoder := json.NewDecoder(r.Body)

	req := UserReq{}
	err := decoder.Decode(&req)
	if err != nil {
		panic(err)
	}

	username := req.User

	// todo: remove this function?
	keysMap := rfid_db.GetUpdatedKeys(conn, username)

	jsonElem := make([]LockKeyDates, len(keysMap))

	i := 0
	for k := range keysMap {

		dates := keysMap[k].Dates
		keys := keysMap[k].Keys

		jsonElem[i] = LockKeyDates{k, keys, dates}
		i += 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonElem)

}
func Login(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	r.ParseForm()

	decoder := json.NewDecoder(r.Body)

	req := UserReq{}
	err := decoder.Decode(&req)
	if err != nil {
		panic(err)
	}

	admin := req.Admin
	user := req.User

	if admin {

		if !rfid_db.IsAdmin(conn, user, "") {

			w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
			w.WriteHeader(401)
			w.Write([]byte("User is not an administrator\n"))
			return
		}

		masterkeys := rfid_db.GetAdminMasterKeys(conn, user)

		jsonString, err := json.Marshal(masterkeys)
		if err != nil {

			panic(err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonString)
	} else {

		keysMap := rfid_db.GetUpdatedKeys(conn, user)
		jsonElem := make([]LockKeyDates, len(keysMap))

		i := 0
		// todo: marshal?
		for k := range keysMap {

			dates := keysMap[k].Dates
			keys := keysMap[k].Keys

			for i := 0; i < len(keys); i++ {

				keys[i] = base64.StdEncoding.EncodeToString([]byte(keys[i]))
			}

			jsonElem[i] = LockKeyDates{k, keys, dates}
			i += 1
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jsonElem)

		userHash := rfid_db.GetUserHash(conn, user)
		userHashB64 := base64.StdEncoding.EncodeToString(userHash)
		w.Write([]byte(userHashB64))
	}

	// todo: test
}

func RegisterUser(w http.ResponseWriter, r *http.Request) {

	r.ParseForm()

	name := r.Form.Get("User")
	pass := r.Form.Get("Pass")
	email := r.Form.Get("Email")

	fmt.Println("name ", name)
	rfid_db.RegisterUser(conn, name, email, pass)
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

	r.ParseForm()
	username := r.Form.Get("User")
	lockalias := r.Form.Get("Lock")

	if !rfid_db.IsAdmin(conn, username, lockalias) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("User has no admin privileges or lock doesn't exist\n"))
		return
	}

	masterkey := rfid_db.GenerateMasterKey(conn)
	userid := rfid_db.SearchUser(conn, username)
	lockid := rfid_db.SearchLock(conn, lockalias)
	typeid := rfid_db.SearchUserType(conn, "Admin")

	userlockid := rfid_db.SearchUserLock(conn, userid, lockid, typeid)

	rfid_db.UpdateMasterkey(conn, userlockid, masterkey)

	// TODO: test
}

func GetMasterkey(w http.ResponseWriter, r *http.Request) {

	if !basicAuth(w, r) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
		return
	}

	r.ParseForm()
	username := r.Form.Get("User")
	lockalias := r.Form.Get("Lock")

	if !rfid_db.IsAdmin(conn, username, lockalias) {

		w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("User has no admin privileges or lock doesn't exist\n"))
		return
	}

	lockid := rfid_db.SearchLock(conn, lockalias)
	_, masterkey := rfid_db.GetMasterKey(conn, lockid)

	// TODO: test
	w.Write([]byte(masterkey))
}
