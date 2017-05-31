package rfid_db

import (
	//"container/list"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/hkdf"
	//"bytes"
)

var MASTER_KEY string
var INSERT_USER = "Insert into Users(username, email,password) values(?,?,?)"
var SEARCH_USER_LOCK = "select UL.id from UserLocks as UL join Users as U on U.id=UL.userid join Locks as L on L.id=UL.lockid where U.username=? and L.lockalias=?"
var SEARCH_USER = "select id from Users where username=?"

type KeysDates struct {
	Admin bool
	Dates []string
	Keys  []string
}

type AdminKeys struct {
	Lockalias      string
	Masterkey      string
	Maintenancekey string
}

func InitConn() *sql.DB {

	byteArray := []byte{0x4c, 0xcd, 0x08, 0x9b, 0x28, 0xff, 0x96, 0xda, 0x9d, 0xb6, 0xc3, 0x46, 0xec, 0x11, 0x4e, 0x0f, 0x5b, 0x8a, 0x31, 0x9f, 0x35, 0xab, 0xa6, 0x24, 0xda, 0x8c, 0xf6, 0xed, 0x4f, 0xb8, 0xa6, 0xfb}
	MASTER_KEY = string(byteArray)

	conn, err := sql.Open("sqlite3", "./rfid_db/db_new.db")
	CheckErr(err)

	// hardcoded for testing purposes

	/*hash := sha256.New()

	passBytes := []byte("password")
	hash.Write(passBytes)

	password := hex.EncodeToString(passBytes)
	fmt.Println("Inserted password ", password)

	InsertUser(conn, "John Doe", "", password)
	InsertUser(conn, "Peter Doe", "", password)

	InsertLock(conn, "lock 1")
	InsertLock(conn, "lock 2")
	InsertLock(conn, "lock 3")

	// InsertAdmin(conn, "John Admin", password, "lock 1")
	AssignLockToUser(conn, "John Doe", "lock 1", true)
	AssignLockToUser(conn, "Peter Doe", "lock 1", false)
	AssignLockToUser(conn, "John Doe", "lock 2", false)*/

	// InsertUserLock(conn, userid, lockid, "Client")

	/*tx, err := conn.Begin()
	stmt, err := conn.Prepare("insert into UserLock(userid, lockid, typeid, masterkey) values (?,?,?,?)")
	CheckErr(err)

	defer stmt.Close()
	_, err = tx.Stmt(stmt).Exec(userid, lockid, typeid, "")
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)*/

	return conn
}

func RegisterUser(conn *sql.DB, name string, email string, password string) int {

	return InsertUser(conn, name, email, password)
}

// Insertion methods /////////
func InsertUser(conn *sql.DB, name string, email string, password string) int {

	tx, err := conn.Begin()
	CheckErr(err)

	stmt, err := conn.Prepare(INSERT_USER)
	CheckErr(err)

	defer stmt.Close()

	_, err = tx.Stmt(stmt).Exec(name, email, password)
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)

	fmt.Println("user inserted\n")

	return SearchUser(conn, name)
}

func InsertAdmin(conn *sql.DB, username string, password string, lockalias string) int {

	userid := SearchUser(conn, username)

	if lockalias == "" {

		return -1
	}

	if userid != -1 {
		// username already exists
		return -1
	}

	userid = InsertUser(conn, username, "", password)
	lockid := SearchLock(conn, lockalias)

	if userid != -1 && lockid != -1 {
		InsertUserLock(conn, userid, lockid, "Admin")
	} else {

		return -1
	}

	return userid
}

func InsertUserLock(conn *sql.DB, userid int, lockid int, usertype string) (userlockid int) {

	if userid < 0 || lockid < 0 {

		fmt.Println("Invalid id on InsertUserLock")
		return -1
	}

	typeid := SearchUserType(conn, usertype)

	if typeid < 0 {

		fmt.Println("Invalid typeid on InsertUserLock")
		return -1
	}

	query := "insert into UserLock (userid, lockid, typeid) values (?,?,?)"

	tx, err := conn.Begin()
	CheckErr(err)

	stmt, err := conn.Prepare(query)
	CheckErr(err)

	defer stmt.Close()

	_, err = tx.Stmt(stmt).Exec(userid, lockid, typeid)
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)

	userlockid, _ = SearchUserLock(conn, userid, lockid)
	return userlockid
}

