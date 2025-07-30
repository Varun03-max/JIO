package handlers

// LoginRequestBodyData represents Request body for password based login request
type LoginRequestBodyData struct {
	Username string `json:"username"` // Simplified
	Password string `json:"password"`
}

// LoginSendOTPRequestBodyData represents Request body for OTP based login request
type LoginSendOTPRequestBodyData struct {
	MobileNumber string `json:"mobileNumber"` // ✅ Fix this tag
}

// LoginVerifyOTPRequestBodyData  represents Request body for OTP verification request
type LoginVerifyOTPRequestBodyData struct {
	MobileNumber string `json:"mobileNumber"` // ✅ Fix this tag
	OTP          string `json:"otp"`
}

type RefreshTokenResponse struct {
	AccessToken string `json:"authToken"`
}

type RefreshSSOTokenResponse struct {
	SSOToken string `json:"ssoToken"`
}

type DrmMpdOutput struct {
	LicenseUrl  string
	PlayUrl     string
	Tv_url_host string
	Tv_url_path string
}
