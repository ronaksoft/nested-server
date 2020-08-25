## Pre Face
This project is a merge of the following projects:
1. server-model-nested
2. server-gateway
3. server-ntfy
4. server-mta


Since these 4 projects are closely coupled together we moved all in one project. However in case of need
access to Model we can create a mirror project of the 'model' to be accessible by external developers.


## Docker Images
This project will create 3 docker image in `registry.ronaksoft.com`:
1. registry.ronaksoft.com/nested/server/gateway:4.0
2. registry.ronaksoft.com/nested/server/ntfy:3.0
3. registry.ronaksoft.com/nested/server/mta:2.0

