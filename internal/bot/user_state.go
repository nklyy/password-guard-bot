package bot

type UserState struct {
	State    string
	Page     int
	From     string
	Pin      string
	Login    string
	Password string
}

func (u *UserState) UpdateState(state string) {
	u.State = state
}

func (u *UserState) IncPage() {
	u.Page += 1
}

func (u *UserState) DecPage() {
	u.Page -= 1
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
	u.Page = 1
	u.From = ""
	u.Pin = ""
	u.Login = ""
	u.Password = ""
}
