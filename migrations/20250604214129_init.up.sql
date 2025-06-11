begin;

create table sales (
    id serial primary key,
    created_at timestamptz not null,
    start_at timestamptz not null,
    end_at timestamptz not null
);

create table items (
    id serial primary key,
    name text not null,
    created_at timestamptz not null,
    sale_id int not null references sales (id) on delete cascade,
    sale_start timestamptz not null,
    sale_end timestamptz not null,
    sold boolean not null default false,
    reserved_until timestamptz,
    reserved_by int, -- user_id
    code text
);

create table checkouts (
    created_at timestamptz not null,
    user_id int not null,
    item_id int not null references items (id) on delete cascade,
    code text,
    error text
);

commit;
