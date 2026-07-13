create table if not exists data
(
    data_id   char(36)     not null,
    bucket_id bigint       not null,
    point_id  bigint,
    time      timestamp    not null,
    level     varchar(10),
    val       jsonb,
    primary key (data_id, bucket_id)
) partition by list (bucket_id);

create index if not exists data_time_point_bucket_index
    on data (time, point_id, bucket_id);
