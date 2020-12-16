#!/bin/sh


if [ -f data/enrollment.dump ]; then
    pg_restore \
        -Fc -j $(nproc) \
        --data-only --table=enrollment \
        -h localhost -d $POSTGRES_DB -U $POSTGRES_USER \
        data/enrollment.dump
fi
