/*
SQLyog Ultimate v12.5.0 (64 bit)
MySQL - 8.0.11 : Database - pusoy
*********************************************************************
*/

/*!40101 SET NAMES utf8 */;

/*!40101 SET SQL_MODE=''*/;

/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
CREATE DATABASE /*!32312 IF NOT EXISTS*/`pusoy` /*!40100 DEFAULT CHARACTER SET utf8 */;

USE `pusoy`;

/*Table structure for table `diamond_records` */

DROP TABLE IF EXISTS `diamond_records`;

CREATE TABLE `diamond_records` (
  `timestamp` int(10) unsigned NOT NULL,
  `from` int(10) unsigned NOT NULL,
  `to` int(10) unsigned NOT NULL,
  `diamonds` int(10) unsigned NOT NULL,
  KEY `index_timestamp` (`timestamp`),
  KEY `index_from` (`from`),
  KEY `index_to` (`to`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `facebook_users` */

DROP TABLE IF EXISTS `facebook_users`;

CREATE TABLE `facebook_users` (
  `uid` int(11) unsigned NOT NULL,
  `fbid` varchar(64) NOT NULL,
  PRIMARY KEY (`uid`),
  UNIQUE KEY `index_fbid` (`fbid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `free_diamonds` */

DROP TABLE IF EXISTS `free_diamonds`;

CREATE TABLE `free_diamonds` (
  `uid` int(10) unsigned NOT NULL,
  `time` int(10) unsigned NOT NULL COMMENT '上一次领取的时间',
  PRIMARY KEY (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `game_records` */

DROP TABLE IF EXISTS `game_records`;

CREATE TABLE `game_records` (
  `round_id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `roomid` int(10) unsigned NOT NULL COMMENT 'room id',
  `round` int(10) unsigned NOT NULL COMMENT 'round',
  `result` blob NOT NULL COMMENT 'result',
  `timestamp` int(10) unsigned NOT NULL,
  PRIMARY KEY (`round_id`),
  KEY `index_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `game_statistics` */

DROP TABLE IF EXISTS `game_statistics`;

CREATE TABLE `game_statistics` (
  `timestamp` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '每天0点时间戳',
  `new_users` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '新增用户数',
  `active_users` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '活跃用户数(>=5局)',
  `play_users` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '游戏用户数',
  `cost_diamonds_users` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '消耗钻石用户数',
  `create_rooms` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '创建的房间数',
  `played_rounds` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '游戏局数',
  `cost_diamonds` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '消耗钻石数',
  `offer_diamonds` int(10) unsigned NOT NULL DEFAULT '0' COMMENT '钻石发放数',
  PRIMARY KEY (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `online_statistics` */

DROP TABLE IF EXISTS `online_statistics`;

CREATE TABLE `online_statistics` (
  `timestamp` int(10) unsigned NOT NULL,
  `max_online_users` int(10) unsigned NOT NULL,
  `max_playing_users` int(10) unsigned NOT NULL,
  `max_playing_rooms` int(10) unsigned NOT NULL,
  PRIMARY KEY (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `room_records` */

DROP TABLE IF EXISTS `room_records`;

CREATE TABLE `room_records` (
  `room_id` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'room id',
  `name` varchar(200) NOT NULL COMMENT 'name',
  `number` int(10) unsigned NOT NULL COMMENT 'room number encoded',
  `owner_uid` int(10) unsigned NOT NULL COMMENT 'uid of the user created room',
  `hands` int(10) unsigned NOT NULL COMMENT 'total hands,0-无限',
  `played_hands` int(10) unsigned NOT NULL COMMENT 'hands already played',
  `is_share` tinyint(1) NOT NULL COMMENT 'aa制',
  `min_bet` int(10) unsigned NOT NULL COMMENT 'min bet',
  `max_bet` int(10) unsigned NOT NULL COMMENT 'max bet',
  `credit_points` int(10) unsigned NOT NULL COMMENT 'credit points',
  `create_time` int(10) unsigned NOT NULL COMMENT 'create time',
  `close_time` int(10) unsigned NOT NULL COMMENT 'close time',
  `closed` tinyint(1) DEFAULT NULL COMMENT '是否已关闭',
  PRIMARY KEY (`room_id`),
  KEY `index_number_closed` (`number`,`closed`),
  KEY `index_owner_closed` (`owner_uid`,`closed`),
  KEY `index_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `round_players` */

DROP TABLE IF EXISTS `round_players`;

CREATE TABLE `round_players` (
  `round_id` int(10) unsigned NOT NULL,
  `uid` int(10) unsigned NOT NULL,
  PRIMARY KEY (`round_id`,`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `scoreboard` */

DROP TABLE IF EXISTS `scoreboard`;

CREATE TABLE `scoreboard` (
  `roomid` int(10) unsigned NOT NULL,
  `uid` int(10) unsigned NOT NULL,
  `score` int(11) DEFAULT NULL,
  KEY `index_roomid_uid` (`roomid`,`uid`),
  KEY `index_uid` (`uid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

/*Table structure for table `users` */

DROP TABLE IF EXISTS `users`;

CREATE TABLE `users` (
  `uid` int(11) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL COMMENT '名字',
  `signture` varchar(64) DEFAULT '' COMMENT '签名',
  `avatar` varchar(255) NOT NULL COMMENT '头像',
  `diamonds` int(11) unsigned NOT NULL DEFAULT '0' COMMENT '钻石',
  `platform` int(10) unsigned NOT NULL COMMENT '0-fb',
  `regtime` int(11) unsigned NOT NULL COMMENT '注册时间',
  `logintime` int(11) unsigned NOT NULL COMMENT '登录时间',
  `status` int(10) unsigned NOT NULL COMMENT '0-正常,1-封号',
  PRIMARY KEY (`uid`),
  KEY `index_regtime` (`regtime`) COMMENT '注册时间'
) ENGINE=InnoDB AUTO_INCREMENT=100000 DEFAULT CHARSET=utf8;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
