package session

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

// func FCM(d Device, req CMDPushExternal) {
// 	// TODO:: Handle FCM
// 	message := messaging.Message{
// 		Data:  req.Data,
// 		Token: d.Token,
// 		Android: &messaging.AndroidConfig{
// 			Priority: "high",
// 			Data:     req.Data,
// 		},
// 		APNS: &messaging.APNSConfig{
// 			Payload: &messaging.APNSPayload{
// 				Aps: &messaging.Aps{
// 					Alert: &messaging.ApsAlert{
// 						Title: req.Data["title"],
// 						Body:  req.Data["msg"],
// 					},
// 					Badge:      &d.Badge,
// 					CustomData: make(map[string]interface{}),
// 				},
// 			},
// 		},
// 	}
// 	for k, v := range req.Data {
// 		message.APNS.Payload.Aps.CustomData[k] = v
// 	}
//
// 	// ctx := context.Background()
// 	// if client, err := _FCM.Messaging(ctx); err != nil {
// 	// 	log.Warn(err.Error())
// 	// } else if _, err := client.Send(ctx, &message); err != nil {
// 	// 	log.Warn(err.Error())
// 	// 	_Model.Device.Remove(d.ID)
// 	// }
// }
