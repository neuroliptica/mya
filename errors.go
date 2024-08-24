package main

type Error struct {
	// Error message.
	E string `json:"error"`
	// ErrorCode uint `json:"error_code"`
}

// error interface impl.
func (e Error) Error() string {
	return e.E
}

var (
	// General errors.
	ErrorBadRequest = Error{"oops! bad request"}
	ErrorInternal   = Error{"oops! internal server error, check logs"}

	// Databse errors.
	ErrorCreateFailed = Error{"failed to create record in db, check logs"}
	ErrorGetFailed    = Error{"failed to get record from db, check logs"}

	// Board API errors.
	ErrorEmptyLink = Error{"link field is empty"}
	ErrorEmptyName = Error{"name field is empty"}

	// Captcha API errors.
	ErrorNewCaptcha       = Error{"failed to generate new captcha, check logs"}
	ErrorNoCaptchaId      = Error{"no captcha id provided"}
	ErrorInvalidCaptchaId = Error{"invalid id or captcha expired"}

	// Posting API errors.
	// Bans and other dynamic errors isn't hardcoded here.
	ErrorBumpFailed     = Error{"failed to bump parent thread, check logs"}
	ErrorEmptySubject   = Error{"empty subject for thread"}
	ErrorNameTooLong    = Error{"name is too long"}
	ErrorInvalidCaptcha = Error{"captcha is invalid or has expired"}

	// todo(zvezdochka): HTML pages errors.
)
