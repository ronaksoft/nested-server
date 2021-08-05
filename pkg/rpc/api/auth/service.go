package nestedServiceAuth

import (
	"git.ronaksoft.com/nested/server/nested"
	"git.ronaksoft.com/nested/server/pkg/rpc"
	"git.ronaksoft.com/nested/server/pkg/rpc/api"
)

const (
	ServicePrefix = "auth"
)
const (
	CmdGetVerificationCode      = "auth/get_verification"
	CmdGetEmailVerificationCode = "auth/get_email_verification"
	CmdVerifyCode               = "auth/verify_code"
	CmdSendCodeSms              = "auth/send_text"
	CmdRegisterUser             = "auth/register_user"
	CmdRecoverPassword          = "auth/recover_pass"
	CmdRecoverUsername          = "auth/recover_username"
	CmdPhoneAvailable           = "auth/phone_available"
)

type AuthService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewAuthService(worker *api.Worker) *AuthService {
	s := new(AuthService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CmdGetVerificationCode:      {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getPhoneVerificationCode},
		CmdGetEmailVerificationCode: {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.getEmailVerificationCode},
		CmdVerifyCode:               {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.verifyCode},
		CmdSendCodeSms:              {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.sendCodeByText},
		CmdRecoverPassword:          {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.recoverPassword},
		CmdRecoverUsername:          {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.recoverUsername},
		CmdPhoneAvailable:           {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.phoneAvailable},
		CmdRegisterUser:             {MinAuthLevel: api.AuthLevelUnauthorized, Execute: s.registerUserAccount},
	}

	return s
}

func (s *AuthService) GetServicePrefix() string {
	return ServicePrefix
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
