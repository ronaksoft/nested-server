package nestedServiceClient

import (
    "git.ronaksoftware.com/nested/server/model"
    "encoding/json"
    "strings"
    "log"
    "git.ronaksoftware.com/nested/server/server-gateway/client"
)

type ClientSettings struct {
    ClientID    string     `json:"_cid" bson:"_cid"`
    Language    string     `json:"lang" bson:"lang"`
    PlaceOrders PlaceOrder `json:"places_order" bson:"places_order"`
}

type ClientContacts struct {
    AccountID string                  `json:"-"`
    Contacts  []nested.AccountContact `json:"contacts" bson:"contacts"`
}

type PlaceOrder map[string]int

// @Command:	client/get_server_details
func (s *ClientService) getServerDetails(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    r := nested.M{
        "cyrus_id":         s.Worker().Config().GetString("BUNDLE_ID"),
        "server_timestamp": nested.Timestamp(),
    }
    response.OkWithData(r)

}

// @Command:	client/upload_contacts
// @Input:	contacts		string 	*	(json)
func (s *ClientService) uploadContacts(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    contacts := new(ClientContacts)
    if v, ok := request.Data["contacts"].(string); ok {
        if err := json.Unmarshal([]byte(v), contacts); err != nil {
            response.Error(nested.ERR_INVALID, []string{"contacts"})
            return
        }
        // fix the phone numbers
        for _, c := range contacts.Contacts {
            c.AccountID = requester.ID
            for i, p := range c.Phones {
                c.Phones[i] = strings.TrimLeft(p, "+0 ")
                if c.Phones[i] != "" {
                    _Model.Phone.AddContactToPhone(requester.ID, c.Phones[i])
                }
            }
            //_Model.Phone.SaveContact(c)
        }
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"contacts"})
        return
    }

    response.Ok()
}

// @Command:	client/save_key
// @Input:	key_name		string		*
// @Input:	key_value	string		*
func (s *ClientService) saveKey(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var keyName, keyValue string
    if v, ok := request.Data["key_name"].(string); ok {
        keyName = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"key_name"})
        return
    }
    if v, ok := request.Data["key_value"].(string); ok {
        if len(keyValue) > nested.DEFAULT_MAX_CLIENT_OBJ_SIZE {
            response.Error(nested.ERR_LIMIT, []string{"key_value"})
            return
        }
        keyValue = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"key_value"})
        return
    }
    if requester.Counters.Keys >= requester.Limits.Keys {
        response.Error(nested.ERR_LIMIT, []string{"keys"})
        return
    }
    if _Model.Account.SaveKey(requester.ID, keyName, keyValue) {
        response.Ok()
    } else {
        response.Error(nested.ERR_UNKNOWN, []string{})
    }
}

// @Command:	client/read_key
// @Input:	key_name		string		*
func (s *ClientService) getKey(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var keyName string
    if v, ok := request.Data["key_name"].(string); ok {
        keyName = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"key_name"})
        return
    }
    keyValue := _Model.Account.GetKey(requester.ID, keyName)
    response.OkWithData(nested.M{"key_value": keyValue})
}

// @Command:	client/get_all_keys
func (s *ClientService) getAllKeys(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    keys := s.Worker().Model().Account.GetAllKeys(requester.ID)
    keyNames := make([]string, 0, len(keys))
    log.Println(keys)
    for _, m := range keys {
        key := strings.SplitN(m["_id"], ".", 2)
        if len(key) > 1 {
            keyNames = append(keyNames, key[1])
        }

    }
    response.OkWithData(nested.M{"keys": keyNames})
}

// @Command:	client/remove_key
// @Input:	key_name		string		*
func (s *ClientService) removeKey(requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
    var keyName string
    if v, ok := request.Data["key_name"].(string); ok {
        keyName = v
    } else {
        response.Error(nested.ERR_INCOMPLETE, []string{"key_name"})
        return
    }
    _Model.Account.RemoveKey(requester.ID, keyName)
    response.Ok()
}
