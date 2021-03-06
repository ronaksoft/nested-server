package nestedServiceSearch

import (
	"git.ronaksoft.com/nested/server/cmd/server-gateway/client"
	"git.ronaksoft.com/nested/server/cmd/server-gateway/gateway_api"
	"git.ronaksoft.com/nested/server/model"
)

const (
	SERVICE_PREFIX string = "search"
)
const (
	CMD_PLACES_FOR_COMPOSE        = "search/places_for_compose"
	CMD_PLACES_FOR_SEARCH         = "search/places_for_search"
	CMD_ACCOUNTS_FOR_INVITE       = "search/accounts_for_invite"
	CMD_ACCOUNTS_FOR_ADD          = "search/accounts_for_add"
	CMD_ACCOUNTS_FOR_MENTION      = "search/accounts_for_mention"
	CMD_ACCOUNTS_FOR_TASK_MENTION = "search/accounts_for_task_mention"
	CMD_ACCOUNTS_FOR_SEARCH       = "search/accounts_for_search"
	CMD_ACCOUNTS_FOR_ADMIN        = "search/accounts_for_admin"
	CMD_ACCOUNTS                  = "search/accounts"
	CMD_SUGGESTIONS               = "search/suggestions"
	CMD_LABELS                    = "search/labels"
	CMD_POSTS                     = "search/posts"
	CMD_POSTS_CONVERSATION        = "search/posts_conversation"
	CMD_TASKS                     = "search/tasks"
	CMD_APPS                      = "search/apps"
)

type SearchService struct {
	worker          *api.Worker
	serviceCommands api.ServiceCommands
}

func NewSearchService(worker *api.Worker) *SearchService {
	s := new(SearchService)
	s.worker = worker

	s.serviceCommands = api.ServiceCommands{
		CMD_ACCOUNTS:                  {MinAuthLevel: api.AUTH_LEVEL_APP_L1,Execute: s.accounts},
		CMD_ACCOUNTS_FOR_ADMIN:        {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.accountsForAdmin},
		CMD_ACCOUNTS_FOR_SEARCH:       {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.accountsForSearch},
		CMD_ACCOUNTS_FOR_ADD:          {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.accountsForAdd},
		CMD_ACCOUNTS_FOR_INVITE:       {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.accountsForInvite},
		CMD_ACCOUNTS_FOR_MENTION:      {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.accountsForMention},
		CMD_ACCOUNTS_FOR_TASK_MENTION: {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.accountsForTaskMention},
		CMD_LABELS:                    {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.labels},
		CMD_PLACES_FOR_COMPOSE:        {MinAuthLevel: api.AUTH_LEVEL_APP_L1,Execute: s.placesForCompose},
		CMD_PLACES_FOR_SEARCH:         {MinAuthLevel: api.AUTH_LEVEL_APP_L1,Execute: s.placesForSearch},
		CMD_POSTS:                     {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.posts},
		CMD_POSTS_CONVERSATION:        {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.conversation},
		CMD_SUGGESTIONS:               {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.suggestions},
		CMD_TASKS:                     {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.tasks},
		CMD_APPS:                      {MinAuthLevel: api.AUTH_LEVEL_USER,Execute: s.apps},
	}

	return s
}

func (s *SearchService) GetServicePrefix() string {
	return SERVICE_PREFIX
}

func (s *SearchService) ExecuteCommand(authLevel api.AuthLevel, requester *nested.Account, request *nestedGateway.Request, response *nestedGateway.Response) {
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

func (s *SearchService) Worker() *api.Worker {
	return s.worker
}
