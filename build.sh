rm rfid_db/*.db 
sqlite3 rfid_db/db_new.db < rfid_db/db.sql
go build webserver_new.go