func InsertLock(conn *sql.DB, alias string) int {

	masterkey := GenerateMasterKey(conn)
	maintenancekey := GenerateMasterKey(conn)

	tx, err := conn.Begin()
	CheckErr(err)

	stmt, err := conn.Prepare("Insert into Locks(lockalias, masterkey, adminkey) values(?,?,?)")
	CheckErr(err)

	defer stmt.Close()
	_, err = tx.Stmt(stmt).Exec(alias, masterkey, maintenancekey)
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)

	lockid := SearchLock(conn, alias)

	/*
		if providerName != "" {

			providerid := FindProvider(conn, providerName)

			typeid := SearchUserType(conn, "Provider")

			if typeid != -1 {

				userlockid := InsertUserLock(conn, providerid, lockid, "Provider")

				if userlockid == -1 {

					fmt.Println("Invalid userlockid after insertion")
				}

			} else {

				fmt.Errorf("Invalid type ID on InsertLock")
			}

		}*/

	fmt.Println("insert lock ", lockid)
	return lockid
}

func AssignLockToUser(conn *sql.DB, username string, lockalias string, isAdmin bool) {

	userid := SearchUser(conn, username)
	lockid := SearchLock(conn, lockalias)

	role := ""

	if isAdmin {

		role = "Admin"
	} else {

		role = "Client"
	}

	typeid := SearchUserType(conn, role)

	rows, err := conn.Query(SEARCH_USER_LOCK, username, lockalias)

	if rows != nil {
		defer rows.Close()
	}

	userlockid := -1

	if rows != nil && rows.Next() {

		err = rows.Scan(&userlockid)
		CheckErr(err)

		tx, err := conn.Begin()
		CheckErr(err)

		stmt, err := conn.Prepare("Update into UserLock set typeid=? where id=?")
		CheckErr(err)

		defer stmt.Close()

		_, err = tx.Stmt(stmt).Exec(typeid, userlockid)
		CheckErr(err)

		err = tx.Commit()
		CheckErr(err)

	} else {

		InsertUserLock(conn, userid, lockid, role)
		// query = "Insert into UserLock(userid, lockid, typeid) values(?,?,?)"
	}

}

/*func AssignLockToUser(conn *sql.DB, username string, lockalias string) {

	userid := SearchUser(conn, username)
	lockid := SearchLock(conn, lockalias)
	typeid := SearchUserType(conn, "Client")

	tx, err := conn.Begin()
	CheckErr(err)

	stmt, err := conn.Prepare("Insert into UserLock(userid, lockid, typeid) values(?,?,?)")
	CheckErr(err)

	defer stmt.Close()
	_, err = tx.Stmt(stmt).Exec(userid, lockid, typeid)
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)
}*/

// Querie / Getters
func SearchUser(conn *sql.DB, username string) int {

	rows, err := conn.Query("select id from Users where username=?", username)
	defer rows.Close()

	var userid = -1
	if rows.Next() {

		err = rows.Scan(&userid)
		CheckErr(err)
	}

	return userid
}

func SearchUserType(conn *sql.DB, usertype string) int {

	rows, err := conn.Query("select id from UserTypes where type=?", usertype)
	defer rows.Close()

	var id = -1
	if rows.Next() {

		err = rows.Scan(&id)
		CheckErr(err)
	}

	return id
}

func SearchLock(conn *sql.DB, lockalias string) int {

	rows, err := conn.Query("select id from Locks where lockalias=?", lockalias)
	defer rows.Close()

	var lockid = -1
	if rows.Next() {

		err = rows.Scan(&lockid)
		CheckErr(err)
	}

	return lockid
}

func MatchPassword(conn *sql.DB, username string, password string) bool {

	fmt.Println("DEBUG: ", username, password)

	rows, err := conn.Query("select password from Users where username=?", username)

	defer rows.Close()
	CheckErr(err)

	hash := sha256.New()

	passwordBytes := []byte(password)
	hash.Write(passwordBytes)

	password = hex.EncodeToString(passwordBytes)

	var pass string
	if rows.Next() {

		err = rows.Scan(&pass)
		CheckErr(err)

		fmt.Println(pass)
		fmt.Println(password)

		return password == pass
	}

	return false
}

