create table if not exists public.control
(
    control_id   serial
        constraint control_pk
            primary key,
    login varchar(128) not null,
    password   varchar(128)
);

create unique index if not exists control_login_uindex
    on public.control (login);