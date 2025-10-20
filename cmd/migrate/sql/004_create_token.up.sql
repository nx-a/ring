create table if not exists public.token
(
    token_id   serial
        constraint token_pk
            primary key,
    bucket_id bigint not null,
    type integer  not null,
    value varchar(512) not null
);

create unique index if not exists token_value_uindex
    on public.token (value);