CREATE TABLE  User  (
	 id  integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	 name 	text NOT NULL,
	 password text NOT NULL,
	 email 	text 
);

CREATE TABLE  Client  (
	id  integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	userid  integer NOT NULL,
	foreign key( userid ) references User(id)
);

CREATE TABLE  Admin  (
	 id  integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	 userid  integer NOT NULL,
	foreign key( userid ) references User(id)
);

CREATE TABLE  Provider  (
	 id 	integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	 userid  integer NOT NULL,
	foreign key( userid ) references User(id)
);

	
CREATE TABLE Locker (
	id integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	description text,
	key text not null,
	providerid integer NOT NULL,
	foreign key(providerid) references Provider(id)
);



CREATE TABLE Attribution (
	id	integer NOT NULL PRIMARY KEY AUTOINCREMENT,
	lockerid integer NOT NULL,
	clientid integer not null,	
	key text not null,
	date text not null,
	foreign key(lockerid) references Locker(id),
	foreign key(clientid) references Client(id)
	
);

