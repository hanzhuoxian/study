#!/bin/bash

set -e

echo `service mysql status`

mysql < /mysql/database.sql

