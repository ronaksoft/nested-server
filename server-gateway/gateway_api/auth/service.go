package nestedServiceAuth

import (
	"git.ronaksoftware.com/nested/server/server-gateway/client"
	"git.ronaksoftware.com/nested/server/server-gateway/gateway_api"
	"git.ronaksoftware.com/nested/server/model"
)

const (
	SERVICE_PREFIX  = "auth"
)
const (
	CMD_GET_VERIFICATION_CODE        = "auth/get_verification"
	CMD_GET_EMAIL_VERIFICATION_CODE  = "auth/get_email_verification"
	CMD_VERIFY_CODE                  = "auth/verify_code"
	CMD_SEND_CODE_SMS                = "auth/send_text"
	CMD_SEND_CODE_PHONE              = "auth/call_phone"
	CMD_SEND_CODE_EMAIL              = "auth/send_email"
	CMD_REGISTER_USER                = "auth/register_user"
	CMD_RECOVER_PASSWORD             = "auth/recover_pass"
	CMD_RECOVER_USERNAME             = "auth/recover_username"
	CMD_PHONE_AVAILABLE              = "auth/phone_available"
)

type AuthService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAuthService(worker *api.Worker) *AuthService {
	s := new(AuthService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_GET_VERIFICATION_CODE:       {api.AUTH_LEVEL_UNAUTHORIZED, s.getPhoneVerificationCode},
		CMD_GET_EMAIL_VERIFICATION_CODE: {api.AUTH_LEVEL_UNAUTHORIZED, s.getEmailVerificationCode},
		CMD_VERIFY_CODE:                 {api.AUTH_LEVEL_UNAUTHORIZED, s.verifyCode},
		CMD_SEND_CODE_EMAIL:             {api.AUTH_LEVEL_UNAUTHORIZED, s.sendCodeByEmail},
		CMD_SEND_CODE_SMS:               {api.AUTH_LEVEL_UNAUTHORIZED, s.sendCodeByText},
		CMD_SEND_CODE_PHONE:             {api.AUTH_LEVEL_UNAUTHORIZED, s.sendCodeByPhoneCall},
		CMD_RECOVER_PASSWORD:            {api.AUTH_LEVEL_UNAUTHORIZED, s.recoverPassword},
		CMD_RECOVER_USERNAME:            {api.AUTH_LEVEL_UNAUTHORIZED, s.recoverUsername},
		CMD_PHONE_AVAILABLE:             {api.AUTH_LEVEL_UNAUTHORIZED, s.phoneAvailable},
		CMD_REGISTER_USER:               {api.AUTH_LEVEL_UNAUTHORIZED, s.registerUserAccount},
	}

	return s
}

func (s *AuthService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *AuthService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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
