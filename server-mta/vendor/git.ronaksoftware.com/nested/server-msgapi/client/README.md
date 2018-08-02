# Msgapi Client

# Protocol

## <a name="request-packet"></a>Request Packet
<pre>
 ------------------------------------------------ ------------------------------ -------------------------
| - Address {NATS Request Subject = "MSGAPI.V1"} | - Type {Datagram Type = "q"} | - User Data {Arguments} |
|                                                | - Command {API Command}      |                         |
 ------------------------------------------------ ------------------------------ -------------------------

                                                                                |----Payload--------------|
                                                 |----Request---------------------------------------------|
|----Packet-----------------------------------------------------------------------------------------------|
</pre>
```json
{
  "subject": "MSGAPI.V1",
  "data": {
    "type": "q",
    "cmd": "api-command-cat/action",
    "data": {}
  }
}
```
