create table data (
id uuid default gen_random_uuid(),
date timestamp default current_timestamp,
data jsonb not null,

primary key (id)
);

create index data_date on data (date);