func SearchUserLock(conn *sql.DB, userid int, lockid int) (userlockid int, typeid int) {

	fmt.Println("searching userlock... ", userid, lockid)
	rows, err := conn.Query("select UL.id, UL.typeid from UserLock as UL where UL.userid=? and UL.lockid=?", userid, lockid)

	CheckErr(err)
	defer rows.Close()

	userlockid = -1
	if rows.Next() {

		err = rows.Scan(&userlockid, &typeid)
		CheckErr(err)

	}

	return userlockid, typeid
}

func UpdateMasterkey(conn *sql.DB, lockid int, masterkey string) {

	tx, err := conn.Begin()
	CheckErr(err)

	stmt, err := conn.Prepare("update Locks set masterkey=? where id=?")

	CheckErr(err)
	defer stmt.Close()

	_, err = tx.Stmt(stmt).Exec(masterkey, lockid)
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)

}

func UpdateMaintenancekey(conn *sql.DB, lockid int, adminkey string) {

	tx, err := conn.Begin()
	CheckErr(err)

	stmt, err := conn.Prepare("update Locks set adminkey=? where id=?")

	CheckErr(err)
	defer stmt.Close()

	_, err = tx.Stmt(stmt).Exec(adminkey, lockid)
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)
}

func SearchUserTypeByID(conn *sql.DB, typeid int) (usertype string) {

	rows, err := conn.Query("select type from UserTypes where id=?", typeid)
	CheckErr(err)
	defer rows.Close()

	if rows.Next() {

		err = rows.Scan(&usertype)
		CheckErr(err)
	}

	return usertype
}

/*
conn
* date
* lock
* returns: dates, keys, userHash
*/
func ComputeKeys(conn *sql.DB, date string, username string, lockalias string, duration int) ([]string, [][]byte, []byte) {

	// setting date layout
	dateLayout := "2006-Jan-02"
	currentDate := time.Now().Local().Format(dateLayout)

	currentDateFormated, err := time.Parse(dateLayout, currentDate)
	CheckErr(err)

	dateObj, err := time.Parse(dateLayout, date)
	CheckErr(err)

	fmt.Println("Date: ", dateObj)
	if currentDateFormated.After(dateObj) {
		return nil, nil, nil
	}

	if duration < 1 {
		fmt.Println("Invalid duration")
		return nil, nil, nil
	}

	// retrieving IDs
	userid := SearchUser(conn, username)
	// typeid := SearchUserType(conn, "Client")
	lockid := SearchLock(conn, lockalias)

	// checking if userlock instance exists
	userlockid, typeid := SearchUserLock(conn, userid, lockid)

	if userlockid == -1 {

		userlockid = InsertUserLock(conn, userid, lockid, "Client")

		if userlockid == -1 {

			fmt.Errorf("Unable to create new userlock instance")
			return nil, nil, nil
		}
	}

	usertype := SearchUserTypeByID(conn, typeid)
	hash := sha256.New

	// Cryptographically secure master key

	var lockKeys []string
	maintenancekey := ""

	master := GetMasterKey(conn, lockid)

	if master == "" {

		master = GenerateMasterKey(conn)
		UpdateMasterkey(conn, lockid, master)

	}

	lockKeys = append(lockKeys, master)

	if usertype == "Admin" {

		maintenancekey = GetMaintenanceKey(conn, userlockid)

		lockKeys = append(lockKeys, maintenancekey)
	}

	if len(lockKeys) < 1 || len(lockKeys) > 2 {

		fmt.Errorf("error retrieving lock keys")
		return nil, nil, nil
	}

	dateNumbers := make([]int, 3)
	dateBytes := make([]byte, 3)

	// array of required keys
	// keys := make([][]byte, duration)
	// dateStrings := make([]string, duration)
	keys := make([][]byte, duration*len(lockKeys))
	dateStrings := make([]string, duration)

	// retrieving user 'key'
	userBytes := []byte(username)
	userHash := sha1.New()
	userHash.Write(userBytes)

	userHashBytes := userHash.Sum(nil)
	str := hex.EncodeToString(userHashBytes)
	fmt.Println("User ", str)

	lockKeysLen := len(lockKeys)

	initialDate := dateObj
	for k := 0; k < lockKeysLen; k++ {

		dateIdx := 0

		for i := 0; i < duration; i++ {

			dateStrings[dateIdx] = strings.Fields(dateObj.String())[0]

			fmt.Println("Date strings", dateStrings)
			// getting last two chars of year
			yearStr := strings.SplitN(dateStrings[i], "-", 3)[0]

			yearArr := strings.SplitN(yearStr, "", 4)
			year, err := strconv.Atoi(yearArr[2] + yearArr[3])
			CheckErr(err)

			dateNumbers[0] = year
			dateNumbers[1] = int(dateObj.Month())
			dateNumbers[2] = int(dateObj.Day())

			//dateSalt := make([]byte,3)

			for j := 0; j < 3; j++ {

				dateBytes[j] = byte(dateNumbers[j])
				//dateSalt[j]alt,dateBytes[j]
			}

			str := hex.EncodeToString(dateBytes)
			fmt.Println("Date bytes ", str)

			// Create the key derivation function
			// TODO: maybe rewrite operation to obtain raw PRK,
			// 		 retrieve when needed and then expand it
			// hkdf := hkdf.New(hash, []byte(master), dateBytes, userHashBytes)
			hkdf := hkdf.New(hash, []byte(lockKeys[k]), dateBytes, userHashBytes)
			keys[i] = make([]byte, 32)

			n, err := io.ReadFull(hkdf, keys[i])
			CheckErr(err)

			if n != len(keys[i]) {
				fmt.Println("error with key length\n")
				return nil, nil, nil
			}

			AssignKeyToUser(conn, userlockid, keys[i], dateNumbers)

			// incrementing date
			dateObj = dateObj.Add(time.Hour * 24)
			dateIdx++
		}

		dateObj = initialDate
	}

	return dateStrings, keys, userHashBytes
}

