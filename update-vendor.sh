#!/usr/bin/env bash

govendor add +e
govendor update +v
govendor fetch +m
govendor remove +u