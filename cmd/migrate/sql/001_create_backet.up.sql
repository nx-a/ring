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

create unique index if not exists bucket_system_name_uindex
    on public.bucket (system_name);