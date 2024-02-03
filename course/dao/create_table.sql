show databases;
create database search;
create user 'tester' identified by '123456';
grant all on search.* to tester;
use search;
create table if not exists bili_video(
	id char(12) comment 'bili视频ID',
	title varchar(250) not null comment '视频标题',
    author varchar(60) not null comment '视频作者',
	post_time datetime not null comment '视频发布时间',
    keywords varchar(200) not null comment '标签关键词',
    view int not null default 0 comment '播放量',
    thumbs_up int not null default 0 comment '点赞量',
    coin int not null default 0 comment '投币',
    favorite int not null default 0 comment '收藏',
    share int not null default 0 comment '分享',
	primary key (id)
)default charset=utf8mb4 comment '抓取的bili视频信息';