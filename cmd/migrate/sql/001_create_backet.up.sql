create table if not exists public.bucket
(
    bucket_id   serial
        constraint bucket_pk
            primary key,
    control_id  bigint       not null,
    system_name varchar(128) not null,
    time_life   integer      not null,
    time_zone   varchar(128)
);

create index if not exists bucket_system_name_index
    on public.bucket (system_name);