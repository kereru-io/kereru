CREATE TABLE `Migration` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `Version` int(11) DEFAULT NULL,
  PRIMARY KEY (`ID`)
);

CREATE TABLE `Images` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `GUID` varchar(500) DEFAULT NULL,
  `ImageName` varchar(255) DEFAULT NULL,
  `Description` varchar(512) DEFAULT NULL,
  `UploadTime` bigint(20) DEFAULT NULL,
  `Filename` varchar(512) DEFAULT NULL,
  `Filesize` int(11) DEFAULT NULL,
  `Notes` varchar(255) DEFAULT NULL,
  `MediaID` text DEFAULT NULL,
  `MediaTime` int(11) DEFAULT NULL,
  PRIMARY KEY (`ID`)
);

CREATE TABLE `PasswordResets` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `UID` int(11) DEFAULT NULL,
  `Email` varchar(120) DEFAULT NULL,
  `Token` varchar(120) DEFAULT NULL,
  `ResetTime` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`ID`)
);

CREATE TABLE `Roles` (
  `RoleID` int(11) NOT NULL AUTO_INCREMENT,
  `RoleName` varchar(120) DEFAULT NULL,
  `Access` int(11) DEFAULT NULL,
  PRIMARY KEY (`RoleID`)
);

LOCK TABLES `Roles` WRITE;
INSERT INTO `Roles` VALUES (1,'No Access',0),(2,'Admin',65535),(3,'Editor',61368),(4,'Publisher',61432);
UNLOCK TABLES;

CREATE TABLE `TweetAudit` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `UserID` int(11) DEFAULT NULL,
  `Time` bigint(20) DEFAULT NULL,
  `TweetID` int(20) DEFAULT NULL,
  `Status` int(20) DEFAULT NULL,
  PRIMARY KEY (`ID`)
);

CREATE TABLE `Tweets` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `SendTime` bigint(20) DEFAULT NULL,
  `Message` varchar(8192) DEFAULT NULL,
  `ImageA` int(10) DEFAULT 0,
  `ImageB` int(10) DEFAULT 0,
  `ImageC` int(10) DEFAULT 0,
  `ImageD` int(10) DEFAULT 0,
  `VideoA` int(10) DEFAULT 0,
  `Notes` text DEFAULT NULL,
  `Status` int(11) DEFAULT NULL,
  PRIMARY KEY (`ID`)
);

CREATE TABLE `Users` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `Username` varchar(50) DEFAULT NULL,
  `Password` varchar(120) DEFAULT NULL,
  `EmailAddress` varchar(120) DEFAULT NULL,
  `Role` int(10) DEFAULT 0,
  `FirstName` varchar(50) DEFAULT NULL,
  `LastName` varchar(50) DEFAULT NULL,
  PRIMARY KEY (`ID`)
);

CREATE TABLE `Videos` (
  `ID` int(11) NOT NULL AUTO_INCREMENT,
  `GUID` varchar(500) DEFAULT NULL,
  `VideoName` varchar(255) DEFAULT NULL,
  `Description` varchar(512) DEFAULT NULL,
  `UploadTime` bigint(20) DEFAULT NULL,
  `Filename` varchar(512) DEFAULT NULL,
  `Filesize` int(11) DEFAULT NULL,
  `Notes` varchar(255) DEFAULT NULL,
  `MediaID` text DEFAULT NULL,
  `MediaTime` int(11) DEFAULT NULL,
  PRIMARY KEY (`ID`)
);
