#!/usr/bin/env bash

govendor add +e
govendor update +v
govendor remove +u
govendor fetch +m