#!/bin/sh

# wait for postgres
while ! nc -z db 5432; do
    sleep 0.1
done

# build
go build -o /usr/bin/picovpn

# proceed to docker command
exec "$@"