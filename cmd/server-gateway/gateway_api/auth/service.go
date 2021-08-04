package nestedServiceAuth

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
)

const (
	SERVICE_PREFIX = "auth"
)
const (
	CMD_GET_VERIFICATION_CODE       = "auth/get_verification"
	CMD_GET_EMAIL_VERIFICATION_CODE = "auth/get_email_verification"
	CMD_VERIFY_CODE                 = "auth/verify_code"
	CMD_SEND_CODE_SMS               = "auth/send_text"
	CMD_REGISTER_USER               = "auth/register_user"
	CMD_RECOVER_PASSWORD            = "auth/recover_pass"
	CMD_RECOVER_USERNAME            = "auth/recover_username"
	CMD_PHONE_AVAILABLE             = "auth/phone_available"
)

type AuthService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAuthService(worker *api.Worker) *AuthService {
	s := new(AuthService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_GET_VERIFICATION_CODE:       {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getPhoneVerificationCode},
		CMD_GET_EMAIL_VERIFICATION_CODE: {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getEmailVerificationCode},
		CMD_VERIFY_CODE:                 {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.verifyCode},
		CMD_SEND_CODE_SMS:               {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.sendCodeByText},
		CMD_RECOVER_PASSWORD:            {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.recoverPassword},
		CMD_RECOVER_USERNAME:            {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.recoverUsername},
		CMD_PHONE_AVAILABLE:             {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.phoneAvailable},
		CMD_REGISTER_USER:               {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.registerUserAccount},
	}

	return s
}

func (s *AuthService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *AuthService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *rpc.Request, response *rpc.Response) {
	commandName := request.Command
	if cmd, ok := s.serviceCommands[commandName]; ok {
		if authLevel >= cmd.MinAuthLevel {
			cmd.Execute(requester, request, response)
		} else {
			response.NotAuthorized()
		}
	} else {
		response.NotImplemented()
	}
}

func (s *AuthService) Worker() *api.Worker {
	return s.worker
}
