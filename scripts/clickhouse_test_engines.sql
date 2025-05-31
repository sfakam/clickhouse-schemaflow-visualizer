-- Create a test database for engine coverage
drop database if exists test_engines;
create database test_engines;

-- MergeTree (most common)
drop table if exists test_engines.mt;
create table test_engines.mt (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = MergeTree() order by id;

-- ReplacingMergeTree
drop table if exists test_engines.rmt;
create table test_engines.rmt (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = ReplacingMergeTree() order by id;

-- SummingMergeTree
drop table if exists test_engines.smt;
create table test_engines.smt (
    id UInt32,
    value Float64,
    count UInt32,
    total Float64,
    created_at DateTime,
    status String
) engine = SummingMergeTree() order by id;

-- AggregatingMergeTree
drop table if exists test_engines.amt;
create table test_engines.amt (
    id UInt32,
    sum_value AggregateFunction(sum, Float64),
    avg_value AggregateFunction(avg, Float64),
    min_value AggregateFunction(min, Float64),
    max_value AggregateFunction(max, Float64),
    count_value AggregateFunction(count, UInt32)
) engine = AggregatingMergeTree() order by id;

-- CollapsingMergeTree
drop table if exists test_engines.cmt;
create table test_engines.cmt (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String,
    sign Int8
) engine = CollapsingMergeTree(sign) order by id;

-- VersionedCollapsingMergeTree
drop table if exists test_engines.vcmt;
create table test_engines.vcmt (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String,
    sign Int8,
    version UInt32
) engine = VersionedCollapsingMergeTree(sign, version) order by id;

-- GraphiteMergeTree
drop table if exists test_engines.gmt;
create table test_engines.gmt (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = GraphiteMergeTree('/etc/clickhouse-server/graphite.xml') order by id;

-- ReplicatedMergeTree (example path, needs cluster for real use)
drop table if exists test_engines.rpmt;
create table test_engines.rpmt (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = ReplicatedMergeTree('/clickhouse/tables/{shard}/rpmt', '{replica}') order by id;

-- Distributed (requires a cluster, so just for schema)
drop table if exists test_engines.dist;
create table test_engines.dist as test_engines.mt engine = Distributed('test_cluster', 'test_engines', 'mt', rand());

-- Log
drop table if exists test_engines.log;
create table test_engines.log (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = Log();

-- TinyLog
drop table if exists test_engines.tinylog;
create table test_engines.tinylog (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = TinyLog();

-- StripeLog
drop table if exists test_engines.stripelog;
create table test_engines.stripelog (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = StripeLog();

-- Memory
drop table if exists test_engines.mem;
create table test_engines.mem (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = Memory();

-- Null
drop table if exists test_engines.null;
create table test_engines.null (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = Null();

-- File (CSV)
drop table if exists test_engines.file_csv;
create table test_engines.file_csv (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = File(CSV);

-- Set
drop table if exists test_engines.set;
create table test_engines.set (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = Set;

-- Join
drop table if exists test_engines.jn;
create table test_engines.jn (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = Join(ANY, LEFT, id);

-- Buffer
drop table if exists test_engines.buf;
create table test_engines.buf (
    id UInt32,
    name String,
    value Float64,
    created_at DateTime,
    status String
) engine = Buffer(test_engines, mt, 16, 10, 60, 10, 10000, 100000, 1000000, 10000000);

-- Dictionary (if dictionaries are configured)
-- create table test_engines.dict (
--     id UInt32, val String
--) engine = Dictionary('dict_name');

-- Materialized View (example)
drop view if exists test_engines.mv;
create materialized view test_engines.mv to test_engines.mt as
    select id, name, value, created_at, status from test_engines.mt;

-- Example aggregation query for amt
-- select id, sumMerge(sum_value), avgMerge(avg_value), minMerge(min_value), maxMerge(max_value), countMerge(count_value) from test_engines.amt group by id;

-- Example insert for amt
-- insert into test_engines.amt select id, sumState(value), avgState(value), minState(value), maxState(value), countState() from test_engines.mt group by id;
