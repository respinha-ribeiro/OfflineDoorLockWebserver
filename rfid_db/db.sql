CREATE TABLE  Users  (
	 id  integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	 username 	text NOT NULL,
	 password text NOT NULL,
	 email 	text 
);

CREATE TABLE Locks (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	lockalias text,
	masterkey text not null,
	adminkey text not null
);

CREATE TABLE UserTypes (
	id integer not null primary key autoincrement,
	type text not null unique
);

CREATE TABLE  UserLock  (
	id  integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	userid integer not null,
	lockid integer not null,
	typeid integer not null, 
	foreign key (userid) references Users(id),
	foreign key (lockid) references Locks(id),
	foreign key (typeid) references UserTypes(id)
);

CREATE TABLE Keys (
	id  integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	key text,
	date text,
	userlockid integer not null,
	admin not null,
	foreign key(userlockid) references UserLock(id)
);

insert into UserTypes(type) values ("Client");
insert into UserTypes(type) values ("Admin");
insert into UserTypes(type) values ("Provider");

/*insert into Users(username, password) values ("John Doe", "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8");
insert into Locks(lockalias) values ("MyLock");
insert into UserLock(userid, lockid, typeid) values (1, 1, 1);*/
