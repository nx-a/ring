create table if not exists public.point
(
    point_id  serial
        constraint point_pk
            primary key,
    bucket_id bigint not null,
    external_id  varchar(128),
    time_zone varchar(128)
);

create index if not exists point_bucket_index
    on public.point (bucket_id, external_id);