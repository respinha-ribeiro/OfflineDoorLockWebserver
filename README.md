# OfflineDoorLockWebserver

Webserver implemented in Go for the Radio Frequency Identification course of my graduation. 
The project was an offline access-control doorlock system based on periodic key-derivation mechanism using [HDKF](https://tools.ietf.org/html/rfc5869). 
A user could interact with the door lock - an NFC-enabled embedded system which derivates periodic access keys - with an Android app which stores day-specific keys offline.
The system was enhanced to provide role-based access control and maintenance operations without any network connection on the doorlock. 

This resulted in [an article](http://wiki.ieeta.pt/wiki/index.php/Azevedo-2017a) for the portuguese national conference [INForum](http://inforum.org.pt/INForum2017).
