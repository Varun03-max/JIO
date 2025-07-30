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

// LoginSendOTPHandler sends OTP for login
func LoginSendOTPHandler(c *fiber.Ctx) error {
	formBody := new(LoginSendOTPRequestBodyData)
	if err := c.BodyParser(formBody); err != nil {
		utils.Log.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
	}
	if err := checkFieldExist("Mobile Number", formBody.MobileNumber != "", c); err != nil {
		return err
	}

	result, err := utils.LoginSendOTP(formBody.MobileNumber)
	if err != nil {
		utils.Log.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": err.Error()})
	}
	return c.JSON(fiber.Map{"status": result})
}

// LoginVerifyOTPHandler verifies OTP and login
func LoginVerifyOTPHandler(c *fiber.Ctx) error {
	formBody := new(LoginVerifyOTPRequestBodyData)
	if err := c.BodyParser(formBody); err != nil {
		utils.Log.Println(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
	}
	if err := checkFieldExist("Mobile Number", formBody.MobileNumber != "", c); err != nil {
		return err
	}
	if err := checkFieldExist("OTP", formBody.OTP != "", c); err != nil {
		return err
	}

	result, err := utils.LoginVerifyOTP(formBody.MobileNumber, formBody.OTP)
	if err != nil {
		utils.Log.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
	}
	Init()
	return c.JSON(result)
}

// LoginPasswordHandler is used to login with password
func LoginPasswordHandler(c *fiber.Ctx) error {
	var username, password string

	if c.Method() == fiber.MethodGet {
		username = c.Query("username")
		password = c.Query("password")
	} else {
		formBody := new(LoginRequestBodyData)
		if err := c.BodyParser(formBody); err != nil {
			utils.Log.Println(err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Invalid JSON"})
		}
		username = formBody.Username
		password = formBody.Password
	}

	if err := checkFieldExist("Username", username != "", c); err != nil {
		return err
	}
	if err := checkFieldExist("Password", password != "", c); err != nil {
		return err
	}

	result, err := utils.Login(username, password)
	if err != nil {
		utils.Log.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
	}
	Init()
	return c.JSON(result)
}

// LogoutHandler logs out the user
func LogoutHandler(c *fiber.Ctx) error {
	if !isLogoutDisabled {
		if err := utils.Logout(); err != nil {
			utils.Log.Println(err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"message": "Internal server error"})
		}
		Init()
	}
	return c.Redirect("/", fiber.StatusFound)
}

func LoginRefreshAccessToken() error {
	tokenData, err := utils.GetJIOTVCredentials()
	if err != nil {
		return err
	}

	reqBody := map[string]string{
		"appName":      "RJIL_JioTV",
		"deviceId":     utils.GetDeviceID(),
		"refreshToken": tokenData.RefreshToken,
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(REFRESH_TOKEN_URL)
	req.Header.SetMethod("POST")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accessToken", tokenData.AccessToken)
	req.SetBody(bodyJSON)

	resp := fasthttp.AcquireResponse()
	client := utils.GetRequestClient()
	if err := client.Do(req, resp); err != nil {
		return err
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("AccessToken refresh failed with status code: %d", resp.StatusCode())
	}

	var result RefreshTokenResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return err
	}

	if result.AccessToken != "" {
		tokenData.AccessToken = result.AccessToken
		tokenData.LastTokenRefreshTime = strconv.FormatInt(time.Now().Unix(), 10)
		if err := utils.WriteJIOTVCredentials(tokenData); err != nil {
			return err
		}
		TV = television.New(tokenData)
		go RefreshTokenIfExpired(tokenData)
	}
	return nil
}

func LoginRefreshSSOToken() error {
	tokenData, err := utils.GetJIOTVCredentials()
	if err != nil {
		return err
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(REFRESH_SSO_TOKEN_URL)
	req.Header.SetMethod("GET")
	req.Header.Set("ssoToken", tokenData.SSOToken)
	req.Header.Set("deviceid", utils.GetDeviceID())

	resp := fasthttp.AcquireResponse()
	client := utils.GetRequestClient()
	if err := client.Do(req, resp); err != nil {
		return err
	}
	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("SSOToken refresh failed with status code: %d", resp.StatusCode())
	}

	var result RefreshSSOTokenResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return err
	}

	if result.SSOToken != "" {
		tokenData.SSOToken = result.SSOToken
		tokenData.LastSSOTokenRefreshTime = strconv.FormatInt(time.Now().Unix(), 10)
		if err := utils.WriteJIOTVCredentials(tokenData); err != nil {
			return err
		}
		TV = television.New(tokenData)
		go RefreshSSOTokenIfExpired(tokenData)
	}
	return nil
}

func RefreshTokenIfExpired(creds *utils.JIOTV_CREDENTIALS) error {
	lastRefresh, _ := strconv.ParseInt(creds.LastTokenRefreshTime, 10, 64)
	next := time.Unix(lastRefresh, 0).Add(1*time.Hour + 50*time.Minute)
	if next.Before(time.Now()) {
		return LoginRefreshAccessToken()
	}
	scheduler.Add(REFRESH_TOKEN_TASK_ID, time.Until(next), func() error {
		return RefreshTokenIfExpired(creds)
	})
	return nil
}

func RefreshSSOTokenIfExpired(creds *utils.JIOTV_CREDENTIALS) error {
	lastRefresh, _ := strconv.ParseInt(creds.LastSSOTokenRefreshTime, 10, 64)
	next := time.Unix(lastRefresh, 0).Add(24 * time.Hour)
	if next.Before(time.Now()) {
		return LoginRefreshSSOToken()
	}
	scheduler.Add(REFRESH_SSOTOKEN_TASK_ID, time.Until(next), func() error {
		return RefreshSSOTokenIfExpired(creds)
	})
	return nil
}
