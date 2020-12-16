#!/bin/sh

set -e

LOG=data/docker-postgres-init.log
DBFILE="data/mercedtime.dump"
ENRLT="data/spring-2021/enrollment.dump"

if [ -f $DBFILE ]; then
    pg_restore \
        -Fc -j $(nproc) --data-only \
        -d $POSTGRES_DB -U $POSTGRES_USER \
        $DBFILE
    echo "[$(date)] loaded data from backup file"
else
    psql -d $POSTGRES_DB -U $POSTGRES_USER -c '\i data/load.sql'
    echo "[$(date)] loaded csv files"

    if [ -f $ENRLT ]; then
        echo "[$(date)] loading enrollment table backup"
        pg_restore \
            -Fc -j $(nproc) \
            --data-only --table=enrollment \
            -d $POSTGRES_DB -U $POSTGRES_USER \
            $ENRLT
    else
        echo "[$(date)] enrollment data dump not found from $(pwd)"
    fi
fi
