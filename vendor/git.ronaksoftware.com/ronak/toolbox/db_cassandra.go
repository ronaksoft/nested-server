package ronak

import (
    "github.com/gocql/gocql"
    "time"
    "fmt"
    "bytes"
    "text/template"
    "github.com/scylladb/gocqlx"
    "go.uber.org/zap"
)

/*
    Creation Time: 2018 - Apr - 07
    Created by:  Ehsan N. Moosa (ehsan)
    Maintainers:
        1.  Ehsan N. Moosa (ehsan)
    Auditor: Ehsan N. Moosa
    Copyright Ronak Software Group 2018
*/

const (
    CQL_VERSION = "3.4.4"
)

var (
    tpCreateTable = `CREATE TABLE IF NOT EXISTS {{.TableName}} ( 
    {{range  .Columns}}{{.Name}} {{.Type}}, {{end}}
    PRIMARY KEY (
        {{range $idx, $elem := .PrimaryKeys}}{{if ne $idx 0}}, {{end}}{{$elem}}{{end}}
    ))`

    tpCreateTableWithClustering = `CREATE TABLE IF NOT EXISTS {{.TableName}} ( 
    {{range  .Columns}}{{.Name}} {{.Type}}, {{end}}
    PRIMARY KEY (
        ({{range $idx, $elem := .PrimaryKeys}}
            {{if ne $idx 0}}, {{end}}
            {{$elem}}
        {{end}}),
        {{range $idx, $elem := .ClusteringKeys}}
            {{if ne $idx 0}}, {{end}}
            {{$elem}}
        {{end}}
    ))
    {{$length := len .ClusteringColumns}}
    {{if ne $length 0}}
    WITH CLUSTERING ORDER BY (
    {{range $idx, $elem := .ClusteringColumns}}
    {{if ne $idx 0}}, {{end}}{{$elem.Name}} {{$elem.Order}}
    {{end}}
    )
    {{end}}`
)

// CqlCreateTable
type CqlCreateTable struct {
    TableName         string
    Columns           []CqlTableColumn
    PrimaryKeys       []string
    ClusteringKeys    []string
    ClusteringColumns []CqlTableClusteringColumn
}

// CqlTableColumn
type CqlTableColumn struct {
    Name string
    Type string
}

// CqlTableClusteringColumn
type CqlTableClusteringColumn struct {
    Name  string
    Order string // ASC or DESC
}

// CassDB
type CassDB struct {
    config  CassConfig
    session *gocql.Session
}

// CassConfig
type CassConfig struct {
    Host              string
    Username          string
    Password          string
    Keyspace          string
    Retries           int
    RetryMinBackOff   time.Duration
    RetryMaxBackOff   time.Duration
    ConnectTimeout    time.Duration
    Timeout           time.Duration
    ReconnectInterval time.Duration
    Concurrency       int
    Consistency       Consistency
    SerialConsistency SerialConsistency
}

type Consistency uint16
type SerialConsistency uint16

const (
    Any         Consistency       = 0x00
    One         Consistency       = 0x01
    Two         Consistency       = 0x02
    Three       Consistency       = 0x03
    Quorum      Consistency       = 0x04
    All         Consistency       = 0x05
    LocalQuorum Consistency       = 0x06
    EachQuorum  Consistency       = 0x07
    LocalOne    Consistency       = 0x0A
    Serial      SerialConsistency = 0x08
    LocalSerial SerialConsistency = 0x09
)

var (
    DefaultCassConfig = CassConfig{
        Concurrency:       5,
        Timeout:           10 * time.Second,
        ConnectTimeout:    10 * time.Second,
        Retries:           10,
        RetryMinBackOff:   10 * time.Millisecond,
        RetryMaxBackOff:   1 * time.Second,
        ReconnectInterval: 3 * time.Second,
        Consistency:       LocalQuorum,
        SerialConsistency: LocalSerial,
    }
)