func GetMaintenanceKey(conn *sql.DB, userlockid int) (key string) {

	rows, err := conn.Query("select adminkey from Locks as L join UserLock as UL on UL.lockid=L.id where UL.id=?", userlockid)
	CheckErr(err)

	defer rows.Close()

	if rows.Next() {

		err = rows.Scan(&key)
		CheckErr(err)
	}

	return key
}

func AssignKeyToUser(conn *sql.DB, userlockid int, key []byte, dateNumbers []int) {

	tx, err := conn.Begin()
	CheckErr(err)

	fullDate := time.Date(dateNumbers[0], time.Month(dateNumbers[1]), dateNumbers[2], 0, 0, 0, 0, time.UTC).String()
	date := strings.Fields(fullDate)[0]

	stmt, err := conn.Prepare("insert into Keys(key, date, userlockid) values(?,?,?)")
	CheckErr(err)

	defer stmt.Close()

	_, err = tx.Stmt(stmt).Exec(key, date, userlockid)
	CheckErr(err)

	err = tx.Commit()
	CheckErr(err)
}

func FindProvider(conn *sql.DB, providerName string) int {

	rows, err := conn.Query("select U.id from Users as U join UserLock as UL on U.id=UL.userid where U.name=?", providerName)
	CheckErr(err)

	defer rows.Close()

	providerid := 0
	if rows.Next() {
		err = rows.Scan(&providerid)
		CheckErr(err)
	}

	return providerid

}

func GetUserHash(conn *sql.DB, username string) []byte {

	userid := SearchUser(conn, username)

	if userid == -1 {

		return nil
	}

	userBytes := []byte(username)

	userHash := sha1.New()
	userHash.Write(userBytes)

	return userBytes
}

