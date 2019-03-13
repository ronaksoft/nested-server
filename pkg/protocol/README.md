# Protocol

# Table of Contents
* [Request Stack](#request-stack)
* [Protocol Data Unit](#pdu)
  * [Packet](#packet)
  * [Generic Request Packet](#generic-request-packet)
  * [Stream Request Packet](#stream-request-packet)
  * [Push Packet](#push-packet)
  * [Generic Response Packet](#generic-response-packet)
  * [Stream Response Packet](#stream-response-packet)
* [Errors](#errors)
  * [Generic Error](#generic-error)
  * [Forbidden Error](#forbidden-error)
  * [Unavailable Error](#unavailable-error)
  * [Invalid Error](#invalid-error)
  * [Incomplete Error](#incomplete-error)
  * [Duplicate Error](#duplicate-error)
  * [Limit Error](#limit-error)

# <a name="request-stack"></a>Request Stack

|              | Client             |     |                    |     | Server             |            |
| ------------ | ------------------ | --- | ------------------ | --- | ------------------ | ---------- |
| Service User | **Request Data**   |     |                    |     | **Request Data**   | Model      |
| Client       | **Command**        |     |                    |     | **Command**        | Worker     |
| API Client   | **Api Key**        |     |                    |     | **Api Key**        | API worker |
| NATS         | **Subject**        |     | Internal Transport |     | **Subject**        | NATS       |
| Router       | **Subject Prefix** |     | External Transport |     | **Subject Prefix** | Router     |

# <a name="pdu"></a>Protocol Data Unit

## <a name="packet"></a>Packet
<pre>
 -------------------------- ------------------------
| - Address {NATS Subject} | - Type {Datagram Type} |
 -------------------------- ------------------------
 
                           |----Datagram------------|
|----Packet-----------------------------------------|
</pre>
```json
{
  "subject": "NATS Subject",
  "data": {
    "type": "Datagram Type"
  }
}
```

## <a name="generic-request-packet"></a>Generic Request Packet
<pre>
 ---------------------------------- ------------------------------ -------------------------
| - Address {NATS Request Subject} | - Type {Datagram Type = "q"} | - User Data {Arguments} |
|                                  | - Command {API Command}      |                         |
 ---------------------------------- ------------------------------ -------------------------

                                                                  |----Payload--------------|
                                   |----Request---------------------------------------------|
|----Packet---------------------------------------------------------------------------------|
</pre>
```json
{
  "subject": "NATS Request Subject",
  "data": {
    "type": "q",
    "cmd": "api-command-cat/action",
    "data": {}
  }
}
```

## <a name="stream-request-packet"></a>Stream Request Packet
<pre>
 ---------------------------------- ------------------------------ -------------------------
| - Address {NATS Request Subject} | - Type {Datagram Type = "q"} | - User Data {Arguments} |
|                                  | - Command {API Command}      |                         |
|                                  | - Request ID                 |                         |
 ---------------------------------- ------------------------------ -------------------------

                                                                  |----Payload--------------|
                                   |----Request---------------------------------------------|
|----Packet---------------------------------------------------------------------------------|
</pre>
```json
{
  "subject": "NATS Request Subject",
  "data": {
    "type": "q",
    "cmd": "api-command-cat/action",
    "_reqid": "Request ID",
    "data": {}
  }
}
```

## <a name="push-packet"></a>Push Packet
<pre>
 ---------------------------------- ------------------------------ -------------------------
| - Address {NATS Request Subject} | - Type {Datagram Type = "p"} | - User Data {Arguments} |
|                                  | - Command {API Command}      |                         |
 ---------------------------------- ------------------------------ -------------------------

                                                                  |----Payload--------------|
                                   |----Request---------------------------------------------|
|----Packet---------------------------------------------------------------------------------|
</pre>
```json
{
  "subject": "NATS Request Subject",
  "data": {
    "type": "p",
    "cmd": "api-command-cat/action",
    "data": {}
  }
}
```

## <a name="generic-response-packet"></a>Generic Response Packet
<pre>
 -------------------------------- ------------------------------ ------------------
| - Address {NATS Reply Subject} | - Type {Datagram Type = "r"} | - Data {Results} |
|                                | - Status {Response Status}   |                  |
 -------------------------------- ------------------------------ ------------------

                                                                |----Payload-------|
                                 |----Response-------------------------------------|
|----Packet------------------------------------------------------------------------|
</pre>
```json
{
  "subject": "NATS Reply Subject",
  "data": {
    "type": "r",
    "status": "ok|err",
    "data": {}
  }
}
```

## <a name="stream-response-packet"></a>Stream Response Packet
<pre>
 -------------------------------- ------------------------------ ------------------
| - Address {NATS Reply Subject} | - Type {Datagram Type = "r"} | - Data {Results} |
|                                | - Status {Response Status}   |                  |
|                                | - Request ID                 |                  |
 -------------------------------- ------------------------------ ------------------

                                                                |----Payload-------|
                                 |----Response-------------------------------------|
|----Packet------------------------------------------------------------------------|
</pre>
```json
{
  "subject": "NATS Reply Subject",
  "data": {
    "type": "r",
    "status": "ok|err",
    "_reqid": "Request ID",
    "data": {}
  }
}
```

# <a name="errors"></a>Errors
In case of error occurred while processing the request, Response's data would be an instance of Error:
```go
type Error interface {
  Payload

  Code() ErrorCode
  Data() Payload
}
```

## <a name="generic-error"></a>Generic Error
Code: ```ERROR_GENERIC```

Response.Data():
```json
{
  "code": 0,
  "data": {}
}
```

## <a name="forbidden-error"></a>Forbidden Error
Code: ```ERROR_FORBIDDEN```

Response.Data():
```json
{
  "code": 1,
  "data": {}
}
```

## <a name="unavailable-error"></a>Unavailable Error
Code: ```ERROR_UNAVAILABLE```

Response.Data():
```json
{
  "code": 2,
  "data": {
    "items": []
  }
}
```

## <a name="invalid-error"></a>Invalid Error
Code: ```ERROR_INVALID```

Response.Data():
```json
{
  "code": 3,
  "data": {
    "items": []
  }
}
```

## <a name="incomplete-error"></a>Incomplete Error
Code: ```ERROR_INCOMPLETE```

Response.Data():
```json
{
  "code": 4,
  "data": {}
}
```

## <a name="duplicate-error"></a>Duplicate Error
Code: ```ERROR_DUPLICATE```

Response.Data():
```json
{
  "code": 5,
  "data": {
    "items": []
  }
}
```

## <a name="limit-error"></a>Limit Error
Code: ```ERROR_LIMIT```

Response.Data():
```json
{
  "code": 6,
  "data": {
    "items": []
  }
}
```
