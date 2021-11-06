DROP DATABASE IF EXISTS acm_bot;
CREATE DATABASE acm_bot;
\c acm_bot;

CREATE TYPE chan_type AS ENUM ('daily', 'set', 'action sack');
CREATE TYPE picking_method AS ENUM ('any', 'easy', 'medium', 'hard', 'none');

DROP TABLE IF EXISTS member CASCADE;
CREATE TABLE member (
	user_id		VARCHAR(100),
	leetcode_user	VARCHAR(100),

	PRIMARY KEY (user_id),
	UNIQUE (leetcode_user)
);

DROP TABLE IF EXISTS guild CASCADE;
CREATE TABLE guild (
	guild_id varchar(100),
	PRIMARY KEY (guild_id)
);

DROP TABLE IF EXISTS problem CASCADE;
CREATE TABLE problem (
	slug	VARCHAR(256),
	title   VARCHAR(256),
	total_accept int,
	total_subs int,
	total_likes int,
	total_dislikes int,
	difficulty int,

	PRIMARY KEY (slug)
);

DROP TABLE IF EXISTS channel CASCADE;
CREATE TABLE channel (
	channel_id	VARCHAR(100),
	guild_id	VARCHAR(100),

	PRIMARY KEY (channel_id),
	FOREIGN KEY (guild_id) REFERENCES guild(guild_id)
);

DROP TABLE IF EXISTS daily_channel CASCADE;
CREATE TABLE daily_channel (
	channel_id	VARCHAR(100),
	pick		picking_method,	
	current_prob	INTEGER NOT NULL,

	PRIMARY KEY (channel_id),
	FOREIGN KEY (channel_id) REFERENCES channel(channel_id)
);

DROP TABLE IF EXISTS daily_participant CASCADE;
CREATE TABLE daily_participant (
	user_id		VARCHAR(100),
	channel_id	VARCHAR(100),
	last_solved	VARCHAR(256),
	
	FOREIGN KEY (user_id) REFERENCES member(user_id),
	FOREIGN KEY (channel_id) REFERENCES daily_channel(channel_id),
	PRIMARY KEY (user_id, channel_id),
	FOREIGN KEY (last_solved) REFERENCES problem(slug)
);

DROP TABLE IF EXISTS score CASCADE;
CREATE TABLE score (
	guild_id	VARCHAR(100),
	user_id		VARCHAR(100),
	value		INTEGER,

	FOREIGN KEY (guild_id) REFERENCES guild(guild_id),
	FOREIGN KEY (user_id) REFERENCES member(user_id),
	PRIMARY KEY (guild_id, user_id)
);

DROP TABLE IF EXISTS schedule CASCADE;
CREATE TABLE schedule (
	prob_number	INTEGER,
	channel_id	VARCHAR(100),
	problem_slug	VARCHAR(256),
	
	PRIMARY KEY (prob_number, channel_id),
	FOREIGN KEY (channel_id) REFERENCES daily_channel(channel_id),
	FOREIGN KEY (problem_slug) REFERENCES problem(slug)
);

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO acm_bot;
