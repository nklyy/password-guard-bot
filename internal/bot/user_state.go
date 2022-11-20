package bot

type UserState struct {
	State    string
	From     string
	Pin      string
	Login    string
	Password string
}

func (u *UserState) UpdateState(state string) {
	u.State = state
}

func (u *UserState) UpdateFrom(from string) {
	u.From = from
}

func (u *UserState) UpdatePin(pin string) {
	u.Pin = pin
}

func (u *UserState) UpdateLogin(login string) {
	u.Login = login
}

func (u *UserState) UpdatePassword(password string) {
	u.Password = password
}

func (u *UserState) Refresh() {
	u.State = ""
	u.From = ""
	u.Pin = ""
	u.Login = ""
	u.Password = ""
}