func GetUpdatedKeys(conn *sql.DB, username string) map[string]KeysDates {

	// setting date layout
	dateLayout := "2006-01-02"
	currentDate := time.Now().Local().Format(dateLayout)

	currentDateFormated, err := time.Parse(dateLayout, currentDate)
	CheckErr(err)

	userid := SearchUser(conn, username)
	// typeid := SearchUserType(conn, "Client")

	rows, err := conn.Query("select UL.id, UL.typeid from UserLock as UL where UL.userid=?", userid)

	defer rows.Close()

	m := make(map[string]KeysDates)

	for rows.Next() {

		// todo: fazer isto para
		var userlockid = -1
		var typeid = -1
		rows.Scan(&userlockid, &typeid)

		usertype := SearchUserTypeByID(conn, typeid)
		admin := (usertype == "Admin")

		fmt.Println("result", userlockid)

		// todo: continue refactoring
		results, err := conn.Query("select L.lockalias, K.date, K.key from Keys as K join UserLock as UL on UL.id=K.userlockid join Locks as L on L.id=UL.lockid where K.userlockid=?", userlockid)
		CheckErr(err)

		defer results.Close()

		for results.Next() {

			var date string
			var key string
			var lockalias string

			err = results.Scan(&lockalias, &date, &key)
			CheckErr(err)

			lockMap, ok := m[lockalias]

			if !ok {

				var tmpDates []string
				var tmpKeys []string
				lockMap = KeysDates{Admin: admin, Dates: tmpDates, Keys: tmpKeys}
			}

			date = "2" + date[1:len(date)]
			dateDB, err := time.Parse(dateLayout, date)
			CheckErr(err)

			if !currentDateFormated.After(dateDB) {

				lockMap.Dates = append(lockMap.Dates, dateDB.String())
				lockMap.Keys = append(lockMap.Keys, key)

				m[lockalias] = lockMap
			}

		}

		return m
	}

	return nil
}

func IsAdmin(conn *sql.DB, username string, lockalias string) bool {

	userid := SearchUser(conn, username)

	if userid == -1 {

		return false
	}

	typeid := SearchUserType(conn, "Admin")

	query := "select id from UserLock where userid=?"

	if lockalias != "" {

		query += " and lockid=? and typeid=?"
		lockid := SearchLock(conn, lockalias)

		if lockid == -1 {

			return false
		}

		rows, err := conn.Query(query, userid, lockid, typeid)
		CheckErr(err)

		defer rows.Close()

		if rows.Next() {

			return true
		}

	} else {

		query += " and typeid=?"

		rows, err := conn.Query(query, userid, typeid)
		CheckErr(err)

		defer rows.Close()

		if rows.Next() {

			// if user owns any lock
			return true
		}
	}

	return false
}

func GetAdminKeys(conn *sql.DB, username string) []AdminKeys {

	typeid := SearchUserType(conn, "Admin")
	rows, err := conn.Query("select L.lockalias, L.masterkey, L.adminkey from Locks as L join UserLock as UL on UL.lockid=L.lockid join Users as U on U.id=UL.userid where U.username=? and UL.typeid=?", username, typeid)
	CheckErr(err)

	defer rows.Close()

	var keys []AdminKeys

	// m := make(map[string]string)

	for rows.Next() {

		var lockalias string
		var masterkey string
		var adminkey string

		err = rows.Scan(&lockalias, &masterkey, &adminkey)
		CheckErr(err)

		instance := AdminKeys{Lockalias: lockalias, Masterkey: masterkey, Maintenancekey: adminkey}
		keys = append(keys, instance)
		// m[lockalias] = masterkey
	}

	return keys
}

func GetMasterKey(conn *sql.DB, lockid int) string {

	fmt.Println("Query beggining")

	fmt.Println("Lock id searching master key...", lockid)
	rows, err := conn.Query("select masterkey from Locks where id=?", lockid)

	defer rows.Close()

	masterkey := ""

	if rows.Next() {

		err = rows.Scan(&masterkey)
		CheckErr(err)
	}

	fmt.Println(masterkey)
	return masterkey
}

func GenerateMaintenanceKey(conn *sql.DB) string {

	masterkey := [32]byte{0x4c, 0xcd, 0x08, 0x9b, 0x28, 0xff, 0x96, 0xda,
		0x9d, 0xb6, 0xc3, 0x46, 0xec, 0x11, 0x4e, 0x0f,
		0x5b, 0x8a, 0x31, 0x9f, 0x35, 0xab, 0xa6, 0x24,
		0xda, 0x8c, 0xf6, 0xed, 0x4f, 0xb8, 0xa6, 0xfb}

	return string(masterkey[:32])
}

func GenerateMasterKey(conn *sql.DB) string {

	token := make([]byte, 32)
	rand.Read(token)

	return string(token)
}

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}
