package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"

	"github.com/Varun03-max/JIO/pkg/scheduler"
	"github.com/Varun03-max/JIO/pkg/television"
	"github.com/Varun03-max/JIO/pkg/utils"
)

const (
	REFRESH_TOKEN_TASK_ID    = "jiotv_refresh_token"
	REFRESH_SSOTOKEN_TASK_ID = "jiotv_refresh_sso_token"
)

// Structs for request binding
type LoginSendOTPRequestBodyData struct {
	MobileNumber string `json:"mobileNumber"`
}

type LoginVerifyOTPRequestBodyData struct {
	MobileNumber string `json:"mobileNumber"`
	OTP          string `json:"otp"`
}

type LoginRequestBodyData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Utility validation
func checkFieldExist(name string, ok bool, c *fiber.Ctx) {
	if !ok {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": fmt.Sprintf("%s is required", name),
		})
	}
}

// LoginSendOTPHandler sends OTP to user
func LoginSendOTPHandler(c *fiber.Ctx) error {
	form := new(LoginSendOTPRequestBodyData)
	if err := c.BodyParser(form); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
	}
	if form.MobileNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mobile number is required"})
	}

	result, err := utils.LoginSendOTP(form.MobileNumber)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}

	return c.JSON(fiber.Map{"status": result})
}

// LoginVerifyOTPHandler logs in using OTP
func LoginVerifyOTPHandler(c *fiber.Ctx) error {
	form := new(LoginVerifyOTPRequestBodyData)
	if err := c.BodyParser(form); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
	}
	if form.MobileNumber == "" || form.OTP == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Mobile number and OTP are required"})
	}

	result, err := utils.LoginVerifyOTP(form.MobileNumber, form.OTP)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
	}

	Init() // re-initialize session, store, tokens etc.
	return c.JSON(result)
}

// LoginPasswordHandler logs in using mobile + password
func LoginPasswordHandler(c *fiber.Ctx) error {
	form := new(LoginRequestBodyData)
	if err := c.BodyParser(form); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
	}
	if form.Username == "" || form.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Username and Password required"})
	}

	result, err := utils.Login(form.Username, form.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
	}

	Init()
	return c.JSON(result)
}

// LogoutHandler destroys session and clears credentials
func LogoutHandler(c *fiber.Ctx) error {
	if err := utils.Logout(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Failed to logout"})
	}
	Init()
	return c.Redirect("/", fiber.StatusFound)
}

// LoginRefreshAccessToken refreshes auth access token
func LoginRefreshAccessToken() error {
	utils.Log.Println("Refreshing AccessToken...")
	tokenData, err := utils.GetJIOTVCredentials()
	if err != nil {
		return err
	}

	reqBody := map[string]string{
		"appName":      "RJIL_JioTV",
		"deviceId":     utils.GetDeviceID(),
		"refreshToken": tokenData.RefreshToken,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(REFRESH_TOKEN_URL)
	req.Header.SetMethod("POST")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("accessToken", tokenData.AccessToken)
	req.SetBody(jsonBody)

	client := utils.GetRequestClient()
	if err := client.Do(req, resp); err != nil {
		return err
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("AccessToken refresh failed with status: %d", resp.StatusCode())
	}

	var response RefreshTokenResponse
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return err
	}

	if response.AccessToken != "" {
		tokenData.AccessToken = response.AccessToken
		tokenData.LastTokenRefreshTime = strconv.FormatInt(time.Now().Unix(), 10)
		_ = utils.WriteJIOTVCredentials(tokenData)
		TV = television.New(tokenData)
		go RefreshTokenIfExpired(tokenData)
	}
	return nil
}

// LoginRefreshSSOToken refreshes the SSO token
func LoginRefreshSSOToken() error {
	utils.Log.Println("Refreshing SSOToken...")
	tokenData, err := utils.GetJIOTVCredentials()
	if err != nil {
		return err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(REFRESH_SSO_TOKEN_URL)
	req.Header.SetMethod("GET")
	req.Header.Set("ssoToken", tokenData.SSOToken)
	req.Header.Set("deviceid", utils.GetDeviceID())

	client := utils.GetRequestClient()
	if err := client.Do(req, resp); err != nil {
		return err
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("SSOToken refresh failed with status: %d", resp.StatusCode())
	}

	var response RefreshSSOTokenResponse
	if err := json.Unmarshal(resp.Body(), &response); err != nil {
		return err
	}

	if response.SSOToken != "" {
		tokenData.SSOToken = response.SSOToken
		tokenData.LastSSOTokenRefreshTime = strconv.FormatInt(time.Now().Unix(), 10)
		_ = utils.WriteJIOTVCredentials(tokenData)
		TV = television.New(tokenData)
		go RefreshSSOTokenIfExpired(tokenData)
	}
	return nil
}

// RefreshTokenIfExpired handles refresh scheduling of AccessToken
func RefreshTokenIfExpired(token *utils.JIOTV_CREDENTIALS) error {
	last, _ := strconv.ParseInt(token.LastTokenRefreshTime, 10, 64)
	next := time.Unix(last, 0).Add(1*time.Hour + 50*time.Minute)
	if time.Now().After(next) {
		return LoginRefreshAccessToken()
	}
	return scheduler.Add(REFRESH_TOKEN_TASK_ID, time.Until(next), func() error {
		return RefreshTokenIfExpired(token)
	})
}

// RefreshSSOTokenIfExpired handles refresh scheduling of SSOToken
func RefreshSSOTokenIfExpired(token *utils.JIOTV_CREDENTIALS) error {
	last, _ := strconv.ParseInt(token.LastSSOTokenRefreshTime, 10, 64)
	next := time.Unix(last, 0).Add(24 * time.Hour)
	if time.Now().After(next) {
		return LoginRefreshSSOToken()
	}
	return scheduler.Add(REFRESH_SSOTOKEN_TASK_ID, time.Until(next), func() error {
		return RefreshSSOTokenIfExpired(token)
	})
}
