package common

import (
	"fmt"
	"strings"

	M "github.com/lumavpn/luma/metadata"
	R "github.com/lumavpn/luma/rule"
)

type InUser struct {
	*Base
	users   []string
	adapter string
	payload string
}

func (u *InUser) Match(metadata *M.Metadata) (bool, string) {
	for _, user := range u.users {
		if metadata.InUser == user {
			return true, u.adapter
		}
	}
	return false, ""
}

func (u *InUser) RuleType() R.RuleType {
	return R.InUser
}

func (u *InUser) Adapter() string {
	return u.adapter
}

func (u *InUser) Payload() string {
	return u.payload
}

func NewInUser(iUsers, adapter string) (*InUser, error) {
	users := strings.Split(iUsers, "/")
	if len(users) == 0 {
		return nil, fmt.Errorf("in user couldn't be empty")
	}

	return &InUser{
		Base:    &Base{},
		users:   users,
		adapter: adapter,
		payload: iUsers,
	}, nil
}