// NewCassDB
// Returns CassDB struct which has a 'gocql' session object enclosed.
// You can use DefaultCassConfig for quick configuration but make sure to set
// Username, Password and KeySpace
//
// example :
//  conf := ronak.DefaultCassConfig
//  conf.Username = "username"
//  conf.Password = "password"
//  conf.KeySpace = "key-space"
//  db := NewCassDB(conf)
func NewCassDB(conf CassConfig) *CassDB {
    db := new(CassDB)
    db.config = conf
    cassCluster := gocql.NewCluster(conf.Host)
    retryPolicy := new(gocql.ExponentialBackoffRetryPolicy)
    retryPolicy.NumRetries = conf.Retries
    retryPolicy.Min = conf.RetryMinBackOff
    retryPolicy.Max = conf.RetryMaxBackOff

    cassCluster.RetryPolicy = retryPolicy
    cassCluster.ConnectTimeout = conf.ConnectTimeout
    cassCluster.Timeout = conf.Timeout
    cassCluster.ReconnectInterval = conf.ReconnectInterval

    cassCluster.Authenticator = gocql.PasswordAuthenticator{
        Username: conf.Username,
        Password: conf.Password,
    }

    cassCluster.NumConns = conf.Concurrency
    if len(conf.Keyspace) > 0 {
        cassCluster.Keyspace = conf.Keyspace
        cassCluster.Consistency = gocql.Consistency(conf.Consistency)
        cassCluster.SerialConsistency = gocql.SerialConsistency(conf.SerialConsistency)
        cassCluster.CQLVersion = CQL_VERSION
        if session, err := cassCluster.CreateSession(); err != nil {
            _LOG.Fatal(err.Error())
            return nil
        } else {
            db.session = session
        }
    }
    return db
}

func CreateKeySpace(conf CassConfig) error {
    cassCluster := gocql.NewCluster(conf.Host)
    cassCluster.Authenticator = gocql.PasswordAuthenticator{
        conf.Username, conf.Password,
    }
    cassCluster.CQLVersion = CQL_VERSION
    if session, err := cassCluster.CreateSession(); err != nil {
        return err
    } else {
        session.Query(
            fmt.Sprintf(
                "CREATE KEYSPACE IF NOT EXISTS %s WITH REPLICATION = {'class' : 'SimpleStrategy', 'replication_factor' : 1 }",
                conf.Keyspace,
            ),
        ).Exec()
        session.Close()
    }
    return nil
}

func DropKeySpace(conf CassConfig) error {
    cassCluster := gocql.NewCluster(conf.Host)
    cassCluster.Authenticator = gocql.PasswordAuthenticator{
        conf.Username, conf.Password,
    }
    cassCluster.CQLVersion = CQL_VERSION
    if session, err := cassCluster.CreateSession(); err != nil {
        _LOG.Fatal(err.Error())
        return err
    } else {
        session.Query(
            fmt.Sprintf(
                "DROP KEYSPACE IF EXISTS %s", conf.Keyspace,
            ),
        ).Exec()
        session.Close()
    }
    return nil
}

func (db *CassDB) SetConsistency(c Consistency) {
    db.session.SetConsistency(gocql.Consistency(c))
}

func (db *CassDB) GetSession() *gocql.Session {
    return db.session
}

func (db *CassDB) CloseSession() {
    db.session.Close()
}

func (db *CassDB) CreateTable(createTableQueries map[string]CqlCreateTable) error {
    for tableName, query := range createTableQueries {
        if len(query.ClusteringKeys) > 0 {
            t, err := template.New(tableName).Parse(tpCreateTableWithClustering)
            if err != nil {
                _LOG.Warn(err.Error(),
                    zap.String("TableName", tableName),
                )
                return err
            }
            buf := new(bytes.Buffer)
            t.Execute(buf, query)
            if err := db.session.Query(buf.String()).Exec(); err != nil {
                _LOG.Warn(err.Error(),
                    zap.String("TableName", tableName),
                )
                return err
            }
        } else {
            t, err := template.New(tableName).Parse(tpCreateTable)
            if err != nil {
                _LOG.Warn(err.Error(),
                    zap.String("TableName", tableName),
                )
                return err
            }
            buf := new(bytes.Buffer)
            t.Execute(buf, query)
            if err := db.session.Query(buf.String()).Exec(); err != nil {
                _LOG.Warn(err.Error(),
                    zap.String("TableName", tableName),
                )
                return err
            }
        }
        time.Sleep(500 * time.Millisecond)
    }
    return nil
}

// ExecuteRelease
// Make sure call this function when 'q' is an Insert or idempotent Update operation otherwise you will
// get panic or unpredictable results.
func (db *CassDB) ExecuteRelease(q *gocqlx.Queryx) error {
    defer q.Release()
    retries := 0
    sleepTime := 10
    for {
        retries++
        if err := q.Exec(); err != nil {
            switch err {
            case gocql.ErrNoHosts, gocql.ErrTimeoutNoResponse, gocql.ErrNoConnections, gocql.ErrConnectionClosed, gocql.ErrNoStreams:
                if retries > db.config.Retries {
                    return err
                }
                time.Sleep(time.Duration(sleepTime) * time.Millisecond)
                sleepTime *= 2
            default:
                return err
            }

        } else {
            if retries > 1 {
                _LOG.Debug("Successful after:",
                    zap.Int("Attempts", q.Attempts()),
                        zap.Int("Retries", retries),
                )
            }
            break
        }
    }
    return nil
}