## <a name="router">Router [![build status](http://git.ronaksoft.com/nested/server-router/badges/master/build.svg)](http://git.ronaksoft.com/nested/server-router/commits/master)
Every Nested *Bundle* needs a router in order to have ability to communicate to other bundles; In fact each internal service/client can send/receive requests to/from other *Bundles*' client/services.

**Note:** *Bundle* is the virtual environment which one or more services are running together and have direct access to each other.

## <a name="usage">Usage
### <a name="as-a-docker-container">As a docker container
```sh
docker pull registry.ronaksoft.com/nested/server-router
docker run --env NST_BUNDLE_ID=GOBRYAS-0001 registry.ronaksoft.com/nested/server-router
```

## <a name="configuration">Configuration
### <a name="available-configurations"></a>Available Configurations

| Parameter | Description |
|-----------|-------------|
| `BUNDLE_ID` | *Bundle*'s unique identifier in form of <BUNDLE GROUP>-<BUNDLE INDEX> (=ROUTING-001) |
| `JOB_INT_ADDRESS` | *Bundle*'s internal NATS server address (=nats://ronak:P0uyan@intern.job.nst:4222) |
| `JOB_EXT_ADDRESS` | External NATS server address (=nats://ronak:P0uyan@xntern.job.nst:4222) |

### <a name="setup-configurations-using-environment-variable">Setup Configurations Using Environment Variable
Set parameters introduced [here](#available-configurations) in ENV which application is being run on preceded by **NST_** prefix. For instance: NST_BUNDLE_ID=CYRUS-0001

### <a name="setup-configurations-using-toml-file">Setup Configurations Using .toml File
